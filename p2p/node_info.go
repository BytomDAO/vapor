package p2p

import (
	"fmt"
	"net"
	"github.com/tendermint/go-crypto"

	cfg "github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/version"
)

const maxNodeInfoSize = 10240 // 10Kb

var (
	errDiffMajorVersion = errors.New("Peer is on a different major version.")
	errDiffNetwork      = errors.New("Peer is on a different network name.")
	errDiffNetworkID    = errors.New("Peer has different network ID.")
)

//NodeInfo peer node info
type NodeInfo struct {
	PubKey  crypto.PubKeyEd25519 `json:"pub_key"`
	Moniker string               `json:"moniker"`
	Network string               `json:"network"`
	//NetworkID used to isolate subnets with same network name
	NetworkID   uint64                `json:"network_id"`
	RemoteAddr  string                `json:"remote_addr"`
	ListenAddr  string                `json:"listen_addr"`
	Version     string                `json:"version"` // major.minor.revision
	ServiceFlag consensus.ServiceFlag `json:"service_flag"`
	// other application specific data
	Other []string `json:"other"`
}

func NewNodeInfo(config *cfg.Config, pubkey crypto.PubKeyEd25519, listenAddr string, netID uint64) *NodeInfo {
	return &NodeInfo{
		PubKey:      pubkey,
		Moniker:     config.Moniker,
		Network:     config.ChainID,
		NetworkID:   netID,
		ListenAddr:  listenAddr,
		Version:     version.Version,
		ServiceFlag: consensus.DefaultServices,
	}
}

type VersionCompatibleWith func(remoteVerStr string) (bool, error)

// CompatibleWith checks if two NodeInfo are compatible with eachother.
// CONTRACT: two nodes are compatible if the major version matches and network match
func (info *NodeInfo) compatibleWith(other *NodeInfo, versionCompatibleWith VersionCompatibleWith) error {
	if info.Network != other.Network {
		return errors.Wrapf(errDiffNetwork, "Peer network: %v, node network: %v", other.Network, info.Network)
	}

	if info.NetworkID != other.NetworkID {
		return errors.Wrapf(errDiffNetworkID, "Peer network id: %v, node network id: %v", other.NetworkID, info.NetworkID)
	}

	compatible, err := versionCompatibleWith(other.Version)
	if err != nil {
		return err
	}

	if !compatible {
		return errors.Wrapf(errDiffMajorVersion, "Peer version: %v, node version: %v", other.Version, info.Version)
	}

	return nil
}

func (info *NodeInfo) getPubkey() crypto.PubKeyEd25519 {
	return info.PubKey
}

//ListenHost peer listener ip address
func (info NodeInfo) ListenHost() string {
	host, _, _ := net.SplitHostPort(info.ListenAddr)
	return host
}

//RemoteAddrHost peer external ip address
func (info NodeInfo) RemoteAddrHost() string {
	host, _, _ := net.SplitHostPort(info.RemoteAddr)
	return host
}

//GetNetwork get node info network field
func (info *NodeInfo) GetNetwork() string {
	return info.Network
}

//String representation
func (info NodeInfo) String() string {
	return fmt.Sprintf("NodeInfo{pk: %v, moniker: %v, network: %v [listen %v], service: %v,version: %v (%v)}", info.PubKey, info.Moniker, info.Network, info.ListenAddr, info.Version, info.ServiceFlag, info.Other)
}
