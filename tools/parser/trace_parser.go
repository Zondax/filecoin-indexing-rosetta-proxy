package parser

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-state-types/builtin/v10/eam"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"go.uber.org/zap"
)

type TransactionParser struct {
	rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin
	rpcClient  api.FullNode
}

func NewTransactionParser(rpcClient api.FullNode) *TransactionParser {
	lib := rosettaFilecoinLib.NewRosettaConstructionFilecoin(&rpcClient)
	if lib == nil {
		zap.S().Fatalf("could not create instance of rosettaLib")
	}

	return &TransactionParser{
		rosettaLib: lib,
		rpcClient:  rpcClient,
	}
}

func (t *TransactionParser) BuildTransactions(states *api.ComputeStateOutput, rawTipset filTypes.TipSet) ([]*Transaction, *types.AddressInfoMap) {
	var discoveredAddresses = types.NewAddressInfoMap()
	var transactions []*Transaction

	for i := range states.Trace {
		trace := states.Trace[i]

		if trace.Msg == nil {
			continue
		}

		// Analyze full trace recursively
		t.ProcessTrace(&trace.ExecutionTrace, trace.MsgCid, transactions, rawTipset, &discoveredAddresses)
		// Add the corresponding "Fee" operation
		if !trace.GasCost.TotalCost.NilOrZero() {

			feeTx := t.BuildFeeTx(trace)

			transactions = append(transactions, txFee)

			transactions = AppendOp(transactions, tools.TotalFeeOp, trace.Msg.From.String(),
				trace.GasCost.TotalCost.Neg().String(), txStatus, false, nil)

			transactions = AppendOp(transactions, tools.OverEstimationBurnOp, trace.Msg.From.String(),
				trace.GasCost.OverEstimationBurn.Neg().String(), txStatus, false, nil)

			transactions = AppendOp(transactions, tools.MinerFeeOp, trace.Msg.From.String(),
				trace.GasCost.MinerTip.Neg().String(), txStatus, false, nil)

			transactions = AppendOp(transactions, tools.BurnFeeOp, trace.Msg.From.String(),
				trace.GasCost.BaseFeeBurn.Neg().String(), txStatus, false, nil)
		}
	}
	return transactions, &discoveredAddresses
}

func (t *TransactionParser) BuildFeeTx(trace *api.InvocResult, height int64, key filTypes.TipSetKey) *Transaction {
	amount := GetCastedAmount(trace.Msg.Value.String())

	TransactionFeeInfo{
		TxHash:      "",
		MethodName:  "",
		TotalCost:   0,
		GasUsage:    0,
		GasLimit:    0,
		GasPremium:  0,
		BaseFeeBurn: 0,
	}

	feeTx := &Transaction{
		Height:      int64(rawTipset.TipSet.Height()),
		TxTimestamp: int64(rawTipset.TipSet.MinTimestamp()),
		TxHash:      tx.TransactionIdentifier.Hash,
		TxFrom:      op.Account.Address,
		Amount:      amount,
		TxType:      op.Type,
		Status:      *op.Status,
		TxMetadata:  baseTxType,
		TxParams:    "",
	}

	switch op.Type {
	case tools.MinerFeeOp:
		// Search the miner who mined this tx
		blocks, ok := rawTipset.BlockMessages[feeTx.TxHash]
		if !ok {
			zap.S().Errorf("Warning: could not find tx hash '%s' in rawTipsetData!", feeTx.TxHash)
			break
		}
		// The miner who mined the first block where this tx was included on, gets the fee
		feeTx.TxTo = blocks[0].BlockMiner
	case tools.BurnFeeOp, tools.OverEstimationBurnOp:
		feeTx.TxTo = types.BurnAddress
	}

	return feeTx
}

func (t *TransactionParser) ProcessTrace(trace *filTypes.ExecutionTrace, msgCid cid.Cid, transactions []*Transaction, rawTipset filTypes.TipSet,
	addresses *types.AddressInfoMap) {

	if trace.Msg == nil {
		return
	}

	txStatus := TransactionStatusOk
	if !trace.MsgRct.ExitCode.IsSuccess() {
		txStatus = TransactionStatusFailed
	}

	baseMethod, err := tools.GetMethodName(trace.Msg, int64(rawTipset.Height()), rawTipset.Key(), t.rosettaLib)
	if err != nil {
		zap.S().Errorf("could not get method name: %v", err)
		baseMethod = UnknownStr
	}

	fromAdd := tools.GetActorAddressInfo(trace.Msg.From, int64(rawTipset.Height()), rawTipset.Key(), t.rosettaLib)
	toAdd := tools.GetActorAddressInfo(trace.Msg.To, int64(rawTipset.Height()), rawTipset.Key(), t.rosettaLib)
	appendAddressInfo(addresses, fromAdd, toAdd)

	// Basic tx info
	tx := &Transaction{
		Height:      int64(rawTipset.Height()),
		TxTimestamp: int64(rawTipset.MinTimestamp()),
		TxHash:      trace.Msg.Cid().String(),
		TxFrom:      fromAdd.GetAddress(),
		TxTo:        toAdd.GetAddress(),
		Amount:      GetCastedAmount(trace.Msg.Value.String()),
		TxType:      baseMethod,
		Status:      txStatus,
	}

	switch baseMethod {
	case "InvokeContract", "InvokeContractReadOnly", "InvokeContractDelegate":
		{
			tx.TxParams = "0x" + hex.EncodeToString(trace.Msg.Params)
			tx.TxReturn = "0x" + hex.EncodeToString(trace.MsgRct.Return)
		}
	case "Create", "Create2":
		{
			metadata := make(map[string]interface{})
			var result eam.CreateReturn
			r := bytes.NewReader(trace.MsgRct.Return)
			if err := result.UnmarshalCBOR(r); err != nil {
				zap.S().Errorf("error unmarshaling return value for method '%s': %v", baseMethod, err)
				break
			}

			ethHash, err := api.NewEthHashFromCid(msgCid)
			if err != nil {
				zap.S().Errorf("error getting eth hash from cid for methos '%s': %v", baseMethod, err)
				break
			}

			metadata["robustAdd"] = result.RobustAddress.String()
			metadata["ethAdd"] = "0x" + hex.EncodeToString(result.EthAddress[:])
			metadata["cid"] = msgCid.String()
			metadata["ethHash"] = ethHash.String()
		}
	case "AddBalance":
		{
			// We don't add anything here to tx
			break
		}
	case "Send":
		{
			tx.TxMetadata = string(trace.Msg.Params)
		}
	case "CreateMiner":
		{
			createdActor, err := searchforactorcreation(trace.Msg, trace.MsgRct, height, key, lib)
			if err != nil {
				rosetta.Logger.Errorf("Could not parse 'CreateMiner' params, err: %v", err)
				break
			}
			appendAddressInfo(addresses, *createdActor)
		}
	case "Exec":
		{
			*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), txStatus, false, nil)
			*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), txStatus, true, nil)

			// Check if this Exec contains actor creation event
			createdActor, err := searchForActorCreation(trace.Msg, trace.MsgRct, height, key, lib)
			if err != nil {
				rosetta.Logger.Errorf("Could not parse Exec params, err: %v", err)
				break
			}

			if createdActor == nil {
				// This is not an actor creation event
				break
			}

			appendAddressInfo(addresses, *createdActor)

			// Check if the created actor is of multisig type and if it was also funded
			if lib.BuiltinActors.IsActor(createdActor.ActorCid, actors.ActorMultisigName) &&
				!trace.Msg.Value.NilOrZero() {
				from := toAdd.Short
				to := createdActor.Short

				*transactions = AppendOp(*transactions, "Exec", from,
					trace.Msg.Value.Neg().String(), txStatus, true, nil)
				*transactions = AppendOp(*transactions, "Exec", to,
					trace.Msg.Value.String(), txStatus, true, nil)
			}
		}
	case "Propose":
		{
			params, err := ParseProposeParams(trace.Msg, height, key, lib)
			if err != nil {
				rosetta.Logger.Errorf("Could not parse message params for %v, error: %v", baseMethod, err.Error())
				break
			}

			*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
				"0", txStatus, false, &params)
			*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
				"0", txStatus, true, &params)
		}
	case "SwapSigner", "AddSigner", "RemoveSigner":
		{
			params, err := ParseMsigParams(trace.Msg, height, key, lib)
			if err == nil {
				var paramsMap map[string]interface{}
				if err := json.Unmarshal([]byte(params), &paramsMap); err == nil {
					switch baseMethod {
					case "SwapSigner":
						{
							*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
								"0", txStatus, false, &paramsMap)
							*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
								"0", txStatus, true, &paramsMap)
						}
					case "AddSigner", "RemoveSigner":
						{
							*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
								"0", txStatus, false, &paramsMap)
							*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
								"0", txStatus, true, &paramsMap)
						}
					}

				} else {
					rosetta.Logger.Error("Could not parse message params for", baseMethod)
				}
			}
		}
	case "AwardBlockReward", "ApplyRewards", "OnDeferredCronEvent",
		"PreCommitSector", "ProveCommitSector", "SubmitWindowedPoSt",
		"DeclareFaultsRecovered", "ChangeWorkerAddress", "PreCommitSectorBatch",
		"ProveCommitAggregate", "ProveReplicaUpdates":
		{
			*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), txStatus, false, nil)
			*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), txStatus, true, nil)
		}
	case "Approve", "Cancel":
		{
			*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), txStatus, false, nil)
			*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), txStatus, true, nil)
		}
	}

	transactions = append(transactions, tx)

	// Only process sub-calls if the parent call was successfully executed
	if txStatus == TransactionStatusOk {
		for i := range trace.Subcalls {
			subTrace := trace.Subcalls[i]
			t.ProcessTrace(&subTrace, msgCid, transactions, rawTipset, addresses)
		}
	}
}

func searchForActorCreation(msg *filTypes.Message, receipt *filTypes.MessageReceipt,
	height int64, key filTypes.TipSetKey, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (*types.AddressInfo, error) {

	toAddressInfo := tools.GetActorAddressInfo(msg.To, height, key, lib)
	actorName, err := lib.BuiltinActors.GetActorNameFromCid(toAddressInfo.ActorCid)
	if err != nil {
		return nil, err
	}

	switch actorName {
	case "init":
		{
			params, err := ParseInitActorExecParams(msg.Params)
			if err != nil {
				return nil, err
			}
			createdActorName, err := lib.BuiltinActors.GetActorNameFromCid(params.CodeCID)
			if err != nil {
				return nil, err
			}
			switch createdActorName {
			case "multisig", "paymentchannel":
				{
					execReturn, err := ParseExecReturn(receipt.Return)
					if err != nil {
						return nil, err
					}

					return &types.AddressInfo{
						Short:     execReturn.IDAddress.String(),
						Robust:    execReturn.RobustAddress.String(),
						ActorCid:  params.CodeCID,
						ActorType: createdActorName,
					}, nil
				}
			default:
				return nil, nil
			}
		}
	case "storagepower":
		{
			execReturn, err := ParseExecReturn(receipt.Return)
			if err != nil {
				return nil, err
			}
			return &types.AddressInfo{
				Short:     execReturn.IDAddress.String(),
				Robust:    execReturn.RobustAddress.String(),
				ActorType: "miner",
			}, nil
		}
	default:
		return nil, nil
	}
}

func AppendOp(ops []*rosettaTypes.Operation, opType string, account string, amount string, status string, relateOp bool, metadata *map[string]interface{}) []*rosettaTypes.Operation {
	opIndex := int64(len(ops))
	op := &rosettaTypes.Operation{
		OperationIdentifier: &rosettaTypes.OperationIdentifier{
			Index: opIndex,
		},
		Type:   opType,
		Status: &status,
		Account: &rosettaTypes.AccountIdentifier{
			Address: account,
		},
		Amount: &rosettaTypes.Amount{
			Value:    amount,
			Currency: rosetta.GetCurrencyData(),
		},
	}

	// Add metadata
	if metadata != nil {
		op.Metadata = *metadata
	}

	// Add related operation
	if relateOp && opIndex > 0 {
		op.RelatedOperations = []*rosettaTypes.OperationIdentifier{
			{
				Index: opIndex - 1,
			},
		}
	}

	return append(ops, op)
}

func appendAddressInfo(addressMap *types.AddressInfoMap, info ...types.AddressInfo) {
	if addressMap == nil {
		return
	}
	for _, i := range info {
		if i.Robust != "" && i.Short != "" && i.Robust != i.Short {
			if _, ok := (*addressMap)[i.Short]; !ok {
				(*addressMap)[i.Short] = i
			}
		}
	}
}
