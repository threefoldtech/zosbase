package flist

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"text/template"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

var templ = template.Must(template.New("script").Parse(`
echo "mount ready" > {{.log}}
`))

// StorageMock is a mock object of the storage modules
type StorageMock struct {
	mock.Mock
}

// CreateFilesystem create filesystem mock
func (s *StorageMock) VolumeCreate(ctx context.Context, name string, size gridtypes.Unit) (pkg.Volume, error) {
	args := s.Called(ctx, name, size)
	return pkg.Volume{
		Path: args.String(0),
	}, args.Error(1)
}

// UpdateFilesystem update filesystem mock
func (s *StorageMock) VolumeUpdate(ctx context.Context, name string, size gridtypes.Unit) error {
	args := s.Called(ctx, name)
	return args.Error(0)
}

// ReleaseFilesystem releases filesystem mock
func (s *StorageMock) VolumeDelete(ctx context.Context, name string) error {
	args := s.Called(ctx, name)
	return args.Error(0)
}

// Path implements the pkg.StorageModules interfaces
func (s *StorageMock) VolumeLookup(ctx context.Context, name string) (pkg.Volume, error) {
	args := s.Called(ctx, name)
	return pkg.Volume{
		Path: args.String(0),
	}, args.Error(1)
}

type testCommander struct {
	*testing.T
	m map[string]string
}

func (t *testCommander) args(args ...string) (map[string]string, []string) {
	var lk string
	v := make(map[string]string)
	var r []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			lk = strings.TrimPrefix(arg, "--")
			v[lk] = ""
			continue
		}

		// if no -, then it must be a value.
		// so if lk is set, update, otherwise append to r
		if len(lk) > 0 {
			v[lk] = arg
			lk = ""
		} else {
			r = append(r, arg)
		}
	}

	t.m = v
	return v, r
}

func (t *testCommander) Command(name string, args ...string) *exec.Cmd {
	if name == "mountpoint" {
		return exec.Command("true")
	} else if name == "findmnt" {
		return exec.Command("echo", "{}")
	} else if name != "g8ufs" {
		t.Fatalf("invalid command name, expected 'g8ufs' got '%s'", name)
	}

	m, _ := t.args(args...)
	var script bytes.Buffer
	err := templ.Execute(&script, map[string]string{
		"log": m["log"],
	})

	require.NoError(t.T, err)

	return exec.Command("sh", "-c", script.String())
}

func (t *testCommander) GetNamespace(name string) (ns.NetNS, error) {
	return nil, nil
}

type testSystem struct {
	mock.Mock
}

func (t *testSystem) Mount(source string, target string, fstype string, flags uintptr, data string) (err error) {
	args := t.Called(source, target, fstype, flags, data)
	return args.Error(0)
}

func (t *testSystem) Unmount(target string, flags int) error {
	args := t.Called(target, flags)
	return args.Error(0)
}

func TestCommander(t *testing.T) {
	cmder := testCommander{T: t}

	m, r := cmder.args("--log", "log-file", "remaining")

	require.Equal(t, []string{"remaining"}, r)
	require.Equal(t, map[string]string{
		"log": "log-file",
	}, m)
}

/**
* TestMount tests that the module actually call the underlying modules an binaries
* with the correct arguments. It's mocked in a way that does not actually mount
* an flist, but emulate the call of the g8ufs binary.
* The method also validate that storage module is called as expected
 */
func TestMountUnmount(t *testing.T) {
	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root := t.TempDir()

	sys := &testSystem{}
	flister := newFlister(root, strg, cmder, sys)

	backend := t.TempDir()

	strg.On("VolumeLookup", mock.Anything, mock.Anything).Return(backend, nil)

	name := "test"
	strg.On("VolumeCreate", name, uint64(256*mib)).
		Return(backend, nil)

	sys.On("Mount", "overlay", filepath.Join(root, "mountpoint", name), "overlay", uintptr(syscall.MS_NOATIME), mock.Anything).Return(nil)

	redisFlist, err := url.JoinPath(environment.MustGet().HubURL, "thabet", "redis.flist")
	require.NoError(t, err)

	mnt, err := flister.mountInNamespace(name, redisFlist, pkg.DefaultMountOptions, "")
	require.NoError(t, err)

	// Trick flister into thinking that 0-fs has exited
	os.Remove(cmder.m["pid"])
	strg.On("VolumeDelete", mock.Anything, filepath.Base(mnt)).Return(nil)

	sys.On("Unmount", mnt, 0).Return(nil)

	err = flister.Unmount(name)
	require.NoError(t, err)
}

func TestMountUnmountRO(t *testing.T) {
	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root := t.TempDir()

	sys := &testSystem{}
	flister := newFlister(root, strg, cmder, sys)

	name := "test"

	flist := mock.Anything
	sys.On("Mount", flist, filepath.Join(root, "mountpoint", name), "bind", uintptr(syscall.MS_BIND), "").Return(nil)

	redisFlist, err := url.JoinPath(environment.MustGet().HubURL, "thabet", "redis.flist")
	require.NoError(t, err)

	mnt, err := flister.mountInNamespace(name, redisFlist, pkg.ReadOnlyMountOptions, "")
	require.NoError(t, err)

	// Trick flister into thinking that 0-fs has exited
	os.Remove(cmder.m["pid"])
	strg.On("VolumeDelete", mock.Anything, filepath.Base(mnt)).Return(nil)

	sys.On("Unmount", mnt, 0).Return(nil)

	err = flister.Unmount(name)
	require.NoError(t, err)
}

func TestIsolation(t *testing.T) {
	require := require.New(t)

	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root := t.TempDir()

	sys := &testSystem{}

	flister := newFlister(root, strg, cmder, sys)

	backend := t.TempDir()
	strg.On("VolumeLookup", mock.Anything, mock.Anything).Return(backend, nil)

	strg.On("VolumeCreate", mock.Anything, mock.Anything, mock.Anything, uint64(256*mib)).
		Return(backend, nil)

	name1 := "test1"
	sys.On("Mount", "overlay", filepath.Join(root, "mountpoint", name1), "overlay", uintptr(syscall.MS_NOATIME), mock.Anything).Return(nil)

	name2 := "test2"
	sys.On("Mount", "overlay", filepath.Join(root, "mountpoint", name2), "overlay", uintptr(syscall.MS_NOATIME), mock.Anything).Return(nil)

	redisFlist, err := url.JoinPath(environment.MustGet().HubURL, "thabet", "redis.flist")
	require.NoError(err)

	path1, err := flister.mountInNamespace(name1, redisFlist, pkg.DefaultMountOptions, "")
	require.NoError(err)
	args1 := cmder.m

	path2, err := flister.mountInNamespace(name2, redisFlist, pkg.DefaultMountOptions, "")
	require.NoError(err)
	args2 := cmder.m

	require.NotEqual(path1, path2)
	require.Equal(args1, args2)
}

func TestDownloadFlist(t *testing.T) {
	require := require.New(t)
	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root := t.TempDir()

	sys := &testSystem{}

	f := newFlister(root, strg, cmder, sys)

	redisFlist, err := url.JoinPath(environment.MustGet().HubURL, "thabet", "redis.flist")
	require.NoError(err)

	hash1, path1, err := f.downloadFlist(redisFlist, "")
	require.NoError(err)

	// now corrupt the flist
	err = os.Truncate(string(path1), 512)
	require.NoError(err)

	hash2, path2, err := f.downloadFlist(redisFlist, "")
	require.NoError(err)

	require.EqualValues(path1, path2)
	require.EqualValues(hash1, hash2)
}
