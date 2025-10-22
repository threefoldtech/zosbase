package mycelium

import (
	"context"
	"crypto/ed25519"

	"github.com/oasisprotocol/curve25519-voi/primitives/x25519"
	"github.com/pkg/errors"
)

type NodeConfig struct {
	KeyFile    string
	TunName    string
	Peers      []string
	privateKey x25519.PrivateKey
}

func FindPeers(ctx context.Context, filter ...Filter) (Peers, error) {
	// fetching a peer list goes as this
	// - Always include the list of peers from
	peers, err := fetchZosMyList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get zos public peer list")
	}

	peers, err = peers.Ups(filter...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter out peer list")
	}

	return peers, nil
}

func (n *NodeConfig) AssignPeers(ctx context.Context, filter ...Filter) error {
	peers, err := FindPeers(ctx, filter...)
	if err != nil {
		return err
	}
	n.Peers = peers
	return nil
}

// GenerateConfig creates a new mycelium configuration
func GenerateConfig(privateKey ed25519.PrivateKey) (cfg NodeConfig) {
	cfg = NodeConfig{
		KeyFile: confPath,
		TunName: tunName,
	}

	if privateKey != nil {
		cfg.privateKey = x25519.PrivateKey(x25519.EdPrivateKeyToX25519([]byte(privateKey)))
	}

	return
}
