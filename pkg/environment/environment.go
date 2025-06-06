package environment

import (
	"os"
	"slices"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zosbase/pkg"

	"github.com/threefoldtech/zosbase/pkg/kernel"
)

const (
	baseExtendedURL = "https://raw.githubusercontent.com/threefoldtech/zos-config/main/"
)

// PubMac specify how the mac address of the public nic
// (in case of public-config) is calculated
type PubMac string

const (
	// PubMacRandom means the mac of the public nic will be chosen by the system
	// the value won't change across reboots, but is based on the node id
	// (default)
	PubMacRandom PubMac = "random"
	// PubMacSwap means the value of the mac is swapped with the physical nic
	// where the public traffic is eventually going through
	PubMacSwap PubMac = "swap"
)

// Environment holds information about running environment of a node
// it defines the different constant based on the running mode (dev, test, prod)
type Environment struct {
	RunningMode RunMode

	FlistURL string
	BinRepo  string

	FarmID pkg.FarmID
	Orphan bool

	FarmSecret   string
	SubstrateURL []string
	// IMPORTANT NOTICE:
	//   SINCE RELAYS FOR A NODE IS STORED ON THE CHAIN IN A LIMITED SPACE
	//   PLEASE MAKE SURE THAT ANY ENV HAS NO MORE THAN FOUR RELAYS CONFIGURED
	relaysURLs    []string
	ActivationURL []string
	GraphQL       []string
	KycURL        string
	RegistrarURL  string

	// private vlan to join
	// if set, zos will use this as its priv vlan
	PrivVlan *uint16

	// pub vlan to join
	// if set, zos will use this as it's pub vlan
	// only in a single nic setup
	PubVlan *uint16

	// PubMac value from environment
	PubMac PubMac
}

// RunMode type
type RunMode string

func (r RunMode) String() string {
	switch r {
	case RunningDev:
		return "development"
	case RunningQA:
		return "qa"
	case RunningMain:
		return "production"
	case RunningTest:
		return "testing"
	}

	return "unknown"
}

// Possible running mode of a node
const (
	// RunningDev mode
	RunningDev RunMode = "dev"
	// RunningQA mode
	RunningQA RunMode = "qa"
	// RunningTest mode
	RunningTest RunMode = "test"
	// RunningMain mode
	RunningMain RunMode = "prod"

	// Orphanage is the default farmid where nodes are registered
	// if no farmid were specified on the kernel command line
	OrphanageDev  pkg.FarmID = 0
	OrphanageTest pkg.FarmID = 0
	OrphanageMain pkg.FarmID = 0
)

var (
	pool     substrate.Manager
	poolOnce sync.Once

	envDev = Environment{
		RunningMode: RunningDev,
		SubstrateURL: []string{
			"wss://tfchain.dev.grid.tf/",
			"wss://tfchain.02.dev.grid.tf",
		},
		relaysURLs: []string{
			"wss://relay.dev.grid.tf",
			"wss://relay.02.dev.grid.tf",
		},
		ActivationURL: []string{
			"https://activation.dev.grid.tf/activation/activate",
			"https://activation.02.dev.grid.tf/activation/activate",
		},
		FlistURL: "redis://hub.grid.tf:9900",
		BinRepo:  "tf-zos-v3-bins.dev",
		GraphQL: []string{
			"https://graphql.dev.grid.tf/graphql",
			"https://graphql.02.dev.grid.tf/graphql",
		},
		KycURL:       "https://kyc.dev.grid.tf",
		RegistrarURL: "http://registrar.dev4.grid.tf",
	}

	envTest = Environment{
		RunningMode: RunningTest,
		SubstrateURL: []string{
			"wss://tfchain.test.grid.tf/",
			"wss://tfchain.02.test.grid.tf",
		},
		relaysURLs: []string{
			"wss://relay.test.grid.tf",
			"wss://relay.02.test.grid.tf",
		},
		ActivationURL: []string{
			"https://activation.test.grid.tf/activation/activate",
			"https://activation.02.test.grid.tf/activation/activate",
		},
		FlistURL: "redis://hub.grid.tf:9900",
		BinRepo:  "tf-zos-v3-bins.test",
		GraphQL: []string{
			"https://graphql.test.grid.tf/graphql",
			"https://graphql.02.test.grid.tf/graphql",
		},
		KycURL:       "https://kyc.test.grid.tf",
		RegistrarURL: "http://registrar.test4.grid.tf",
	}

	envQA = Environment{
		RunningMode: RunningQA,
		SubstrateURL: []string{
			"wss://tfchain.qa.grid.tf/",
			"wss://tfchain.02.qa.grid.tf/",
		},
		relaysURLs: []string{
			"wss://relay.qa.grid.tf",
			"wss://relay.02.qa.grid.tf",
		},
		ActivationURL: []string{
			"https://activation.qa.grid.tf/activation/activate",
			"https://activation.02.qa.grid.tf/activation/activate",
		},
		FlistURL: "redis://hub.grid.tf:9900",
		BinRepo:  "tf-zos-v3-bins.qanet",
		GraphQL: []string{
			"https://graphql.qa.grid.tf/graphql",
			"https://graphql.02.qa.grid.tf/graphql",
		},
		KycURL:       "https://kyc.qa.grid.tf",
		RegistrarURL: "https://registrar.qa4.grid.tf",
	}

	envProd = Environment{
		RunningMode: RunningMain,
		SubstrateURL: []string{
			"wss://tfchain.grid.tf/",
			"wss://tfchain.02.grid.tf",
			"wss://02.tfchain.grid.tf/",
			"wss://03.tfchain.grid.tf/",
			"wss://04.tfchain.grid.tf/",
		},
		relaysURLs: []string{
			"wss://relay.grid.tf",
			// "wss://relay.02.grid.tf",
		},
		ActivationURL: []string{
			"https://activation.grid.tf/activation/activate",
			"https://activation.02.grid.tf/activation/activate",
		},
		FlistURL: "redis://hub.grid.tf:9900",
		BinRepo:  "tf-zos-v3-bins",
		GraphQL: []string{
			"https://graphql.grid.tf/graphql",
			"https://graphql.02.grid.tf/graphql",
		},
		KycURL:       "https://kyc.grid.tf",
		RegistrarURL: "https://registrar.prod4.grid.tf",
	}
)

// MustGet returns the running environment of the node
// panics on error
func MustGet() Environment {
	env, err := Get()
	if err != nil {
		panic(err)
	}

	return env
}

// Get return the running environment of the node
func Get() (Environment, error) {
	params := kernel.GetParams()
	env, err := getEnvironmentFromParams(params)
	if err != nil {
		return Environment{}, err
	}

	return env, nil
}

func GetRelaysURLs() []string {
	config, err := GetConfig()
	if err == nil && len(config.RelaysURLs) > 0 {
		log.Debug().Msg("using relays urls from zos-config")
		return config.RelaysURLs
	}

	log.Debug().Msg("using relays urls from environment")
	env := MustGet()
	return env.relaysURLs
}

// GetSubstrate gets a client to subsrate blockchain
func GetSubstrate() (substrate.Manager, error) {
	env, err := Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get boot environment")
	}

	poolOnce.Do(func() {
		pool = substrate.NewManager(env.SubstrateURL...)
	})

	return pool, nil
}

func getEnvironmentFromParams(params kernel.Params) (Environment, error) {
	var env Environment
	runmode := ""
	if modes, ok := params.Get("runmode"); ok {
		if len(modes) >= 1 {
			runmode = modes[0]
		}
	} else {
		runmode = os.Getenv("ZOS_RUNMODE")
	}

	if len(runmode) == 0 {
		runmode = string(RunningMain)
	}

	switch RunMode(runmode) {
	case RunningDev:
		env = envDev
	case RunningQA:
		env = envQA
	case RunningTest:
		env = envTest
	case RunningMain:
		env = envProd
	default:
		env = envProd
	}

	if substrate, ok := params.Get("substrate"); ok {
		if len(substrate) > 0 {
			env.SubstrateURL = substrate
		}
	}

	if relay, ok := params.Get("relay"); ok {
		if len(relay) > 0 {
			env.relaysURLs = relay
		}
	}

	if activation, ok := params.Get("activation"); ok {
		if len(activation) > 0 {
			env.ActivationURL = activation
		}
	}

	if farmSecret, ok := params.Get("secret"); ok {
		if len(farmSecret) > 0 {
			env.FarmSecret = farmSecret[len(farmSecret)-1]
		}
	}

	farmerID, found := params.Get("farmer_id")
	if !found || len(farmerID) < 1 || farmerID[0] == "" {
		// fmt.Println("Warning: no valid farmer_id found in kernel parameter, fallback to orphanage")
		env.Orphan = true

		switch env.RunningMode {
		case RunningDev:
			env.FarmID = OrphanageDev
		case RunningTest:
			env.FarmID = OrphanageTest
		case RunningMain:
			env.FarmID = OrphanageMain
		}

	} else {
		env.Orphan = false
		id, err := strconv.ParseUint(farmerID[0], 10, 32)
		if err != nil {
			return env, errors.Wrap(err, "wrong format for farm ID")
		}
		env.FarmID = pkg.FarmID(id)
	}

	if vlan, found := params.GetOne("vlan:priv"); found {
		if !slices.Contains([]string{"none", "untagged", "un"}, vlan) {
			tag, err := strconv.ParseUint(vlan, 10, 16)
			if err != nil {
				return env, errors.Wrap(err, "failed to parse priv vlan value")
			}
			tagU16 := uint16(tag)
			env.PrivVlan = &tagU16
		}
	}

	if vlan, found := params.GetOne("vlan:pub"); found {
		if !slices.Contains([]string{"none", "untagged", "un"}, vlan) {
			tag, err := strconv.ParseUint(vlan, 10, 16)
			if err != nil {
				return env, errors.Wrap(err, "failed to parse pub vlan value")
			}
			tagU16 := uint16(tag)
			env.PubVlan = &tagU16
		}
	}

	if mac, found := params.GetOne("pub:mac"); found {
		v := PubMac(mac)
		if slices.Contains([]PubMac{PubMacRandom, PubMacSwap}, v) {
			env.PubMac = v
		} else {
			env.PubMac = PubMacRandom
		}
	} else {
		env.PubMac = PubMacRandom
	}

	// Checking if there environment variable
	// override default settings

	if e := os.Getenv("ZOS_SUBSTRATE_URL"); e != "" {
		env.SubstrateURL = []string{e}
	}

	if e := os.Getenv("ZOS_FLIST_URL"); e != "" {
		env.FlistURL = e
	}

	if e := os.Getenv("ZOS_BIN_REPO"); e != "" {
		env.BinRepo = e
	}

	// if the node running v4 chage flisturl to use v4.hub.grid.tf
	if params.IsV4() {
		env.FlistURL = "redis://v4.hub.grid.tf:9940"
	}

	return env, nil
}
