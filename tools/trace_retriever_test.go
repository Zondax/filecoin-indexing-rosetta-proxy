package tools_test

import (
	"context"
	"errors"
	"testing"

	ds "github.com/Zondax/zindexer/components/connections/data_store"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tests/mocks"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

var testCid cid.Cid

func TestMain(m *testing.M) {
	c, err := cid.Decode("bafyreicmaj5hhoy5mgqvamfhgexxyergw7hdeshizghodwkjg6qmpoco7i")
	if err != nil {
		panic(err)
	}
	testCid = c
	tools.ConnectedToLotusVersion = "v1.26.0"
	m.Run()

}
func TestGetStateCompute(t *testing.T) {
	tb := []struct {
		name        string
		useCache    bool
		bucket      string
		wantErrCode int32
		want        *tools.ComputeStateVersioned
		mockFn      func(*testing.T, *tools.TraceRetriever) (*filTypes.TipSet, api.FullNode)
	}{
		{
			name:        "get stored state compute error getting file from DS",
			useCache:    true,
			bucket:      "test-1",
			wantErrCode: rosetta.ErrUnableToGetTrace.Code,
			mockFn: func(t *testing.T, traceReceiver *tools.TraceRetriever) (*filTypes.TipSet, api.FullNode) {
				blks := []*filTypes.BlockHeader{
					testBlockHeader(t),
				}
				ts, err := filTypes.NewTipSet(blks)
				assert.NoError(t, err)

				dsMock := &mocks.DataStoreMock{}
				dsMock.On("GetFile", "traces_85919298723.json", mock.Anything).Return(
					nil, errors.New("test error"),
				).Once()

				traceReceiver.DataStoreClient = ds.DataStoreClient{Client: dsMock}

				return ts, nil
			},
		},
		{
			name:        "get stored state compute error unmarshalling the trace",
			useCache:    true,
			bucket:      "test-1",
			wantErrCode: rosetta.ErrUnableToGetTrace.Code,
			mockFn: func(t *testing.T, traceReceiver *tools.TraceRetriever) (*filTypes.TipSet, api.FullNode) {
				blks := []*filTypes.BlockHeader{
					testBlockHeader(t),
				}
				ts, err := filTypes.NewTipSet(blks)
				assert.NoError(t, err)

				dsMock := &mocks.DataStoreMock{}
				dsMock.On("GetFile", "traces_85919298723.json", mock.Anything).Return(
					[]byte(`{
						"Root": "invalid",
						"Trace": [
							{
								"MsgCid":"invalid"
							}
						],
						"LotusVersion": "v1.26.0"

					}`), nil,
				).Once()
				traceReceiver.DataStoreClient = ds.DataStoreClient{Client: dsMock}
				return ts, nil
			},
		},
		{
			name:        "get lotus state compute error",
			useCache:    false,
			bucket:      "test-1",
			wantErrCode: rosetta.ErrUnableToGetTrace.Code,
			mockFn: func(t *testing.T, _ *tools.TraceRetriever) (*filTypes.TipSet, api.FullNode) {
				blks := []*filTypes.BlockHeader{
					testBlockHeader(t),
				}
				ts, err := filTypes.NewTipSet(blks)
				assert.NoError(t, err)

				fullNodeMock := &mocks.FullNodeMock{}
				fullNodeMock.On("StateCompute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Once().Return(nil, errors.New("test error"))
				return ts, fullNodeMock
			},
		},
		{
			name:     "get lotus state compute success",
			useCache: false,
			bucket:   "test-1",
			want: &tools.ComputeStateVersioned{
				Root:         testCid,
				Trace:        []*api.InvocResult{{MsgCid: testCid}},
				LotusVersion: "v1.26.0",
			},
			mockFn: func(t *testing.T, _ *tools.TraceRetriever) (*filTypes.TipSet, api.FullNode) {
				blks := []*filTypes.BlockHeader{
					testBlockHeader(t),
				}
				ts, err := filTypes.NewTipSet(blks)
				assert.NoError(t, err)

				fullNodeMock := &mocks.FullNodeMock{}
				fullNodeMock.On("StateCompute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Once().Return(&api.ComputeStateOutput{
					Root: testCid,
					Trace: []*api.InvocResult{
						{MsgCid: testCid},
					},
				}, nil)
				return ts, fullNodeMock
			},
		},
		{
			name:     "get stored state compute success",
			useCache: true,
			bucket:   "test-1",
			want: &tools.ComputeStateVersioned{
				Root:         testCid,
				Trace:        []*api.InvocResult{{MsgCid: testCid}},
				LotusVersion: "v1.26.0",
			},
			mockFn: func(t *testing.T, traceReceiver *tools.TraceRetriever) (*filTypes.TipSet, api.FullNode) {
				blks := []*filTypes.BlockHeader{
					testBlockHeader(t),
				}
				ts, err := filTypes.NewTipSet(blks)
				assert.NoError(t, err)

				dsMock := &mocks.DataStoreMock{}
				dsMock.On("GetFile", "traces_85919298723.json", mock.Anything).Return(
					[]byte(`{
						"Root": {"/":"bafyreicmaj5hhoy5mgqvamfhgexxyergw7hdeshizghodwkjg6qmpoco7i"},
						"Trace": [
							{
								"MsgCid":{"/":"bafyreicmaj5hhoy5mgqvamfhgexxyergw7hdeshizghodwkjg6qmpoco7i"}
							}
						],
						"LotusVersion": "v1.26.0"

					}`), nil,
				).Once()

				traceReceiver.DataStoreClient = ds.DataStoreClient{Client: dsMock}

				return ts, nil
			},
		},
	}

	for _, tt := range tb {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			traceReceiver := tools.NewTraceRetriever(tt.useCache, tt.bucket, ds.DataStoreConfig{Service: "local", S3MaxRetries: 0})

			tipset, fullNodeMock := tt.mockFn(t, traceReceiver)
			got, gotErr := traceReceiver.GetStateCompute(ctx, &fullNodeMock, tipset)
			if tt.wantErrCode > 0 {
				assert.NotNil(t, gotErr)
				assert.Equal(t, tt.wantErrCode, gotErr.Code)
				return
			}
			assert.Nil(t, gotErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

// this is how lotus creates test block headers
func testBlockHeader(t *testing.T) *filTypes.BlockHeader {
	t.Helper()

	addr, err := address.NewIDAddress(12512063)
	if err != nil {
		t.Fatal(err)
	}

	return &filTypes.BlockHeader{
		Miner: addr,
		Ticket: &filTypes.Ticket{
			VRFProof: []byte("vrf proof0000000vrf proof0000000"),
		},
		ElectionProof: &filTypes.ElectionProof{
			VRFProof: []byte("vrf proof0000000vrf proof0000000"),
		},
		Parents:               []cid.Cid{testCid, testCid},
		ParentMessageReceipts: testCid,
		BLSAggregate:          &crypto.Signature{Type: crypto.SigTypeBLS, Data: []byte("boo! im a signature")},
		ParentWeight:          filTypes.NewInt(123125126212),
		Messages:              testCid,
		Height:                85919298723,
		ParentStateRoot:       testCid,
		BlockSig:              &crypto.Signature{Type: crypto.SigTypeBLS, Data: []byte("boo! im a signature")},
		ParentBaseFee:         filTypes.NewInt(3432432843291),
	}
}
