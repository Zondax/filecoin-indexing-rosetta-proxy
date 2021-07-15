package types

import "github.com/ipfs/go-cid"

type AddressInfo struct {
	// Short is the address in 'short' format
	Short string
	// Robust is the address in 'robust' format
	Robust string
	// ActorCid is the actor's cid for this address
	ActorCid cid.Cid
	// ActorType is the actor's type name of this address
	ActorType string
}

type AddressInfoMap map[string]AddressInfo

func NewAddressInfoMap() AddressInfoMap {
	return make(AddressInfoMap)
}

type MinerInfo struct {
	// Short is the miner's address in 'short' format
	Short string
	// Robust is the miner's address in 'robust' format
	Robust string
	// Owner is the owner address of this miner
	Owner string
	// Worker is the worker address of this miner
	Worker string
}

func (a AddressInfo) GetAddress() string {
	if a.Robust != "" {
		return a.Robust
	}
	if a.Short != "" {
		return a.Short
	}
	return "unknown"
}
