package gridtypes

import substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"

type Node struct {
	NodeID     uint64    `json:"node_id"`
	FarmID     uint64    `json:"farm_id"`
	TwinID     uint64    `json:"twin_id"`
	ZosVersion string    `json:"zos_version"`
	NodeType   string    `json:"node_type"`
	Location   Location  `json:"location"`
	Resources  Resources `json:"resources"`
	Status     string    `json:"Status"`

	Interface   Interface         `json:"interface"`
	SecureBoot  bool              `json:"secure_boot"`
	Virtualized bool              `json:"virtualized"`
	BoardSerial OptionBoardSerial `json:"board_serial"`
}

type Resources struct {
	HRU uint64 `json:"hru"`
	SRU uint64 `json:"sru"`
	CRU uint64 `json:"cru"`
	MRU uint64 `json:"mru"`
}

type Location struct {
	City      string `json:"city"`
	Country   string `json:"country"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type Interface struct {
	Name string   `json:"name"`
	Mac  string   `json:"mac"`
	IPs  []string `json:"ips"`
}

type OptionBoardSerial struct {
	HasValue bool   `json:"has_value"`
	AsValue  string `json:"as_value"`
}
type Identity substrate.Identity
