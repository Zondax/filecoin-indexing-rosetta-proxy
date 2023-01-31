package tools

import (
	"context"
	"encoding/json"
	"fmt"
	ds "github.com/Zondax/zindexer/components/connections/data_store"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/ipfs/go-cid"
	parserTypes "github.com/zondax/fil-parser/types"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"strconv"
	"time"
)

type TraceRetriever struct {
	useCachedTraces bool
	tracesBucket    string
	ds.DataStoreClient
}

type ComputeStateVersioned struct {
	Root         cid.Cid            `json:"Root"`
	Trace        []*api.InvocResult `json:"Trace"`
	LotusVersion string             `json:"LotusVersion"`
}

func NewTraceRetriever(useCache bool, bucket string, config ds.DataStoreConfig) *TraceRetriever {
	var client ds.DataStoreClient
	if useCache {
		var err error
		client, err = ds.NewDataStoreClient(config)
		if err != nil {
			panic(err)
		}
	}

	return &TraceRetriever{
		useCachedTraces: useCache,
		tracesBucket:    bucket,
		DataStoreClient: client,
	}
}

func (t *TraceRetriever) GetStateCompute(ctx context.Context, node *api.FullNode, tipSet *filTypes.TipSet) (*ComputeStateVersioned, *rosettaTypes.Error) {
	if t.useCachedTraces {
		return t.getStoredStateCompute(tipSet)
	}

	return t.getLotusStateCompute(ctx, node, tipSet)
}

func (t *TraceRetriever) getLotusStateCompute(ctx context.Context, node *api.FullNode, tipSet *filTypes.TipSet) (*ComputeStateVersioned, *rosettaTypes.Error) {
	defer rosetta.TimeTrack(time.Now(), "[Lotus]StateCompute")

	// StateCompute includes the messages at height N-1.
	// So, we're getting the traces of the messages created at N-1, executed at N
	states, err := (*node).StateCompute(ctx, tipSet.Height(), nil, tipSet.Key())
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}

	return &ComputeStateVersioned{
		Root:         states.Root,
		Trace:        states.Trace,
		LotusVersion: ConnectedToLotusVersion,
	}, nil
}

func (t *TraceRetriever) getStoredStateCompute(tipSet *filTypes.TipSet) (*ComputeStateVersioned, *rosettaTypes.Error) {
	defer rosetta.TimeTrack(time.Now(), "getStoredStateCompute")

	data, err := t.DataStoreClient.Client.GetFile(fmt.Sprintf("traces_%s.json", tipSet.Height().String()), t.tracesBucket)
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}

	// Unmarshall it
	var trace ComputeStateVersioned
	err = json.Unmarshal(*data, &trace)
	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}

	return &ComputeStateVersioned{
		Root:         trace.Root,
		Trace:        trace.Trace,
		LotusVersion: trace.LotusVersion,
	}, nil
}

func (t *TraceRetriever) GetEthLogs(ctx context.Context, node *api.FullNode, tipSet *filTypes.TipSet) ([]parserTypes.EthLog, *rosettaTypes.Error) {
	fromBlockHex := strconv.FormatUint(uint64(tipSet.Height()), 16)
	res, err := (*node).EthGetLogs(ctx, &ethtypes.EthFilterSpec{
		FromBlock: &fromBlockHex,
		ToBlock:   &fromBlockHex,
	})

	if err != nil {
		return nil, rosetta.BuildError(rosetta.ErrUnableToGetTrace, err, true)
	}

	if len(res.Results) == 0 {
		return nil, nil
	}

	logs := make([]parserTypes.EthLog, 0, len(res.Results))
	for _, result := range res.Results {
		var log parserTypes.EthLog
		log, ok := result.(parserTypes.EthLog)
		if !ok {
			return nil, rosetta.BuildError(rosetta.ErrMalformedValue, err, true)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
