package parser

import (
	"encoding/json"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
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

func BuildTransactions(states *api.ComputeStateOutput) (*[]*rosettaTypes.Transaction, *types.AddressInfoMap) {
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
		ProcessTrace(&trace.ExecutionTrace, &operations, &discoveredAddresses)
		if len(operations) > 0 {
			// Add the corresponding "Fee" operation
			if !trace.GasCost.TotalCost.Nil() {
				opStatus := rosetta.OperationStatusOk
				operations = AppendOp(operations, "Fee", trace.Msg.From.String(),
					trace.GasCost.TotalCost.Neg().String(), opStatus, false, nil)
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

func BuildFee(states *api.ComputeStateOutput) *[]types.TransactionFeeInfo {
	var fees []types.TransactionFeeInfo

	for i := range states.Trace {
		trace := states.Trace[i]

		if trace.Msg == nil {
			continue
		}

		if trace.MsgRct.ExitCode.IsError() {
			continue
		}

		baseMethod, err := tools.GetMethodName(trace.Msg)
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

func ProcessTrace(trace *filTypes.ExecutionTrace, operations *[]*rosettaTypes.Operation, addresses *types.AddressInfoMap) {

	if trace.Msg == nil {
		return
	}

	baseMethod, err := tools.GetMethodName(trace.Msg)
	if err != nil {
		rosetta.Logger.Error("could not get method name. Error:", err.Message, err.Details)
		baseMethod = "unknown"
	}

	fromAdd := tools.GetActorAddressInfo(trace.Msg.From)
	toAdd := tools.GetActorAddressInfo(trace.Msg.To)
	appendAddressInfo(addresses, fromAdd, toAdd)

	if tools.IsOpSupported(baseMethod) {
		opStatus := rosetta.OperationStatusFailed
		if trace.MsgRct.ExitCode.IsSuccess() {
			opStatus = rosetta.OperationStatusOk
		}

		switch baseMethod {
		case "Send", "AddBalance":
			{
				*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
					trace.Msg.Value.Neg().String(), opStatus, false, nil)
				*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
					trace.Msg.Value.String(), opStatus, true, nil)
			}

		case "CreateMiner":
			{
				createdActor, err := searchForActorCreation(trace.Msg, trace.MsgRct)
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
				createdActor, err := searchForActorCreation(trace.Msg, trace.MsgRct)
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
				if rosetta.GetActorNameFromCid(createdActor.ActorCid) == "multisig" &&
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
				params, err := ParseProposeParams(trace.Msg)
				if err != nil {
					rosetta.Logger.Error("Could not parse message params for", baseMethod)
					break
				}

				*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
					"0", opStatus, false, &params)
				*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
					"0", opStatus, true, &params)
			}
		case "SwapSigner", "AddSigner", "RemoveSigner":
			{
				params, err := ParseMsigParams(trace.Msg)
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
			"PreCommitSector", "ProveCommitSector", "SubmitWindowedPoSt":
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
	}

	for i := range trace.Subcalls {
		subTrace := trace.Subcalls[i]
		ProcessTrace(&subTrace, operations, addresses)
	}
}

func searchForActorCreation(msg *filTypes.Message, receipt *filTypes.MessageReceipt) (*types.AddressInfo, error) {

	toAddressInfo := tools.GetActorAddressInfo(msg.To)
	actorName := rosetta.GetActorNameFromCid(toAddressInfo.ActorCid)
	switch actorName {
	case "init":
		{
			params, err := ParseInitActorExecParams(msg.Params)
			if err != nil {
				return nil, err
			}
			createdActorName := rosetta.GetActorNameFromCid(params.CodeCID)
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
