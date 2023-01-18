package parser

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-state-types/builtin/v10/eam"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/ipfs/go-cid"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"go.uber.org/zap"
	"time"
)

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

func BuildTransactions(states *ComputeStateVersioned, height int64, key filTypes.TipSetKey, ethLogs []EthLog, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (*[]*rosettaTypes.Transaction, *types.AddressInfoMap) {
	defer rosetta.TimeTrack(time.Now(), "[Proxy]TraceAnalysis")

	var transactions []*rosettaTypes.Transaction
	var discoveredAddresses = types.NewAddressInfoMap()
	for i := range states.Trace {
		trace := states.Trace[i]

		if trace.Msg == nil {
			continue
		}

		var operations []*rosettaTypes.Operation

		// Analyze full trace recursively
		ProcessTrace(&trace.ExecutionTrace, &trace.MsgCid, ethLogs, &operations, height, &discoveredAddresses, key, lib)
		if len(operations) > 0 {
			// Add the corresponding "Fee" operation
			if !trace.GasCost.TotalCost.NilOrZero() {
				opStatus := rosetta.OperationStatusOk

				operations = AppendOp(operations, tools.TotalFeeOp, trace.Msg.From.String(),
					trace.GasCost.TotalCost.Neg().String(), opStatus, false, nil)

				operations = AppendOp(operations, tools.OverEstimationBurnOp, trace.Msg.From.String(),
					trace.GasCost.OverEstimationBurn.Neg().String(), opStatus, false, nil)

				operations = AppendOp(operations, tools.MinerFeeOp, trace.Msg.From.String(),
					trace.GasCost.MinerTip.Neg().String(), opStatus, false, nil)

				operations = AppendOp(operations, tools.BurnFeeOp, trace.Msg.From.String(),
					trace.GasCost.BaseFeeBurn.Neg().String(), opStatus, false, nil)
			}

			transactions = append(transactions, &rosettaTypes.Transaction{
				TransactionIdentifier: &rosettaTypes.TransactionIdentifier{
					Hash: trace.MsgCid.String(),
				},
				Operations: operations,
			})
		}
	}
	return &transactions, &discoveredAddresses
}

func BuildFee(states *api.ComputeStateOutput, height int64, key filTypes.TipSetKey, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) *[]types.TransactionFeeInfo {
	var fees []types.TransactionFeeInfo

	for i := range states.Trace {
		trace := states.Trace[i]

		if trace.Msg == nil {
			continue
		}

		if trace.MsgRct.ExitCode.IsError() {
			continue
		}

		baseMethod, err := tools.GetMethodName(trace.Msg, height, key, lib)
		if err != nil {
			rosetta.Logger.Error("could not get method name. Error:", err.Message, err.Details)
			continue
		}

		if !tools.IsOpSupported(baseMethod) {
			continue
		}

		fee := types.TransactionFeeInfo{
			TxHash:      trace.MsgCid.String(),
			MethodName:  baseMethod,
			TotalCost:   trace.GasCost.TotalCost.Uint64(),
			GasUsage:    trace.GasCost.GasUsed.Uint64(),
			GasLimit:    trace.Msg.GasLimit,
			GasPremium:  trace.Msg.GasPremium.Uint64(),
			BaseFeeBurn: trace.GasCost.BaseFeeBurn.Uint64(),
		}

		fees = append(fees, fee)
	}

	return &fees
}

func ProcessTrace(trace *filTypes.ExecutionTrace, mainMsgCid *cid.Cid, ethLogs []EthLog, operations *[]*rosettaTypes.Operation,
	height int64, addresses *types.AddressInfoMap, key filTypes.TipSetKey, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) {

	if trace.Msg == nil {
		return
	}

	opStatus := rosetta.OperationStatusFailed
	if trace.MsgRct.ExitCode.IsSuccess() {
		opStatus = rosetta.OperationStatusOk
	}

	baseMethod, err := tools.GetMethodName(trace.Msg, height, key, lib)
	if err != nil {
		rosetta.Logger.Error("could not get method name. Error:", err.Message, err.Details)
		baseMethod = "unknown"
	}

	fromAdd := tools.GetActorAddressInfo(trace.Msg.From, height, key, lib)
	toAdd := tools.GetActorAddressInfo(trace.Msg.To, height, key, lib)
	appendAddressInfo(addresses, fromAdd, toAdd)

	metadata := make(map[string]interface{})
	metadata["GasUsed"] = trace.MsgRct.GasUsed

	switch baseMethod {
	case "InvokeContract", "InvokeContractReadOnly", "InvokeContractDelegate":
		{

			metadata["Params"] = "0x" + hex.EncodeToString(trace.Msg.Params)
			metadata["Return"] = "0x" + hex.EncodeToString(trace.MsgRct.Return)

			err, logs := searchEthLogs(ethLogs, trace.Msg)
			if err != nil {
				rosetta.Logger.Error("could not get ethLogs:", err.Error())
			}
			metadata["ethLogs"] = logs

			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, &metadata)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, &metadata)
		}
	case "Create", "Create2":
		{
			var result eam.CreateReturn
			r := bytes.NewReader(trace.MsgRct.Return)
			if err := result.UnmarshalCBOR(r); err != nil {
				zap.S().Errorf("error unmarshaling return value for method '%s': %v", baseMethod, err)
				break
			}

			ethHash, err := ethtypes.EthHashFromCid(*mainMsgCid)
			if err != nil {
				zap.S().Errorf("error getting eth hash from cid for methos '%s': %v", baseMethod, err)
				break
			}

			metadata["robustAdd"] = result.RobustAddress.String()
			metadata["ethAdd"] = "0x" + hex.EncodeToString(result.EthAddress[:])
			metadata["cid"] = mainMsgCid.String()
			metadata["ethHash"] = ethHash.String()

			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, &metadata)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, &metadata)
		}
	case "AddBalance":
		{
			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, nil)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, nil)
		}
	case "Send":
		{
			metadata["Params"] = trace.Msg.Params

			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, &metadata)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, &metadata)
		}
	case "CreateMiner":
		{
			createdActor, err := searchForActorCreation(trace.Msg, trace.MsgRct, height, key, lib)
			if err != nil {
				rosetta.Logger.Errorf("Could not parse 'CreateMiner' params, err: %v", err)
				break
			}
			appendAddressInfo(addresses, *createdActor)
		}
	case "Exec":
		{
			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, nil)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, nil)

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

				*operations = AppendOp(*operations, "Exec", from,
					trace.Msg.Value.Neg().String(), opStatus, true, nil)
				*operations = AppendOp(*operations, "Exec", to,
					trace.Msg.Value.String(), opStatus, true, nil)
			}
		}
	case "Propose":
		{
			params, err := ParseProposeParams(trace.Msg, height, key, lib)
			if err != nil {
				rosetta.Logger.Errorf("Could not parse message params for %v, error: %v", baseMethod, err.Error())
				break
			}

			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				"0", opStatus, false, &params)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				"0", opStatus, true, &params)
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
							*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
								"0", opStatus, false, &paramsMap)
							*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
								"0", opStatus, true, &paramsMap)
						}
					case "AddSigner", "RemoveSigner":
						{
							*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
								"0", opStatus, false, &paramsMap)
							*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
								"0", opStatus, true, &paramsMap)
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
			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, nil)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, nil)
		}
	case "Approve", "Cancel":
		{
			*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
				trace.Msg.Value.Neg().String(), opStatus, false, nil)
			*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
				trace.Msg.Value.String(), opStatus, true, nil)
		}
	}

	// Only process sub-calls if the parent call was successfully executed
	if opStatus == rosetta.OperationStatusOk {
		for i := range trace.Subcalls {
			subTrace := trace.Subcalls[i]
			ProcessTrace(&subTrace, mainMsgCid, ethLogs, operations, height, addresses, key, lib)
		}
	}
}

func searchEthLogs(logs []EthLog, msg *filTypes.Message) (error, []EthLog) {
	ethHash, err := ethtypes.EthHashFromCid(msg.Cid())
	if err != nil {
		return err, nil
	}

	res := make([]EthLog, 0)
	for _, log := range logs {
		if log["transactionHash"] == ethHash.String() {
			res = append(res, log)
		}
	}

	return nil, res
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
