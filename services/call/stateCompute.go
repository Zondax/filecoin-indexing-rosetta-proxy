package call

import (
	"context"
	"encoding/json"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-state-types/abi"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/services"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	rosettaTools "github.com/zondax/rosetta-filecoin-proxy/rosetta/tools"
)

const StateComputeCall = "StateCompute"

func (s *CallAPIService) StateComputeVersioned(
	ctx context.Context,
	request *rosettaTypes.CallRequest,
) (*rosettaTypes.CallResponse, *rosettaTypes.Error) {

	blockId := rosettaTypes.BlockIdentifier{}
	params, err := json.Marshal(request.Parameters)
	if err != nil {
		rosetta.Logger.Errorf("Error on request.Parameters: %s", err.Error())
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	err = json.Unmarshal(params, &blockId)
	if err != nil {
		rosetta.Logger.Errorf("Error while unmarshaling parameters: %s", err.Error())
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	requestedHeight := blockId.Index
	if requestedHeight < 0 {
		return nil, rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	rosetta.Logger.Infof(tools.ConnectedToLotusVersion)
	rosetta.Logger.Infof("/StateComputeVersioned - requested index %d", requestedHeight)

	var tipSet *filTypes.TipSet
	impl := func() {
		tipSet, err = s.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(requestedHeight), filTypes.EmptyTSK)
	}

	errTimeOut := rosettaTools.WrapWithTimeout(impl, services.LotusCallTimeOut)
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
		return nil, nil
	}

	computeStateVersioned, err2 := s.traceRetriever.GetStateCompute(ctx, &s.node, tipSet)

	if err2 != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}

	var computeStateMap map[string]interface{}
	m, _ := json.Marshal(computeStateVersioned)
	err = json.Unmarshal(m, &computeStateMap)
	if err != nil {
		return nil, nil
	}

	res := &rosettaTypes.CallResponse{
		Result: computeStateMap,
	}

	return res, nil
}
