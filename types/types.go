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

type TransactionFeeInfo struct {
	// TxHash is the identifier for this transaction
	TxHash string
	// MethodName is the method requested to be executed
	MethodName string
	// TotalCost is the total fee payed by the sender. Expressed in [FIL]
	TotalCost uint64
	// GasUsage is the amount of GAS used to execute this transaction. Expressed in units of [GAS]
	GasUsage uint64
	// GasLimit is the maximum amount of gas that this transaction can use. Expressed in units of [GAS]
	GasLimit int64
	// GasPremium is the amount if FIL payed to the miner per unit of GAS. Expressed in [FIL/GAS]
	GasPremium uint64
	// BaseFeeBurn is this block's burned fee. Expressed in [FIL]
	BaseFeeBurn uint64
}
