package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/shopspring/decimal"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"

	"strings"
)

type Parser struct {
	lib *rosettaFilecoinLib.RosettaConstructionFilecoin
}

func NewParser(lib *rosettaFilecoinLib.RosettaConstructionFilecoin) *Parser {
	return &Parser{
		lib: lib,
	}
}

func (p *Parser) ParseTransactions(traces []*api.InvocResult, tipSet *filTypes.TipSet) (*[]*types.Transaction, error) {
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
		transaction, err := p.parseTrace(trace.Msg, trace.MsgRct, tipSet, *blockHash, trace.MsgCid.String(), tipsetKey)
		if err != nil {
			// TODO: logging
			continue
		}
		transactions = append(transactions, transaction)

		// SubTransactions
		transactions = append(transactions, p.parseSubTxs(trace.ExecutionTrace.Subcalls, tipSet, *blockHash,
			trace.Msg.Cid().String(), tipsetKey)...)

		// Fees
		minerTxs := feesTransactions(trace.Msg, tipSet.Blocks()[0].Miner.String(), transaction.TxHash, *blockHash,
			transaction.TxType, trace.GasCost, uint64(tipSet.Height()), int64(tipSet.MinTimestamp()))
		transactions = append(transactions, minerTxs...)
	}

	return &transactions, nil
}

func (p *Parser) parseSubTxs(subTxs []filTypes.ExecutionTrace, tipSet *filTypes.TipSet, blockHash, txHash string,
	key filTypes.TipSetKey) (txs []*types.Transaction) {

	for _, subTx := range subTxs {
		subTransaction, err := p.parseTrace(subTx.Msg, subTx.MsgRct, tipSet, blockHash, txHash, key)
		if err != nil {
			continue
		}
		txs = append(txs, subTransaction)
		txs = append(txs, p.parseSubTxs(subTx.Subcalls, tipSet, blockHash, txHash, key)...)
	}
	return
}

func (p *Parser) parseTrace(msg *filTypes.Message, msgRct *filTypes.MessageReceipt, tipSet *filTypes.TipSet, blockHash, txHash string,
	key filTypes.TipSetKey) (*types.Transaction, error) {
	txType, err := tools.GetMethodName(msg, int64(tipSet.Height()), key, p.lib)
	if err != nil {
		txType = "unknown"
	}
	if !tools.IsOpSupported(txType) {
		return nil, errors.New("operation not supported") // TODO: define errors
	}
	metadata, mErr := p.getMetadata(txType, msg, msgRct, int64(tipSet.Height()), key)
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
		tools.TotalFeeOp, getCastedAmount(gasCost.TotalCost.String()), height, timestamp))
	feeTxs = append(feeTxs, newFeeTx(msg, tools.BurnAddress, txHash, blockHash, txType,
		tools.OverEstimationBurnOp, getCastedAmount(gasCost.OverEstimationBurn.String()), height, timestamp))
	feeTxs = append(feeTxs, newFeeTx(msg, minerAddress, txHash, blockHash, txType,
		tools.MinerFeeOp, getCastedAmount(gasCost.MinerTip.String()), height, timestamp))
	feeTxs = append(feeTxs, newFeeTx(msg, tools.BurnAddress, txHash, blockHash, txType,
		tools.BurnFeeOp, getCastedAmount(gasCost.BaseFeeBurn.String()), height, timestamp))
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
		Status:      "Ok",
		TxType:      feeType,
		TxMetadata:  txType,
	}
}

func hasMessage(trace *api.InvocResult) bool {
	return trace.Msg != nil
}

func getStatus(code string) string {
	status := strings.Split(code, "(")
	if len(status) == 2 {
		return status[0]
	}
	return code
}

func (p *Parser) getMetadata(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	var err error
	actorCode, err := database.ActorsDB.GetActorCode(msg.To, height, key)
	if err != nil {
		return metadata, err
	}
	actor, err := p.lib.BuiltinActors.GetActorNameFromCid(actorCode)
	if err != nil {
		return metadata, err
	}
	switch actor {
	case "init":
		return p.parseInit(msg, msgRct, height, key)
	case "cron":
		return p.parseCron(msg, height, key)
	case "account":
		return p.parseAccount(msg, height, key)
	case "storagepower":
		return p.parseStoragepower(msg, height, key)
	case "storageminer":
		return p.parseStorageminer(msg, height, key)
	case "storagemarket":
		return p.parseStoragemarket(msg, height, key)
	case "paymentchannel":
		return p.parsePaymentchannel(msg, height, key)
	case "multisig":
		return p.parseMultisig(msg, height, key)
	case "reward":
		return p.parseReward(msg, height, key)
	case "verifiedregistry":
		return p.parseVerifiedregistry(msg, height, key)
	default:
		return metadata, errors.New("not a valid actor")
	}
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
