package services

import (
	"context"
	"github.com/coinbase/rosetta-sdk-go/server"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/parser"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	rosettaTools "github.com/zondax/rosetta-filecoin-proxy/rosetta/tools"
	"time"
)

const (
	// LotusCallTimeOut TimeOut for RPC Lotus calls
	LotusCallTimeOut = 60 * 4 * time.Second

	// BlockCIDsKey is the name of the key in the Metadata map inside a
	// BlockResponse that specifies blocks' CIDs inside a TipSet.
	BlockCIDsKey = "blockCIDs"

	// DiscoveredAddressesKey is the name of the key in the Metadata map inside a
	// BlockResponse that specifies the AddressInfo of actors that participated on transactions.
	DiscoveredAddressesKey = "DiscoveredAddresses"
)

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	network *rosettaTypes.NetworkIdentifier
	node    api.FullNode
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(network *rosettaTypes.NetworkIdentifier, api *api.FullNode) server.BlockAPIServicer {
	return &BlockAPIService{
		network: network,
		node:    *api,
	}
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *rosettaTypes.BlockRequest,
) (*rosettaTypes.BlockResponse, *rosettaTypes.Error) {

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

	rosetta.Logger.Infof("/block - requested index %d", requestedHeight)

	var tipSet *filTypes.TipSet
	var err error
	impl := func() {
		tipSet, err = s.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(requestedHeight), filTypes.EmptyTSK)
	}

	errTimeOut := rosettaTools.WrapWithTimeout(impl, LotusCallTimeOut)
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
		return &rosettaTypes.BlockResponse{}, nil
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
		errTimeOut = rosettaTools.WrapWithTimeout(impl, LotusCallTimeOut)
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
	var transactions *[]*rosettaTypes.Transaction
	var discoveredAddresses *types.AddressInfoMap
	if requestedHeight > 1 {
		states, err := getLotusStateCompute(ctx, &s.node, tipSet)
		if err != nil {
			return nil, err
		}
		transactions, discoveredAddresses = buildTransactions(states)
	}

	// Add block metadata
	md := make(map[string]interface{})
	var blockCIDs []string
	for _, cid := range tipSet.Cids() {
		blockCIDs = append(blockCIDs, cid.String())
	}
	md[BlockCIDsKey] = blockCIDs
	if discoveredAddresses != nil {
		md[DiscoveredAddressesKey] = *discoveredAddresses
	}

	hashTipSet, err := rosetta.BuildTipSetKeyHash(tipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToBuildTipSetHash, nil, true)
	}
	blockId := &rosettaTypes.BlockIdentifier{
		Index: int64(tipSet.Height()),
		Hash:  *hashTipSet,
	}

	parentBlockId := &rosettaTypes.BlockIdentifier{}
	hashParentTipSet, err := rosetta.BuildTipSetKeyHash(parentTipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToBuildTipSetHash, nil, true)
	}
	parentBlockId.Index = int64(parentTipSet.Height())
	parentBlockId.Hash = *hashParentTipSet

	respBlock := &rosettaTypes.Block{
		BlockIdentifier:       blockId,
		ParentBlockIdentifier: parentBlockId,
		Timestamp:             int64(tipSet.MinTimestamp()) * rosetta.FactorSecondToMillisecond, // [ms]
		Metadata:              md,
	}
	if transactions != nil {
		respBlock.Transactions = *transactions
	}

	resp := &rosettaTypes.BlockResponse{
		Block: respBlock,
	}

	return resp, nil
}

func buildTransactions(states *api.ComputeStateOutput) (*[]*rosettaTypes.Transaction, *types.AddressInfoMap) {
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
		parser.ProcessTrace(&trace.ExecutionTrace, &operations, &discoveredAddresses)
		if len(operations) > 0 {
			// Add the corresponding "Fee" operation
			if !trace.GasCost.TotalCost.Nil() {
				opStatus := rosetta.OperationStatusOk
				operations = parser.AppendOp(operations, "Fee", trace.Msg.From.String(),
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

func getLotusStateCompute(ctx context.Context, node *api.FullNode, tipSet *filTypes.TipSet) (*api.ComputeStateOutput, *rosettaTypes.Error) {
	defer rosetta.TimeTrack(time.Now(), "[Lotus]StateCompute")

	// StateCompute includes the messages at height N-1.
	// So, we're getting the traces of the messages created at N-1, executed at N
	states, err := (*node).StateCompute(ctx, tipSet.Height(), nil, tipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}
	return states, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *rosettaTypes.BlockTransactionRequest,
) (*rosettaTypes.BlockTransactionResponse, *rosettaTypes.Error) {
	return nil, rosetta.ErrNotImplemented
}
