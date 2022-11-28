package types

import (
	"encoding/json"
	lotusChainTypes "github.com/filecoin-project/lotus/chain/types"
)

type LightBlockHeader struct {
	Cid        string
	BlockMiner string
}

type BlockMessages map[string][]LightBlockHeader

func (e *ExtendedTipSet) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(&struct {
		lotusChainTypes.ExpTipSet
		BlockMessages
	}{
		ExpTipSet: lotusChainTypes.ExpTipSet{
			Cids:   e.TipSet.Cids(),
			Blocks: e.TipSet.Blocks(),
			Height: e.TipSet.Height(),
		},
		BlockMessages: e.BlockMessages,
	})

	return data, err
}

func (e *ExtendedTipSet) UnmarshalJSON(data []byte) error {
	auxTipset := &struct {
		lotusChainTypes.TipSet
	}{}

	if err := json.Unmarshal(data, &auxTipset); err != nil {
		// try other way
		auxTipset := &struct {
			Tipset lotusChainTypes.TipSet
		}{}

		if err := json.Unmarshal(data, &auxTipset); err != nil {
			return err
		}
		e.TipSet = auxTipset.Tipset
	} else {
		e.TipSet = auxTipset.TipSet
	}

	auxMessages := &struct {
		BlockMessages
	}{}
	if err := json.Unmarshal(data, &auxMessages); err != nil {
		return err
	}

	e.BlockMessages = auxMessages.BlockMessages
	return nil
}

type ExtendedTipSet struct {
	lotusChainTypes.TipSet
	BlockMessages
}
