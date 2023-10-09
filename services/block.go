package services

import (
	"context"
	"encoding/json"
	"github.com/coinbase/rosetta-sdk-go/server"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	filparser "github.com/zondax/fil-parser"
	"github.com/zondax/fil-parser/actors/cache/impl/common"
	parserTypes "github.com/zondax/fil-parser/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	filLib "github.com/zondax/rosetta-filecoin-lib"
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
	network        *rosettaTypes.NetworkIdentifier
	node           api.FullNode
	traceRetriever *tools.TraceRetriever
	rosettaLib     *filLib.RosettaConstructionFilecoin
	p              *filparser.FilecoinParser
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(network *rosettaTypes.NetworkIdentifier, api *api.FullNode,
	retriever *tools.TraceRetriever, r *filLib.RosettaConstructionFilecoin) server.BlockAPIServicer {
	parser, _ := filparser.NewFilecoinParser(r, common.DataSource{}, nil) //TODO: Check this error
	return &BlockAPIService{
		network:        network,
		node:           *api,
		traceRetriever: retriever,
		rosettaLib:     r,
		p:              parser,
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
	var (
		parsedTraces        []*parserTypes.Transaction
		transactions        []*rosettaTypes.Transaction
		discoveredAddresses *parserTypes.AddressInfoMap
		parseError          error
	)

	if requestedHeight > 1 {
		states, err := s.traceRetriever.GetStateCompute(ctx, &s.node, tipSet)
		if err != nil {
			return nil, err
		}

		tracesBytes, marshalErr := json.Marshal(states.Trace)
		if marshalErr != nil {
			return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, marshalErr, true)
		}

		// TODO: uncomment for wallaby
		// ethLogs, err := s.traceRetriever.GetEthLogs(ctx, &s.node, tipSet)
		// if err != nil {
		//	 return nil, err
		// }

		extendedTipset := &parserTypes.ExtendedTipSet{}
		tipsetBytes, marshalErr := json.Marshal(tipSet)
		if marshalErr != nil {
			return nil, rosetta.BuildError(rosetta.ErrUnableToGetTipset, marshalErr, true) //TODO: Move to the part of code where rosetta asks for the tipset
		}

		unmarshalErr := extendedTipset.UnmarshalJSON(tipsetBytes)
		if unmarshalErr != nil {
			return nil, rosetta.BuildError(rosetta.ErrUnableToGetTipset, unmarshalErr, true) //TODO: Move to the part of code where rosetta asks for the tipset
		}

		parsedTraces, discoveredAddresses, parseError = s.p.ParseTransactions(tracesBytes, extendedTipset, nil, nil) // TODO: fill with ethLogs
		if parseError != nil {
			return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, parseError, true)
		}
		transactions = tools.ToRosetta(parsedTraces)
	}

	// Add block metadata
	md := make(map[string]interface{})
	var blockCIDs []string
	for _, cid := range tipSet.Cids() {
		blockCIDs = append(blockCIDs, cid.String())
	}
	md[BlockCIDsKey] = blockCIDs
	if discoveredAddresses != nil {
		md[DiscoveredAddressesKey] = discoveredAddresses.Copy()
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
		respBlock.Transactions = transactions
	}

	resp := &rosettaTypes.BlockResponse{
		Block: respBlock,
	}

	return resp, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *rosettaTypes.BlockTransactionRequest,
) (*rosettaTypes.BlockTransactionResponse, *rosettaTypes.Error) {
	return nil, rosetta.ErrNotImplemented
}
