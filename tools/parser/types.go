package parser

import (
	"github.com/ipfs/go-cid"
	"google.golang.org/genproto/googleapis/type/decimal"
)

type Transaction struct {
	// Height contains the block height.
	Height int64 `json:"height" gorm:"index:idx_transactions_height"`
	// TxTimestamp is the timestamp of the transaction
	TxTimestamp int64 `json:"tx_timestamp"`
	// TxHash is the transaction hash
	TxHash string `json:"tx_hash" gorm:"index:idx_transactions_tx_hash"`
	// TxFrom is the sender address
	TxFrom string `json:"tx_from" gorm:"index:idx_transactions_tx_from"`
	// TxTo is the receiver address
	TxTo string `json:"tx_to" gorm:"index:idx_transactions_tx_to"`
	// Amount is the amount of the tx
	Amount decimal.Decimal `json:"amount" gorm:"type:numeric"`
	// Status
	Status string `json:"status"`
	// TxType is the message type
	TxType string `json:"tx_type"`
	// TxMetadata is the message metadata
	TxMetadata string `json:"tx_metadata"`
	// TxParams contain the transaction params
	TxParams string `json:"tx_params"`
	// TxReturn contains the returned data by the destination actor
	TxReturn string `json:"tx_return"`
}

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
