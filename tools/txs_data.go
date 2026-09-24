package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	parserTypes "github.com/zondax/fil-parser/types"
	"golang.org/x/mod/semver"
)

// NodeInfoFromLotusVersion builds the fil-parser NodeInfo from a lotus version string,
// e.g. "1.37.0-rc1+calibnet+git.15d0c30" or "v1.26.0" -> NodeMajorMinorVersion "v1.37" / "v1.26".
// fil-parser uses NodeMajorMinorVersion to select the trace parser implementation.
func NodeInfoFromLotusVersion(fullVersion string) (parserTypes.NodeInfo, error) {
	version := strings.TrimSpace(strings.SplitN(fullVersion, "+", 2)[0])
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	majorMinor := semver.MajorMinor(version)
	if majorMinor == "" {
		return parserTypes.NodeInfo{}, fmt.Errorf("invalid lotus version format: '%s'", fullVersion)
	}

	return parserTypes.NodeInfo{
		NodeFullVersion:       fullVersion,
		NodeMajorMinorVersion: majorMinor,
	}, nil
}

// BuildTxsData builds the fil-parser input for a tipset. Traces are serialized with the
// shape of lotus' StateCompute output ({"Root": ..., "Trace": [...]}), which is what the
// fil-parser trace parsers decode, and the metadata carries the lotus version the traces
// were generated with, so fil-parser selects the matching parser implementation.
func BuildTxsData(states *ComputeStateVersioned, tipSet *parserTypes.ExtendedTipSet) (parserTypes.TxsData, error) {
	if states == nil {
		return parserTypes.TxsData{}, fmt.Errorf("nil compute state")
	}

	nodeInfo, err := NodeInfoFromLotusVersion(states.LotusVersion)
	if err != nil {
		return parserTypes.TxsData{}, err
	}

	tracesBytes, err := json.Marshal(api.ComputeStateOutput{
		Root:  states.Root,
		Trace: states.Trace,
	})
	if err != nil {
		return parserTypes.TxsData{}, err
	}

	return parserTypes.TxsData{
		Traces:   tracesBytes,
		Tipset:   tipSet,
		Metadata: parserTypes.BlockMetadata{NodeInfo: nodeInfo},
	}, nil
}

// ToExtendedTipSet converts a lotus tipset into the fil-parser ExtendedTipSet.
func ToExtendedTipSet(tipSet *filTypes.TipSet) (*parserTypes.ExtendedTipSet, error) {
	tipsetBytes, err := json.Marshal(tipSet)
	if err != nil {
		return nil, err
	}

	extendedTipset := &parserTypes.ExtendedTipSet{}
	if err = extendedTipset.UnmarshalJSON(tipsetBytes); err != nil {
		return nil, err
	}

	return extendedTipset, nil
}
