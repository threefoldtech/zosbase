package gridtypes

import (
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// Node type
type Node struct {
	// Version       string    `json:"version"`
	Version       Versioned
	NodeID        uint64    `json:"node_id"`
	FarmID        uint64    `json:"farm_id"`
	TwinID        uint64    `json:"twin_id"`
	ZosVersion    string    `json:"zos_version"`
	NodeType      string    `json:"node_type"`
	Location      Location  `json:"location"`
	Resources     Resources `json:"resources"`
	Status        string    `json:"Status"`
	FarmingPolicy uint64    `json:"farming_policy"`

	Interface   Interface         `json:"interface"`
	SecureBoot  bool              `json:"secure_boot"`
	Virtualized bool              `json:"virtualized"`
	BoardSerial OptionBoardSerial `json:"board_serial"`
}

type Farm struct {
	FarmID      uint64 `gorm:"primaryKey;autoIncrement" json:"farm_id"`
	FarmName    string `gorm:"size:40;not null;unique;check:farm_name <> ''" json:"farm_name"`
	TwinID      uint64 `json:"twin_id" gorm:"not null;check:twin_id > 0"`
	Dedicated   bool   `json:"dedicated"`
	FarmFreeIps uint64 `json:"farm_free_ips"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Nodes []Node `gorm:"foreignKey:farm_id;constraint:OnDelete:CASCADE" json:"nodes"`
}

type Identity substrate.Identity

type Versioned struct {
	Version string `json:"version"`
}

// Resources type
type Resources struct {
	HRU uint64 `json:"hru"`
	SRU uint64 `json:"sru"`
	CRU uint64 `json:"cru"`
	MRU uint64 `json:"mru"`
}

// Location type
type Location struct {
	City      string `json:"city"`
	Country   string `json:"country"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// OptionBoardSerial type
type OptionBoardSerial struct {
	HasValue bool   `json:"has_value"`
	AsValue  string `json:"as_value"`
}

// Interface type
type Interface struct {
	Name string   `json:"name"`
	Mac  string   `json:"mac"`
	IPs  []string `json:"ips"`
}
