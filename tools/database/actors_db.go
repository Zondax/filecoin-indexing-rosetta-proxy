package database

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	cmap "github.com/orcaman/concurrent-map"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

var ActorsDB Database

type AddressInfo struct {
	// Short is the address in 'short' format
	Short string
	// Robust is the address in 'robust' format
	Robust string
	// ActorCid is the actor's cid for this address
	ActorCid cid.Cid
}

func (a AddressInfo) GetAddressForActorType() string {
	switch rosetta.GetActorNameFromCid(a.ActorCid) {
	case "multisig":
		return a.Short
	default:
		return a.Robust
	}
}

type Database interface {
	NewImpl(*api.FullNode)
	GetActorCode(robustAdd address.Address) (cid.Cid, error)
	GetRobustAddress(shortAdd address.Address) (string, error)
	GetShortAddress(robustAdd address.Address) (string, error)
	StoreRobustShort(robust string, short string)
	StoreShortRobust(short string, robust string)
	StoreAddressInfo(info AddressInfo)
}

// Cache In-memory database
type Cache struct {
	robustCidMap   cmap.ConcurrentMap
	robustShortMap cmap.ConcurrentMap
	shortRobustMap cmap.ConcurrentMap
	Node           *api.FullNode
}

func (m *Cache) NewImpl(node *api.FullNode) {
	m.robustCidMap = cmap.New()
	m.robustShortMap = cmap.New()
	m.shortRobustMap = cmap.New()
	m.Node = node
}

func (m *Cache) GetActorCode(address address.Address) (cid.Cid, error) {
	robustAdd, _ := m.GetRobustAddress(address)

	code, ok := m.robustCidMap.Get(robustAdd)
	if !ok {
		var err error
		code, err = m.retrieveActorFromLotus(address)
		if err != nil {
			return cid.Cid{}, err
		}
		m.storeActorCode(robustAdd, code.(cid.Cid))
	}

	return code.(cid.Cid), nil
}

func (m *Cache) GetRobustAddress(address address.Address) (string, error) {
	isRobustAddress, err := IsRobustAddress(address)
	if err != nil {
		return "", err
	}

	if isRobustAddress {
		// Already a robust address
		return address.String(), nil
	}

	// This is a short address, get the robust one
	robustAdd, ok := m.shortRobustMap.Get(address.String())
	if !ok {
		var err error
		// Get robust address from lotus
		robustAdd, err = m.retrieveActorPubKeyFromLotus(address, false)
		if err != nil {
			return "", err
		}
		m.StoreShortRobust(address.String(), robustAdd.(string))
	}

	return robustAdd.(string), nil
}

func (m *Cache) GetShortAddress(address address.Address) (string, error) {
	isRobustAddress, err := IsRobustAddress(address)
	if err != nil {
		return "", err
	}

	if !isRobustAddress {
		// Already a short address
		return address.String(), nil
	}

	// This is a robust address, get the short one
	shortAdd, ok := m.robustShortMap.Get(address.String())
	if !ok {
		var err error
		shortAdd, err = m.retrieveActorPubKeyFromLotus(address, true)
		if err != nil {
			return address.String(), err
		}
		m.StoreRobustShort(address.String(), shortAdd.(string))
	}

	return shortAdd.(string), nil
}

func (m *Cache) StoreRobustShort(robust string, short string) {
	m.robustShortMap.Set(robust, short)
}

func (m *Cache) StoreShortRobust(short string, robust string) {
	m.shortRobustMap.Set(short, robust)
}

func (m Cache) StoreAddressInfo(info AddressInfo) {
	m.StoreRobustShort(info.Robust, info.Short)
	m.StoreShortRobust(info.Short, info.Robust)
	m.storeActorCode(info.Robust, info.ActorCid)
}

func (m *Cache) storeActorCode(robustAddress string, cid cid.Cid) {
	m.robustCidMap.Set(robustAddress, cid)
}

func (m *Cache) retrieveActorFromLotus(add address.Address) (cid.Cid, error) {
	actor, err := (*m.Node).StateGetActor(context.Background(), add, filTypes.EmptyTSK)
	if err != nil {
		return cid.Cid{}, err
	}

	return actor.Code, nil
}

func (m *Cache) retrieveActorPubKeyFromLotus(add address.Address, reverse bool) (string, error) {
	var key address.Address
	var err error
	if reverse {
		key, err = (*m.Node).StateLookupID(context.Background(), add, filTypes.EmptyTSK)
	} else {
		key, err = (*m.Node).StateAccountKey(context.Background(), add, filTypes.EmptyTSK)
	}

	if err != nil {
		return add.String(), nil
	}
	return key.String(), nil
}

func IsRobustAddress(add address.Address) (bool, error) {
	switch add.Protocol() {
	case address.BLS, address.SECP256K1, address.Actor:
		return true, nil
	case address.ID:
		return false, nil
	default:
		// Consider unknown type as robust
		return false, fmt.Errorf("unknown address type for %s", add.String())
	}
}
