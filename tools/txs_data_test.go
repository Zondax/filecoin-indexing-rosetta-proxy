package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	parserTypes "github.com/zondax/fil-parser/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func TestNodeInfoFromLotusVersion(t *testing.T) {
	tb := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{name: "lotus Version() output", version: "1.37.0-rc1+calibnet+git.15d0c30", want: "v1.37"},
		{name: "mainnet release", version: "1.34.1+mainnet+git.5a8ed2a", want: "v1.34"},
		{name: "v-prefixed", version: "v1.26.0", want: "v1.26"},
		{name: "no build metadata", version: "1.37.0", want: "v1.37"},
		{name: "unknown", version: tools.UnknownStr, wantErr: true},
		{name: "empty", version: "", wantErr: true},
	}

	for _, tt := range tb {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tools.NodeInfoFromLotusVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.NodeMajorMinorVersion)
			assert.Equal(t, tt.version, got.NodeFullVersion)
		})
	}
}

func TestBuildTxsData(t *testing.T) {
	ts, err := filTypes.NewTipSet([]*filTypes.BlockHeader{testBlockHeader(t)})
	require.NoError(t, err)
	extendedTipset, err := tools.ToExtendedTipSet(ts)
	require.NoError(t, err)
	assert.Equal(t, ts.Height(), extendedTipset.Height())

	states := &tools.ComputeStateVersioned{
		Root:         testCid,
		Trace:        []*api.InvocResult{{MsgCid: testCid}},
		LotusVersion: "1.37.0-rc1+calibnet+git.15d0c30",
	}

	txsData, err := tools.BuildTxsData(states, extendedTipset)
	require.NoError(t, err)
	assert.Equal(t, extendedTipset, txsData.Tipset)
	assert.Equal(t, parserTypes.BlockMetadata{NodeInfo: parserTypes.NodeInfo{
		NodeFullVersion:       "1.37.0-rc1+calibnet+git.15d0c30",
		NodeMajorMinorVersion: "v1.37",
	}}, txsData.Metadata)

	// Traces must have the StateCompute output shape (an object with Root and Trace),
	// which is what fil-parser decodes, not a bare trace array.
	var decoded api.ComputeStateOutput
	require.NoError(t, json.Unmarshal(txsData.Traces, &decoded))
	assert.Equal(t, testCid, decoded.Root)
	require.Len(t, decoded.Trace, 1)
	assert.Equal(t, testCid, decoded.Trace[0].MsgCid)

	_, err = tools.BuildTxsData(nil, extendedTipset)
	assert.Error(t, err)

	_, err = tools.BuildTxsData(&tools.ComputeStateVersioned{LotusVersion: tools.UnknownStr}, extendedTipset)
	assert.Error(t, err)
}
