package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	initActor "github.com/filecoin-project/specs-actors/v4/actors/builtin/init"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	filLib "github.com/zondax/rosetta-filecoin-lib"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

func appendAddressInfo(arr *[]types.AddressInfo, info ...types.AddressInfo) {
	for _, i := range info {
		if i.Robust != "" && i.Short != "" {
			*arr = append(*arr, i)
		}
	}
}

func ProcessTrace(trace *filTypes.ExecutionTrace, operations *[]*rosettaTypes.Operation, addresses *[]types.AddressInfo) {

	if trace.Msg == nil {
		return
	}

	baseMethod, err := tools.GetMethodName(trace.Msg)
	if err != nil {
		rosetta.Logger.Error("could not get method name. Error:", err.Message, err.Details)
		baseMethod = "unknown"
	}

	fromAdd, err1 := tools.GetActorAddressInfo(trace.Msg.From)
	toAdd, err2 := tools.GetActorAddressInfo(trace.Msg.To)
	if err1 != nil || err2 != nil {
		rosetta.Logger.Error("could not retrieve one or both pubkeys for addresses:",
			trace.Msg.From.String(), trace.Msg.To.String())
	}

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
		case "Exec":
			{
				*operations = AppendOp(*operations, baseMethod, fromAdd.GetAddress(),
					trace.Msg.Value.Neg().String(), opStatus, false, nil)
				*operations = AppendOp(*operations, baseMethod, toAdd.GetAddress(),
					trace.Msg.Value.String(), opStatus, true, nil)

				// Check if this Exec contains actor creation event
				createdActor, err := searchForActorCreated(trace.Msg, trace.MsgRct)
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
				params, err := parseProposeParams(trace.Msg)
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
				params, err := parseMsigParams(trace.Msg)
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

func searchForActorCreated(msg *filTypes.Message, receipt *filTypes.MessageReceipt) (*types.AddressInfo, error) {

	toAddressInfo, err := tools.GetActorAddressInfo(msg.To)
	if err != nil {
		return nil, err
	}

	switch rosetta.GetActorNameFromCid(toAddressInfo.ActorCid) {
	case "init":
		{
			reader := bytes.NewReader(msg.Params)
			var params initActor.ExecParams
			err := params.UnmarshalCBOR(reader)
			if err != nil {
				return nil, err
			}
			createdActorName := rosetta.GetActorNameFromCid(params.CodeCID)
			switch createdActorName {
			case "multisig", "paymentchannel":
				{
					reader = bytes.NewReader(receipt.Return)
					var execReturn initActor.ExecReturn
					err = execReturn.UnmarshalCBOR(reader)
					if err != nil {
						return nil, err
					}

					info := types.AddressInfo{
						Short:    execReturn.IDAddress.String(),
						Robust:   execReturn.RobustAddress.String(),
						ActorCid: params.CodeCID,
					}
					return &info, nil
				}
			default:
				return nil, nil
			}
		}
	default:
		return nil, nil
	}
}

func parseProposeParams(msg *filTypes.Message) (map[string]interface{}, error) {
	r := &filLib.RosettaConstructionFilecoin{}
	var params map[string]interface{}
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return params, err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To)
	if err != nil {
		return params, err
	}

	if !builtin.IsMultisigActor(actorCode) {
		return params, fmt.Errorf("this id doesn't correspond to a multisig actor")
	}

	innerMethod, parsedParams, err := r.ParseProposeTxParams(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse params. ParseProposeTxParams returned with error:", err.Error())
		return params, err
	}

	err = json.Unmarshal([]byte(parsedParams), &params)
	if err != nil {
		return params, err
	}

	params["Method"] = innerMethod
	return params, nil
}

func parseMsigParams(msg *filTypes.Message) (string, error) {
	r := &filLib.RosettaConstructionFilecoin{}
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return "", err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To)
	if err != nil {
		return "", err
	}

	if !builtin.IsMultisigActor(actorCode) {
		return "", fmt.Errorf("this id doesn't correspond to a multisig actor")
	}

	parsedParams, err := r.ParseParamsMultisigTx(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse params. ParseParamsMultisigTx returned with error:", err.Error())
		return "", err
	}

	return parsedParams, nil
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
