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
	"time"

	"strings"
)

type Parser struct {
	lib       *rosettaFilecoinLib.RosettaConstructionFilecoin
	addresses types.AddressInfoMap
}

func NewParser(lib *rosettaFilecoinLib.RosettaConstructionFilecoin) *Parser {
	return &Parser{
		lib:       lib,
		addresses: types.NewAddressInfoMap(),
	}
}

func (p *Parser) ParseTransactions(traces []*api.InvocResult, tipSet *filTypes.TipSet, ethLogs []EthLog) ([]*types.Transaction, *types.AddressInfoMap, error) {
	var transactions []*types.Transaction
	p.addresses = types.NewAddressInfoMap()
	tipsetKey := tipSet.Key()
	blockHash, err := rosetta.BuildTipSetKeyHash(tipsetKey)
	if err != nil {
		return nil, nil, errors.New("unable to get block hash") // TODO: define errors
	}
	for _, trace := range traces {
		if !hasMessage(trace) {
			continue
		}
		transaction, err := p.parseTrace(trace.ExecutionTrace, tipSet, ethLogs, *blockHash, trace.MsgCid.String(), tipsetKey)
		if err != nil {
			continue
		}
		transactions = append(transactions, transaction)

		// SubTransactions
		transactions = append(transactions, p.parseSubTxs(trace.ExecutionTrace.Subcalls, tipSet, ethLogs, *blockHash,
			trace.Msg.Cid().String(), tipsetKey)...)

		// Fees
		minerTxs := p.feesTransactions(trace.Msg, tipSet.Blocks()[0].Miner.String(), transaction.TxHash, *blockHash,
			transaction.TxType, trace.GasCost, uint64(tipSet.Height()), tipSet.MinTimestamp())
		transactions = append(transactions, minerTxs...)
	}

	return transactions, &p.addresses, nil
}

func (p *Parser) parseSubTxs(subTxs []filTypes.ExecutionTrace, tipSet *filTypes.TipSet, ethLogs []EthLog, blockHash, txHash string,
	key filTypes.TipSetKey) (txs []*types.Transaction) {

	for _, subTx := range subTxs {
		subTransaction, err := p.parseTrace(subTx, tipSet, ethLogs, blockHash, txHash, key)
		if err != nil {
			continue
		}
		txs = append(txs, subTransaction)
		txs = append(txs, p.parseSubTxs(subTx.Subcalls, tipSet, ethLogs, blockHash, txHash, key)...)
	}
	return
}

func (p *Parser) parseTrace(trace filTypes.ExecutionTrace, tipSet *filTypes.TipSet, ethLogs []EthLog, blockHash, txHash string,
	key filTypes.TipSetKey) (*types.Transaction, error) {
	txType, err := tools.GetMethodName(trace.Msg, int64(tipSet.Height()), key, p.lib)
	if err != nil {
		txType = "unknown"
	}

	metadata, mErr := p.getMetadata(txType, trace.Msg, trace.MsgRct, int64(tipSet.Height()), key, ethLogs)
	if mErr != nil {
		// TODO: log
	}
	params := parseParams(metadata)
	jsonMetadata, _ := json.Marshal(metadata)
	txReturn := parseReturn(metadata)

	p.appendAddressInfo(trace.Msg, int64(tipSet.Height()), key)

	return &types.Transaction{
		BasicBlockData: types.BasicBlockData{
			Height: uint64(tipSet.Height()),
			Hash:   blockHash,
		},
		TxTimestamp: p.getTimestamp(tipSet.MinTimestamp()),
		TxHash:      txHash,
		TxFrom:      trace.Msg.From.String(),
		TxTo:        trace.Msg.To.String(),
		Amount:      getCastedAmount(trace.Msg.Value.String()),
		Status:      getStatus(trace.MsgRct.ExitCode.String()),
		TxType:      txType,
		TxMetadata:  string(jsonMetadata),
		TxParams:    fmt.Sprintf("%v", params),
		TxReturn:    txReturn,
	}, nil
}

func (p *Parser) feesTransactions(msg *filTypes.Message, minerAddress, txHash, blockHash, txType string, gasCost api.MsgGasCost,
	height uint64, timestamp uint64) (feeTxs []*types.Transaction) {
	ts := p.getTimestamp(timestamp)
	feeTxs = append(feeTxs, p.newFeeTx(msg, "", txHash, blockHash, txType,
		tools.TotalFeeOp, getCastedAmount(gasCost.TotalCost.String()), height, ts))
	feeTxs = append(feeTxs, p.newFeeTx(msg, tools.BurnAddress, txHash, blockHash, txType,
		tools.OverEstimationBurnOp, getCastedAmount(gasCost.OverEstimationBurn.String()), height, ts))
	feeTxs = append(feeTxs, p.newFeeTx(msg, minerAddress, txHash, blockHash, txType,
		tools.MinerFeeOp, getCastedAmount(gasCost.MinerTip.String()), height, ts))
	feeTxs = append(feeTxs, p.newFeeTx(msg, tools.BurnAddress, txHash, blockHash, txType,
		tools.BurnFeeOp, getCastedAmount(gasCost.BaseFeeBurn.String()), height, ts))
	return
}

func (p *Parser) newFeeTx(msg *filTypes.Message, txTo, txHash, blockHash, txType, feeType string, gasCost string, height uint64,
	timestamp time.Time) *types.Transaction {
	return &types.Transaction{
		BasicBlockData: types.BasicBlockData{
			Height: height,
			Hash:   blockHash,
		},
		TxTimestamp: timestamp,
		TxHash:      txHash,
		TxFrom:      msg.From.String(),
		TxTo:        txTo,
		Amount:      gasCost,
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

func (p *Parser) getMetadata(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt, height int64, key filTypes.TipSetKey, ethLogs []EthLog) (map[string]interface{}, error) {
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
	if _, ok := tempMap[actor]; ok {
		if _, okk := tempMap[actor][txType]; !okk {
			if txType != "CronTick" && txType != "unknown" {
				fmt.Println(fmt.Sprintf("NEW METHOD TO IMPLEMENT: %s | %s", actor, txType))
			} else {
				fmt.Println(fmt.Sprintf("method %s", txType))
			}
		}
	}
	switch actor {
	case "init":
		return p.parseInit(txType, msg, msgRct, height, key)
	case "cron":
		return p.parseCron(txType, msg)
	case "account":
		return p.parseAccount(txType, msg)
	case "storagepower":
		return p.parseStoragepower(txType, msg, msgRct, height, key)
	case "storageminer":
		return p.parseStorageminer(txType, msg, height, key)
	case "storagemarket":
		return p.parseStoragemarket(txType, msg, msgRct)
	case "paymentchannel":
		return p.parsePaymentchannel(txType, msg)
	case "multisig":
		return p.parseMultisig(txType, msg, height, key)
	case "reward":
		return p.parseReward(txType, msg, msgRct)
	case "verifiedregistry":
		return p.parseVerifiedregistry(txType, msg)
	case "evm":
		return p.parseEvm(txType, msg, msgRct, ethLogs)
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

func (p *Parser) appendAddressInfo(msg *filTypes.Message, height int64, key filTypes.TipSetKey) {
	fromAdd := tools.GetActorAddressInfo(msg.From, height, key, p.lib)
	toAdd := tools.GetActorAddressInfo(msg.To, height, key, p.lib)
	p.appendToAddresses(fromAdd, toAdd)
}

func (p *Parser) appendToAddresses(info ...types.AddressInfo) {
	if p.addresses == nil {
		return
	}
	for _, i := range info {
		if i.Robust != "" && i.Short != "" && i.Robust != i.Short {
			if _, ok := p.addresses[i.Short]; !ok {
				p.addresses[i.Short] = i
			}
		}
	}
}

func (p *Parser) getTimestamp(timestamp uint64) time.Time {
	blockTimeStamp := int64(timestamp) * 1000
	return time.Unix(blockTimeStamp/1000, blockTimeStamp%1000)
}
