package parser

import (
	"bytes"
	"errors"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/reward"
)

func (p *Parser) parseReward(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "AwardBlockReward":
		return p.awardBlockReward(msg.Params)
	case "UpdateNetworkKPI":
		metadata := make(map[string]interface{})
		reader := bytes.NewReader(msg.Params)
		var blockRewards reward.State
		err := blockRewards.UnmarshalCBOR(reader)
		if err != nil {
			return metadata, err
		}
		metadata["Params"] = blockRewards
		return metadata, nil
	case "ThisEpochReward":
		metadata := make(map[string]interface{})
		reader := bytes.NewReader(msgRct.Return)
		var epochRewards reward.ThisEpochRewardReturn
		err := epochRewards.UnmarshalCBOR(reader)
		if err != nil {
			return metadata, err
		}
		metadata["Params"] = epochRewards
		return metadata, nil
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) awardBlockReward(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var blockRewards reward.AwardBlockRewardParams
	err := blockRewards.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = blockRewards
	return metadata, nil
}
