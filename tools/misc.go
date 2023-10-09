package tools

import (
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/zondax/fil-parser/types"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

func ToRosetta(transactions []*types.Transaction) []*rosettaTypes.Transaction {
	var result []*rosettaTypes.Transaction
	var operations []*rosettaTypes.Operation
	lastHash := transactions[0].TxCid
	for _, t := range transactions {
		if t.TxCid != lastHash {
			result = append(result, &rosettaTypes.Transaction{
				TransactionIdentifier: &rosettaTypes.TransactionIdentifier{
					Hash: lastHash,
				},
				Operations: operations,
			})
			operations = nil
			lastHash = t.TxCid
		}
		operations = append(operations, operationFromTransaction(t))
	}
	return result
}

func operationFromTransaction(transaction *types.Transaction) *rosettaTypes.Operation {
	return &rosettaTypes.Operation{
		OperationIdentifier: &rosettaTypes.OperationIdentifier{
			Index: int64(transaction.BasicBlockData.Height),
		},
		Type:   transaction.TxType,
		Status: &transaction.Status,
		Account: &rosettaTypes.AccountIdentifier{
			Address: transaction.TxFrom,
		},
		Amount: &rosettaTypes.Amount{
			Value:    transaction.Amount.String(),
			Currency: rosetta.GetCurrencyData(),
		},
	}
}
