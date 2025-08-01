package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"

	"github.com/threefoldtech/0-fs/meta"
	"github.com/threefoldtech/0-fs/rofs"
	"github.com/threefoldtech/0-fs/storage"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg/app"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/kernel"
	"github.com/threefoldtech/zosbase/pkg/stubs"
	"github.com/threefoldtech/zosbase/pkg/upgrade/hub"
	"github.com/threefoldtech/zosbase/pkg/zinit"

	"github.com/rs/zerolog/log"
)

var (
	// ErrRestartNeeded is returned if upgraded requires a restart
	ErrRestartNeeded = fmt.Errorf("restart needed")

	// services that can't be uninstalled with normal procedure
	protected = []string{"identityd", "redis"}
)

const (
	service = "upgrader"

	defaultZinitSocket = "/var/run/zinit.sock"

	checkForUpdateEvery = 60 * time.Minute
	checkJitter         = 10 // minutes
	defaultHubTimeout   = 20 * time.Second

	ZosRepo    = "tf-zos"
	ZosPackage = "zos.flist"
)

type ChainVersion struct {
	SafeToUpgrade bool   `json:"safe_to_upgrade"`
	Version       string `json:"version"`
	VersionLight  string `json:"version_light"`
}

func getRolloutConfig(ctx context.Context, gw *stubs.SubstrateGatewayStub) (ChainVersion, []uint32, error) {
	config, err := environment.GetConfig()
	if err != nil {
		return ChainVersion{}, nil, errors.Wrap(err, "failed to get network config")
	}

	v, err := gw.GetZosVersion(ctx)
	if err != nil {
		return ChainVersion{}, nil, errors.Wrap(err, "failed to get zos version from chain")
	}

	var chainVersion ChainVersion
	err = json.Unmarshal([]byte(v), &chainVersion)
	if err != nil {
		log.Debug().Err(err).Msg("failed to unmarshal chain version and safe to upgrade flag")
		chainVersion = ChainVersion{
			SafeToUpgrade: true,
			Version:       v,
			VersionLight:  "",
		}
	}

	return chainVersion, config.RolloutUpgrade.TestFarms, nil
}

// Upgrader is the component that is responsible
// to keep 0-OS up to date
type Upgrader struct {
	boot         Boot
	zinit        *zinit.Client
	zcl          zbus.Client
	root         string
	noZosUpgrade bool
	hub          *hub.HubClient
	storage      storage.Storage
}

// UpgraderOption interface
type UpgraderOption func(u *Upgrader) error

// NoZosUpgrade option, enable or disable
// the update of zos binaries.
// enabled by default
func NoZosUpgrade(o bool) UpgraderOption {
	return func(u *Upgrader) error {
		u.noZosUpgrade = o

		return nil
	}
}

// ZbusClient option, adds a zbus client to the upgrader
func ZbusClient(cl zbus.Client) UpgraderOption {
	return func(u *Upgrader) error {
		u.zcl = cl

		return nil
	}
}

// Storage option overrides the default hub storage url
// default value is hub.grid.tf
func Storage(url string) UpgraderOption {
	return func(u *Upgrader) error {
		storage, err := storage.NewSimpleStorage(url)
		if err != nil {
			return errors.Wrap(err, "failed to initialize hub storage")
		}
		u.storage = storage
		return nil
	}
}

// Zinit option overrides the default zinit socket
func Zinit(socket string) UpgraderOption {
	return func(u *Upgrader) error {
		zinit := zinit.New(socket)
		u.zinit = zinit
		return nil
	}
}

// NewUpgrader creates a new upgrader instance
func NewUpgrader(root string, opts ...UpgraderOption) (*Upgrader, error) {
	hubClient := hub.NewHubClient(defaultHubTimeout)
	u := &Upgrader{
		root: root,
		hub:  hubClient,
	}

	for _, dir := range []string{u.fileCache(), u.flistCache()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrap(err, "failed to prepare cache directories")
		}
	}

	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, err
		}
	}
	env := environment.MustGet()
	hubStorage := env.HubStorage
	// if kernel.GetParams().IsV4() {
	// 	hub_storage = defaultHubv4Storage
	// }
	if u.storage == nil {
		// no storage option was set. use default
		if err := Storage(hubStorage)(u); err != nil {
			return nil, err
		}
	}

	if u.zinit == nil {
		if err := Zinit(defaultZinitSocket)(u); err != nil {
			return nil, err
		}
	}

	return u, nil
}

// Run starts the upgrader module and run the update according to the detected boot method
func (u *Upgrader) Run(ctx context.Context) error {
	method := u.boot.DetectBootMethod()
	if method == BootMethodOther {
		// we need to do an update one time to fetch all
		// binaries required by the system except for the zos
		// binaries
		// then we should block forever
		log.Info().Msg("system is not booted from the hub")
		if app.IsFirstBoot(service) {
			remote, err := u.remote()
			if err != nil {
				return errors.Wrap(err, "failed to get remote tag")
			}

			if err := u.updateTo(remote, nil); err != nil {
				return errors.Wrap(err, "failed to run update")
			}
		}
		// to avoid redoing the binary installation
		// when service is restarted
		if err := app.MarkBooted(service); err != nil {
			return errors.Wrap(err, "failed to mark system as booted")
		}

		log.Info().Msg("update is disabled")
		<-ctx.Done()
		return nil
	}

	// if the booting method is bootstrap then we run update periodically
	// after u.nextUpdate to make sure all the modules are always up to date
	for {
		err := u.update(ctx)
		if errors.Is(err, ErrRestartNeeded) {
			return err
		} else if err != nil {
			log.Error().Err(err).Msg("failed while checking for updates")
			<-time.After(10 * time.Second)
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(u.nextUpdate()):
		}

	}
}

func (u *Upgrader) Version() semver.Version {
	return u.boot.Version()
}

// nextUpdate returns the interval until the next update
// which is approximately 60 minutes + jitter interval(0-10 minutes)
// to make sure not all nodes run upgrader at the same time
func (u *Upgrader) nextUpdate() time.Duration {
	jitter := rand.Intn(checkJitter)
	next := checkForUpdateEvery + (time.Duration(jitter) * time.Minute)
	log.Info().Str("after", next.String()).Msg("checking for update")
	return next
}

// remote finds the `tag link` associated with the node network (for example devnet)
func (u *Upgrader) remote() (remote hub.TagLink, err error) {
	mode := u.boot.RunMode()
	// find all taglinks that matches the same run mode (ex: development)
	matchName := mode.String()
	if kernel.GetParams().IsLight() {
		matchName = fmt.Sprintf("%s-%s", mode.String(), kernel.GetParams().GetVersion())
	}
	matches, err := u.hub.Find(
		ZosRepo,
		hub.MatchName(matchName),
		hub.MatchType(hub.TypeTagLink),
	)
	if err != nil {
		return remote, err
	}

	if len(matches) != 1 {
		return remote, fmt.Errorf("can't find taglink that matches '%s'", matchName)
	}

	return hub.NewTagLink(matches[0]), nil
}

func (u *Upgrader) update(ctx context.Context) error {
	// here we need to do a normal full update cycle
	current, err := u.boot.Current()
	if err != nil {
		log.Error().Err(err).Msg("failed to get info about current version, update anyway")
	}

	remote, err := u.remote()
	if err != nil {
		return errors.Wrap(err, "failed to get remote tag")
	}

	// obviously a remote tag need to match the current tag.
	// if the remote is different, we actually run the update and exit.
	if remote.Target == current.Target {
		// nothing to do!
		return nil
	}

	env := environment.MustGet()
	gw := stubs.NewSubstrateGatewayStub(u.zcl)
	chainVer, testFarms, err := getRolloutConfig(ctx, gw)
	if err != nil {
		return errors.Wrap(err, "failed to get rollout config and version")
	}

	remoteVer := remote.Target[strings.LastIndex(remote.Target, "/")+1:]
	if kernel.GetParams().IsLight() {
		if env.RunningMode != environment.RunningDev && (remoteVer != chainVer.VersionLight) {
			// nothing to do! hub version is not the same as the chain
			return nil
		}
	} else {
		if env.RunningMode != environment.RunningDev && (remoteVer != chainVer.Version) {
			// nothing to do! hub version is not the same as the chain
			return nil
		}
	}

	if !chainVer.SafeToUpgrade {
		if !slices.Contains(testFarms, uint32(env.FarmID)) {
			// nothing to do! waiting for the flag `safe to upgrade to be enabled after A/B testing`
			// node is not a part of A/B testing
			return nil
		}
	}

	log.Info().Str("running version", u.Version().String()).Str("updating to version", filepath.Base(remote.Target)).Msg("updating system...")
	if err := u.updateTo(remote, &current); err != nil {
		return errors.Wrapf(err, "failed to update to new tag '%s'", remote.Target)
	}

	if err := u.boot.Set(remote); err != nil {
		return err
	}

	return ErrRestartNeeded
}

// updateTo updates flist packages to match "link"
// and only update zos package if u.noZosUpgrade is set to false
func (u *Upgrader) updateTo(link hub.TagLink, current *hub.TagLink) error {
	repo, tag, err := link.Destination()
	if err != nil {
		return errors.Wrap(err, "failed to get destination tag")
	}

	packages, err := u.hub.ListTag(repo, tag)
	if err != nil {
		return errors.Wrapf(err, "failed to list tag '%s' packages", tag)
	}

	var curPkgsNames []string
	if current != nil {
		// get current pkgs list to compare the new pkgs against it
		curRepo, curTag, err := current.Destination()
		if err != nil {
			return errors.Wrap(err, "failed to resolve current link")
		}
		curPkgs, err := u.hub.ListTag(curRepo, curTag)
		if err != nil {
			return errors.Wrapf(err, "failed to list tag %s", curTag)
		}
		// store curPkgs names, the only part needed for the comparison
		for _, pkg := range curPkgs {
			_, name, err := pkg.Destination(curRepo)
			if err == nil {
				curPkgsNames = append(curPkgsNames, name)
			}
		}
	}

	var later [][]string
	for _, pkg := range packages {
		pkgRepo, name, err := pkg.Destination(repo)
		// if the new pkg is the same as the current pkg no need to reinstall it
		if slices.Contains(curPkgsNames, name) {
			log.Info().Str("package", name).Msg("skipping package")
			continue
		}
		if pkg.Name == ZosPackage {
			// this is the last to do to make sure all dependencies are installed before updating zos
			log.Debug().Str("repo", pkgRepo).Str("name", name).Msg("schedule package for later")
			later = append(later, []string{pkgRepo, name})
			continue
		}

		if err != nil {
			return errors.Wrapf(err, "failed to find target for package '%s'", pkg.Target)
		}

		// install package
		if err := u.install(pkgRepo, name); err != nil {
			return errors.Wrapf(err, "failed to install package %s/%s", pkgRepo, name)
		}
	}

	if u.noZosUpgrade {
		return nil
	}

	// probably check flag for zos installation
	for _, pkg := range later {
		repo, name := pkg[0], pkg[1]
		if err := u.install(repo, name); err != nil {
			return errors.Wrapf(err, "failed to install package %s/%s", repo, name)
		}
	}

	return nil
}

func (u *Upgrader) flistCache() string {
	return filepath.Join(u.root, "cache", "flist")
}

func (u *Upgrader) fileCache() string {
	return filepath.Join(u.root, "cache", "files")
}

// getFlist accepts fqdn of flist as `<repo>/<name>.flist`
func (u *Upgrader) getFlist(repo, name string, cache cache) (meta.Walker, error) {
	db, err := u.hub.Download(cache.flistCache(), repo, name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download flist")
	}

	store, err := meta.NewStore(db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load flist db")
	}

	walker, ok := store.(meta.Walker)
	if !ok {
		store.Close()
		return nil, errors.Wrap(err, "flist database of unsupported type")
	}

	return walker, nil
}

type cache interface {
	flistCache() string
	fileCache() string
}

type inMemoryCache struct {
	file  string
	flist string
	root  string
}

func newInMemoryCache() (*inMemoryCache, error) {
	root := filepath.Join("/tmp", service)
	file := filepath.Join(root, "cache", "file")
	flist := filepath.Join(root, "cache", "flist")
	if err := os.MkdirAll(file, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(flist, 0755); err != nil {
		return nil, err
	}
	return &inMemoryCache{file: file, flist: flist, root: root}, nil
}

func (c *inMemoryCache) flistCache() string {
	return c.flist
}

func (c *inMemoryCache) fileCache() string {
	return c.file
}

func (c *inMemoryCache) clean() {
	os.RemoveAll(c.root)
}

// install from a single flist.
func (u *Upgrader) install(repo, name string) error {
	log.Info().Str("repo", repo).Str("name", name).Msg("start installing package")
	var cache cache = u
	store, err := u.getFlist(repo, name, cache)

	if errors.Is(err, syscall.EROFS) ||
		errors.Is(err, syscall.EPERM) ||
		errors.Is(err, syscall.EIO) {
		// try in memory
		inMemoryCache, err := newInMemoryCache()
		if err != nil {
			return fmt.Errorf("failed to create in memory cache: %w", err)
		}
		defer inMemoryCache.clean()
		cache = inMemoryCache

		log.Info().Msg("downloading in memory")
		store, err = u.getFlist(repo, name, cache)
		if err != nil {
			return errors.Wrapf(err, "failed to process flist: %s/%s", repo, name)
		}
	} else if err != nil {
		return errors.Wrapf(err, "failed to process flist: %s/%s", repo, name)
	}
	defer store.Close()

	if err := safe(func() error {
		// copy is done in a safe closer to avoid interrupting
		// the installation
		return u.copyRecursive(store, "/", cache)
	}); err != nil {
		return errors.Wrapf(err, "failed to install flist: %s/%s", repo, name)
	}

	services, err := u.servicesFromStore(store)
	if err != nil {
		return errors.Wrap(err, "failed to list services from flist")
	}

	if err := u.ensureRestarted(services...); err != nil {
		return err
	}

	// restarting mycelium instances on user's namespaces
	return u.restartMyceliumInstances()
}

// this method restarts all mycelium-<usernetwork> instances on user's namespaces to catch mycelium version updates
func (u *Upgrader) restartMyceliumInstances() error {
	const zinitPath = "/etc/zinit"

	// Get all services from host
	entries, err := os.ReadDir(zinitPath)
	if err != nil {
		return fmt.Errorf("failed to read host zinit directory: %w", err)
	}

	var myceliumServices []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") || !strings.HasPrefix(entry.Name(), "mycelium-") {
			continue
		}

		serviceName := strings.TrimSuffix(entry.Name(), ".yaml")
		myceliumServices = append(myceliumServices, serviceName)
	}

	if len(myceliumServices) == 0 {
		return nil
	}

	log.Info().Strs("services", myceliumServices).Msg("restarting mycelium instances")
	if err := u.zinit.StopMultiple(20*time.Second, myceliumServices...); err != nil {
		log.Error().Err(err).Msg("failed to stop all mycelium services")
	}

	for _, name := range myceliumServices {
		log.Info().Str("service", name).Msg("starting mycelium service")
		if err := u.zinit.Start(name); err != nil {
			log.Error().Err(err).Str("service", name).Msg("could not start mycelium service")
		}
	}

	return nil
}

func (u *Upgrader) servicesFromStore(store meta.Walker) ([]string, error) {
	const zinitPath = "/etc/zinit"

	var services []string
	err := store.Walk(zinitPath, func(path string, info meta.Meta) error {
		if info.IsDir() {
			return nil
		}
		dir := filepath.Dir(path)
		if dir != zinitPath {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}

		services = append(services,
			strings.TrimSuffix(info.Name(), ".yaml"))
		return nil
	})

	if err == meta.ErrNotFound {
		return nil, nil
	}

	return services, err
}

func (u *Upgrader) ensureRestarted(service ...string) error {
	// remove protected function from list, these never restarted
	service = slices.DeleteFunc(service, func(e string) bool {
		return slices.Contains(protected, e)
	})

	log.Debug().Strs("services", service).Msg("ensure services")
	if len(service) == 0 {
		return nil
	}

	log.Debug().Strs("services", service).Msg("restarting services")
	if err := u.zinit.StopMultiple(20*time.Second, service...); err != nil {
		// we log here so we don't leave the node in a bad state!
		// by just trying to start as much services as we can
		log.Error().Err(err).Msg("failed to stop all services")
	}

	for _, name := range service {
		log.Info().Str("service", name).Msg("starting service")
		if err := u.zinit.Forget(name); err != nil {
			log.Warn().Err(err).Str("service", name).Msg("could not forget service")
		}

		if err := u.zinit.Monitor(name); err != nil && err != zinit.ErrAlreadyMonitored {
			log.Error().Err(err).Str("service", name).Msg("could not monitor service")
		}

		// this has no effect if Monitor already worked with no issue
		// but we do it anyway for services that could not be forgotten (did not stop)
		// so we start them again
		if err := u.zinit.Start(name); err != nil {
			log.Error().Err(err).Str("service", name).Msg("could not start service")
		}
	}

	return nil
}

func (u *Upgrader) copyRecursive(store meta.Walker, destination string, cache cache, skip ...string) error {
	return store.Walk("", func(path string, info meta.Meta) error {
		dest := filepath.Join(destination, path)
		if isIn(dest, skip) {
			if info.IsDir() {
				return meta.ErrSkipDir
			}
			log.Debug().Str("file", dest).Msg("skipping file")
			return nil
		}

		if info.IsDir() {
			if err := os.MkdirAll(dest, os.FileMode(info.Info().Access.Mode)); err != nil {
				return err
			}
			return nil
		}

		stat := info.Info()

		switch stat.Type {
		case meta.RegularType:
			// regular file (or other types that we don't handle)
			return u.copyFile(dest, info, cache)
		case meta.LinkType:
			// fmt.Println("link target", stat.LinkTarget)
			target := stat.LinkTarget
			if filepath.IsAbs(target) {
				// if target is absolute, we make sure it's under destination
				// other wise use relative path
				target = filepath.Join(destination, stat.LinkTarget)
			}

			if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
				return err
			}

			return os.Symlink(target, dest)
		default:
			log.Debug().Str("type", info.Info().Type.String()).Msg("ignoring not suppored file type")
		}

		return nil
	})
}

func isIn(target string, list []string) bool {
	for _, x := range list {
		if target == x {
			return true
		}
	}
	return false
}

func (u *Upgrader) copyFile(dst string, src meta.Meta, cache cache) error {
	log.Info().Str("source", src.Name()).Str("destination", dst).Msg("copy file")

	var (
		isNew  = false
		dstOld string
	)

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		// case where this is a new file
		// we just need to copy from flist to root
		isNew = true
	}

	var err error
	if !isNew {
		dstOld = dst + ".old"
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}

		defer func() {
			if err == nil {
				if err := os.Remove(dstOld); err != nil {
					log.Error().Err(err).Str("file", dstOld).Msg("failed to clean up backup file")
				}
				return
			}

			if err := os.Rename(dstOld, dst); err != nil {
				log.Error().Err(err).Str("file", dst).Msg("failed to restore file after a failed download")
			}
		}()
	}

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, os.FileMode(src.Info().Access.Mode))
	if err != nil {
		return err
	}
	defer fDst.Close()

	fsCache := rofs.NewCache(cache.fileCache(), u.storage)
	fSrc, err := fsCache.CheckAndGet(src)
	if err != nil {
		return err
	}

	if _, err := io.Copy(fDst, fSrc); err != nil {
		return err
	}

	return nil
}

// safe makes sure function call not interrupted
// with a signal while execution
func safe(fn func() error) error {
	ch := make(chan os.Signal, 4)
	defer close(ch)
	defer signal.Stop(ch)

	// try to upgraded to latest
	// but mean while also make sure the daemon can not be killed by a signal
	signal.Notify(ch)
	return fn()
}
