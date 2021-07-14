package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	initActor "github.com/filecoin-project/specs-actors/v4/actors/builtin/init"
	filLib "github.com/zondax/rosetta-filecoin-lib"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"github.com/zondax/rosetta-filecoin-proxy/rosetta/tools"
	"time"
)

// LotusCallTimeOut TimeOut for RPC Lotus calls
const LotusCallTimeOut = 60 * 4 * time.Second

// BlockCIDsKey is the name of the key in the Metadata map inside a
// BlockResponse that specifies blocks' CIDs inside a TipSet.
const BlockCIDsKey = "blockCIDs"

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	network *types.NetworkIdentifier
	node    api.FullNode
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(network *types.NetworkIdentifier, api *api.FullNode) server.BlockAPIServicer {
	return &BlockAPIService{
		network: network,
		node:    *api,
	}
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {

	if request.BlockIdentifier == nil {
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	if request.BlockIdentifier == nil && request.BlockIdentifier.Hash == nil {
		return nil, rosetta.BuildError(rosetta.ErrInsufficientQueryInputs, nil, true)
	}

	errNet := rosetta.ValidateNetworkId(ctx, &s.node, request.NetworkIdentifier)
	if errNet != nil {
		return nil, errNet
	}

	requestedHeight := *request.BlockIdentifier.Index
	if requestedHeight < 0 {
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	// Check sync status
	status, syncErr := rosetta.CheckSyncStatus(ctx, &s.node)
	if syncErr != nil {
		return nil, syncErr
	}
	if requestedHeight > 0 && !status.IsSynced() {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetUnsyncedBlock, nil, true)
	}

	if request.BlockIdentifier.Index == nil {
		return nil, rosetta.BuildError(rosetta.ErrInsufficientQueryInputs, nil, true)
	}

	var tipSet *filTypes.TipSet
	var err error
	impl := func() {
		tipSet, err = s.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(requestedHeight), filTypes.EmptyTSK)
	}

	errTimeOut := tools.WrapWithTimeout(impl, LotusCallTimeOut)
	if errTimeOut != nil {
		return nil, rosetta.ErrLotusCallTimedOut
	}

	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTipset, err, true)
	}

	// If a TipSet has empty blocks, lotus api will return a TipSet at a different epoch
	// Check if the retrieved TipSet is actually the requested one
	// details on: https://github.com/filecoin-project/lotus/blob/49d64f7f7e22973ca0cfbaaf337fcfb3c2d47707/api/api_full.go#L65-L67
	if int64(tipSet.Height()) != requestedHeight {
		return &types.BlockResponse{}, nil
	}

	if request.BlockIdentifier.Hash != nil {
		tipSetKeyHash, encErr := rosetta.BuildTipSetKeyHash(tipSet.Key())
		if encErr != nil {
			return nil, rosetta.BuildError(rosetta.ErrUnableToBuildTipSetHash, encErr, true)
		}
		if *tipSetKeyHash != *request.BlockIdentifier.Hash {
			return nil, rosetta.BuildError(rosetta.ErrInvalidHash, nil, true)
		}
	}

	// Get parent TipSet
	var parentTipSet *filTypes.TipSet
	if requestedHeight > 0 {
		if tipSet.Parents().IsEmpty() {
			return nil, rosetta.BuildError(rosetta.ErrUnableToGetParentBlk, nil, true)
		}
		impl = func() {
			parentTipSet, err = s.node.ChainGetTipSet(ctx, tipSet.Parents())
		}
		errTimeOut = tools.WrapWithTimeout(impl, LotusCallTimeOut)
		if errTimeOut != nil {
			return nil, rosetta.ErrLotusCallTimedOut
		}
		if err != nil {
			return nil, rosetta.BuildError(rosetta.ErrUnableToGetParentBlk, err, true)
		}
	} else {
		// According to rosetta docs, if the requested tipset is
		// the genesis one, set the same tipset as parent
		parentTipSet = tipSet
	}

	// Build transactions data
	var transactions *[]*types.Transaction
	if requestedHeight > 1 {
		states, err := getLotusStateCompute(ctx, &s.node, tipSet)
		if err != nil {
			return nil, err
		}
		transactions = buildTransactions(states)
	}

	// Add block metadata
	md := make(map[string]interface{})
	var blockCIDs []string
	for _, cid := range tipSet.Cids() {
		blockCIDs = append(blockCIDs, cid.String())
	}
	md[BlockCIDsKey] = blockCIDs

	hashTipSet, err := rosetta.BuildTipSetKeyHash(tipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToBuildTipSetHash, nil, true)
	}
	blockId := &types.BlockIdentifier{
		Index: int64(tipSet.Height()),
		Hash:  *hashTipSet,
	}

	parentBlockId := &types.BlockIdentifier{}
	hashParentTipSet, err := rosetta.BuildTipSetKeyHash(parentTipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToBuildTipSetHash, nil, true)
	}
	parentBlockId.Index = int64(parentTipSet.Height())
	parentBlockId.Hash = *hashParentTipSet

	respBlock := &types.Block{
		BlockIdentifier:       blockId,
		ParentBlockIdentifier: parentBlockId,
		Timestamp:             int64(tipSet.MinTimestamp()) * rosetta.FactorSecondToMillisecond, // [ms]
		Metadata:              md,
	}
	if transactions != nil {
		respBlock.Transactions = *transactions
	}

	resp := &types.BlockResponse{
		Block: respBlock,
	}

	return resp, nil
}

func buildTransactions(states *api.ComputeStateOutput) *[]*types.Transaction {
	defer rosetta.TimeTrack(time.Now(), "[Proxy]TraceAnalysis")

	var transactions []*types.Transaction
	for i := range states.Trace {
		trace := states.Trace[i]

		if trace.Msg == nil {
			continue
		}

		var operations []*types.Operation

		// Analyze full trace recursively
		ProcessTrace(&trace.ExecutionTrace, &operations)
		if len(operations) > 0 {
			// Add the corresponding "Fee" operation
			if !trace.GasCost.TotalCost.Nil() {
				opStatus := rosetta.OperationStatusOk
				operations = appendOp(operations, "Fee", trace.Msg.From.String(),
					trace.GasCost.TotalCost.Neg().String(), opStatus, false, nil)
			}

			transactions = append(transactions, &types.Transaction{
				TransactionIdentifier: &types.TransactionIdentifier{
					Hash: trace.MsgCid.String(),
				},
				Operations: operations,
			})
		}
	}
	return &transactions
}

func getLotusStateCompute(ctx context.Context, node *api.FullNode, tipSet *filTypes.TipSet) (*api.ComputeStateOutput, *types.Error) {
	defer rosetta.TimeTrack(time.Now(), "[Lotus]StateCompute")

	// StateCompute includes the messages at height N-1.
	// So, we're getting the traces of the messages created at N-1, executed at N
	states, err := (*node).StateCompute(ctx, tipSet.Height(), nil, tipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}
	return states, nil
}

func ProcessTrace(trace *filTypes.ExecutionTrace, operations *[]*types.Operation) {

	if trace.Msg == nil {
		return
	}

	baseMethod, err := rosetta.GetMethodName(trace.Msg)
	if err != nil {
		rosetta.Logger.Error("could not get method name. Error:", err.Message, err.Details)
		baseMethod = "unknown"
	}

	if rosetta.IsOpSupported(baseMethod) {
		fromPk, err1 := rosetta.GetActorPubKey(trace.Msg.From)
		toPk, err2 := rosetta.GetActorPubKey(trace.Msg.To)
		if err1 != nil || err2 != nil {
			rosetta.Logger.Error("could not retrieve one or both pubkeys for addresses:",
				trace.Msg.From.String(), trace.Msg.To.String())
			return
		}

		opStatus := rosetta.OperationStatusFailed
		if trace.MsgRct.ExitCode.IsSuccess() {
			opStatus = rosetta.OperationStatusOk
		}

		switch baseMethod {
		case "Send", "AddBalance":
			{
				*operations = appendOp(*operations, baseMethod, fromPk,
					trace.Msg.Value.Neg().String(), opStatus, false, nil)
				*operations = appendOp(*operations, baseMethod, toPk,
					trace.Msg.Value.String(), opStatus, true, nil)
			}
		case "Exec":
			{
				*operations = appendOp(*operations, baseMethod, fromPk,
					trace.Msg.Value.Neg().String(), opStatus, false, nil)
				*operations = appendOp(*operations, baseMethod, toPk,
					trace.Msg.Value.String(), opStatus, true, nil)

				// Check if this Exec op created and funded a msig account
				params, err := parseExecParams(trace.Msg, trace.MsgRct)
				if err == nil {
					var paramsMap map[string]interface{}
					if err := json.Unmarshal([]byte(params), &paramsMap); err == nil {
						if fundedAddress, ok := paramsMap["IDAddress"]; ok {
							paramsMap["Method"] = "Send"
							fromPk = toPk                 // init actor
							toPk = fundedAddress.(string) // new msig address
							*operations = appendOp(*operations, "Exec", fromPk,
								trace.Msg.Value.Neg().String(), opStatus, false, &paramsMap)
							*operations = appendOp(*operations, "Exec", toPk,
								trace.Msg.Value.String(), opStatus, true, &paramsMap)
						}
					} else {
						rosetta.Logger.Error("Could not parse message params for", baseMethod)
					}
				}
			}
		case "Propose":
			{
				params, err := parseProposeParams(trace.Msg)
				if err != nil {
					rosetta.Logger.Error("Could not parse message params for", baseMethod)
					break
				}

				*operations = appendOp(*operations, baseMethod, fromPk,
					"0", opStatus, false, &params)
				*operations = appendOp(*operations, baseMethod, toPk,
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
								*operations = appendOp(*operations, baseMethod, fromPk,
									"0", opStatus, false, &paramsMap)
								*operations = appendOp(*operations, baseMethod, toPk,
									"0", opStatus, true, &paramsMap)
							}
						case "AddSigner", "RemoveSigner":
							{
								*operations = appendOp(*operations, baseMethod, fromPk,
									"0", opStatus, false, &paramsMap)
								*operations = appendOp(*operations, baseMethod, toPk,
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
				*operations = appendOp(*operations, baseMethod, fromPk,
					trace.Msg.Value.Neg().String(), opStatus, false, nil)
				*operations = appendOp(*operations, baseMethod, toPk,
					trace.Msg.Value.String(), opStatus, true, nil)
			}
		case "Approve", "Cancel":
			{
				*operations = appendOp(*operations, baseMethod, fromPk,
					trace.Msg.Value.Neg().String(), opStatus, false, nil)
				*operations = appendOp(*operations, baseMethod, toPk,
					trace.Msg.Value.String(), opStatus, true, nil)
			}
		}
	}

	for i := range trace.Subcalls {
		subTrace := trace.Subcalls[i]
		ProcessTrace(&subTrace, operations)
	}
}

func parseExecParams(msg *filTypes.Message, receipt *filTypes.MessageReceipt) (string, error) {

	actorName := rosetta.GetActorNameFromAddress(msg.To)

	switch actorName {
	case "init":
		{
			reader := bytes.NewReader(msg.Params)
			var params initActor.ExecParams
			err := params.UnmarshalCBOR(reader)
			if err != nil {
				return "", err
			}
			execActorName := rosetta.GetActorNameFromCid(params.CodeCID)
			switch execActorName {
			case "multisig", "paymentchannel":
				{
					reader = bytes.NewReader(receipt.Return)
					var execReturn initActor.ExecReturn
					err = execReturn.UnmarshalCBOR(reader)
					if err != nil {
						return "", err
					}
					jsonResponse, err := json.Marshal(execReturn)
					if err != nil {
						return "", err
					}
					return string(jsonResponse), nil
				}
			default:
				return "", nil
			}
		}
	default:
		return "", nil
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

	actorCode, err := tools.ActorsDB.GetActorCode(msg.To)
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

	actorCode, err := tools.ActorsDB.GetActorCode(msg.To)
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

func appendOp(ops []*types.Operation, opType string, account string, amount string, status string, relateOp bool, metadata *map[string]interface{}) []*types.Operation {
	opIndex := int64(len(ops))
	op := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{
			Index: opIndex,
		},
		Type:   opType,
		Status: &status,
		Account: &types.AccountIdentifier{
			Address: account,
		},
		Amount: &types.Amount{
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
		op.RelatedOperations = []*types.OperationIdentifier{
			{
				Index: opIndex - 1,
			},
		}
	}

	return append(ops, op)
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	return nil, rosetta.ErrNotImplemented
}
