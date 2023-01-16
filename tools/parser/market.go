package parser

import (
	"bytes"
	"errors"
	"github.com/filecoin-project/go-state-types/builtin/v10/market"
	filTypes "github.com/filecoin-project/lotus/chain/types"
)

func (p *Parser) parseStoragemarket(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "Constructor":
	case "AddBalance":
	case "WithdrawBalance":
		return p.withdrawBalance(msg.Params)
	case "PublishStorageDeals":
		return p.publishStorageDeals(msg.Params, msgRct.Return)
	case "VerifyDealsForActivation":
		return p.verifyDealsForActivation(msg.Params, msgRct.Return)
	case "ActivateDeals":
		return p.activateDeals(msg.Params)
	case "OnMinerSectorsTerminate":
		return p.onMinerSectorsTerminate(msg.Params)
	case "ComputeDataCommitment":
		return p.computeDataCommitment(msg.Params, msgRct.Return)
	case "CronTick":
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) withdrawBalance(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params market.WithdrawBalanceParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}

func (p *Parser) publishStorageDeals(rawParams, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(rawParams)
	var params market.PublishStorageDealsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	reader = bytes.NewReader(rawReturn)
	var publishReturn market.PublishStorageDealsReturn
	err = publishReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Return"] = publishReturn
	return metadata, nil
}

func (p *Parser) verifyDealsForActivation(rawParams, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(rawParams)
	var params market.VerifyDealsForActivationParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	reader = bytes.NewReader(rawReturn)
	var dealsReturn market.VerifyDealsForActivationReturn
	err = dealsReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Return"] = dealsReturn
	return metadata, nil
}

func (p *Parser) activateDeals(rawParams []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(rawParams)
	var params market.ActivateDealsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}

func (p *Parser) onMinerSectorsTerminate(rawParams []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(rawParams)
	var params market.OnMinerSectorsTerminateParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}

func (p *Parser) computeDataCommitment(rawParams, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(rawParams)
	var params market.ComputeDataCommitmentParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	reader = bytes.NewReader(rawReturn)
	var computeReturn market.ComputeDataCommitmentReturn
	err = computeReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Return"] = computeReturn
	return metadata, nil
}
