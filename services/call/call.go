package call

import (
	"context"
	"github.com/coinbase/rosetta-sdk-go/server"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/lotus/api"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

type CallAPIService struct {
	network        *rosettaTypes.NetworkIdentifier
	node           api.FullNode
	traceRetriever *tools.TraceRetriever
}

// NewCallAPIService creates a new instance of a CallAPIService.
func NewCallAPIService(network *rosettaTypes.NetworkIdentifier, api *api.FullNode, retriever *tools.TraceRetriever) server.CallAPIServicer {
	return &CallAPIService{
		network:        network,
		node:           *api,
		traceRetriever: retriever,
	}
}

func (s *CallAPIService) Call(ctx context.Context, request *rosettaTypes.CallRequest) (*rosettaTypes.CallResponse, *rosettaTypes.Error) {
	errNet := rosetta.ValidateNetworkId(ctx, &s.node, request.NetworkIdentifier)
	if errNet != nil {
		return nil, errNet
	}

	switch request.Method {
	case StateComputeCall:
		return s.StateComputeVersioned(ctx, request)
	default:
		return nil, rosetta.BuildError(rosetta.ErrOperationNotSupported, nil, true)
	}
}
