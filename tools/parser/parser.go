package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/shopspring/decimal"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"

	"strings"
)

var (
	burnAddress = "f099"
)

func hasMessage(trace *api.InvocResult) bool {
	return trace.Msg != nil
}

func ParseTransactions(traces []*api.InvocResult, tipSet *filTypes.TipSet, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (*[]*types.Transaction, error) {
	var transactions []*types.Transaction
	tipsetKey := tipSet.Key()
	blockHash, err := rosetta.BuildTipSetKeyHash(tipsetKey)
	if err != nil {
		return nil, errors.New("unable to get block hash") // TODO: define errors
	}
	for _, trace := range traces {
		if !hasMessage(trace) {
			continue
		}
		transaction, err := parseTrace(trace.Msg, trace.MsgRct, tipSet, *blockHash, trace.MsgCid.String(), tipsetKey, lib)
		if err != nil {
			// TODO: logging
			continue
		}
		transactions = append(transactions, transaction)

		// SubTransactions
		transactions = append(transactions, parseSubTxs(trace.ExecutionTrace.Subcalls, tipSet, *blockHash,
			trace.Msg.Cid().String(), tipsetKey, lib)...)

		// Fees
		minerTxs := feesTransactions(trace.Msg, tipSet.Blocks()[0].Miner.String(), transaction.TxHash, *blockHash,
			transaction.TxType, trace.GasCost, uint64(tipSet.Height()), int64(tipSet.MinTimestamp()))
		transactions = append(transactions, minerTxs...)
	}

	return &transactions, nil
}

func parseSubTxs(subTxs []filTypes.ExecutionTrace, tipSet *filTypes.TipSet, blockHash, txHash string, key filTypes.TipSetKey,
	lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (txs []*types.Transaction) {

	for _, subTx := range subTxs {
		subTransaction, err := parseTrace(subTx.Msg, subTx.MsgRct, tipSet, blockHash, txHash, key, lib)
		if err != nil {
			continue
		}
		txs = append(txs, subTransaction)
		txs = append(txs, parseSubTxs(subTx.Subcalls, tipSet, blockHash, txHash, key, lib)...)
	}
	return
}

func parseTrace(msg *filTypes.Message, msgRct *filTypes.MessageReceipt, tipSet *filTypes.TipSet, blockHash, txHash string, key filTypes.TipSetKey,
	lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (*types.Transaction, error) {
	txType, err := tools.GetMethodName(msg, int64(tipSet.Height()), key, lib)
	if err != nil {
		txType = "unknown"
	}
	if !tools.IsOpSupported(txType) {
		return nil, errors.New("operation not supported") // TODO: define errors
	}
	metadata, mErr := getMetadata(txType, msg, int64(tipSet.Height()), key, lib)
	if mErr != nil {
		// TODO: log
	}
	params := parseParams(metadata)
	jsonMetadata, _ := json.Marshal(metadata)
	txReturn := parseReturn(metadata)

	return &types.Transaction{
		BasicBlockData: types.BasicBlockData{
			Height: uint64(tipSet.Height()),
			Hash:   blockHash,
		},
		TxTimestamp: int64(tipSet.MinTimestamp()),
		TxHash:      txHash,
		TxFrom:      msg.From.String(),
		TxTo:        msg.To.String(),
		Amount:      getCastedAmount(msg.Value.String()),
		Status:      getStatus(msgRct.ExitCode.String()),
		TxType:      txType,
		TxMetadata:  string(jsonMetadata),
		TxParams:    fmt.Sprintf("%v", params),
		TxReturn:    txReturn,
	}, nil
}

func feesTransactions(msg *filTypes.Message, minerAddress, txHash, blockHash, txType string, gasCost api.MsgGasCost,
	height uint64, timestamp int64) (feeTxs []*types.Transaction) {
	feeTxs = append(feeTxs, newFeeTx(msg, "", txHash, blockHash, txType,
		tools.TotalFeeOp, gasCost.TotalCost.String(), height, timestamp))
	feeTxs = append(feeTxs, newFeeTx(msg, burnAddress, txHash, blockHash, txType,
		tools.OverEstimationBurnOp, gasCost.OverEstimationBurn.String(), height, timestamp))
	feeTxs = append(feeTxs, newFeeTx(msg, minerAddress, txHash, blockHash, txType,
		tools.MinerFeeOp, gasCost.MinerTip.String(), height, timestamp))
	feeTxs = append(feeTxs, newFeeTx(msg, burnAddress, txHash, blockHash, txType,
		tools.BurnFeeOp, gasCost.BaseFeeBurn.String(), height, timestamp))
	return
}

func newFeeTx(msg *filTypes.Message, txTo, txHash, blockHash, txType, feeType string, gasCost string, height uint64,
	timestamp int64) *types.Transaction {
	return &types.Transaction{
		BasicBlockData: types.BasicBlockData{
			Height: height,
			Hash:   blockHash,
		},
		TxTimestamp: timestamp,
		TxHash:      txHash,
		TxFrom:      msg.From.String(),
		TxTo:        txTo,
		Amount:      getCastedAmount(gasCost),
		Status:      "OK",
		TxType:      feeType,
		TxMetadata:  txType,
	}
}

func getStatus(code string) string {
	status := strings.Split(code, "(")
	if len(status) == 2 {
		return status[0]
	}
	return code
}

func getMetadata(txType string, msg *filTypes.Message, height int64, key filTypes.TipSetKey,
	lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	var err error
	switch txType {
	case "Send":
		metadata["Params"] = msg.Params
	case "Propose":
		metadata, err = ParseProposeParams(msg, height, key, lib)
		if err != nil {
			rosetta.Logger.Errorf("Could not parse message params for %v, error: %v", txType, err.Error())
			return metadata, err
		}
	case "SwapSigner", "AddSigner", "RemoveSigner":
		params, err := ParseMsigParams(msg, height, key, lib)
		if err != nil {
			return metadata, err
		}
		err = json.Unmarshal([]byte(params), &metadata)
		if err != nil {
			return metadata, err
		}
	case "Exec":
		break
	case "Constructor":
		//	TODO: implement
	}
	return metadata, nil
}

func parseParams(metadata map[string]interface{}) string {
	params, ok := metadata["Params"].(string)
	if ok && params != "" {
		return params
	}
	jsonMetadata, err := json.Marshal(metadata["Params"])
	if err == nil {
		return string(jsonMetadata)
	}
	return ""
}

func parseReturn(metadata map[string]interface{}) string {
	params, ok := metadata["Return"].(string)
	if ok && params != "" {
		return params
	}
	jsonMetadata, err := json.Marshal(metadata["Return"])
	if err == nil && string(jsonMetadata) != "null" {
		return string(jsonMetadata)
	}
	return ""
}

func getCastedAmount(amount string) string {
	decimal.DivisionPrecision = 18
	parsed, err := decimal.NewFromString(amount)
	if err != nil {
		return amount
	}
	abs := parsed.Abs()
	divided := abs.Div(decimal.NewFromInt(1e+18))
	return divided.String()
}
