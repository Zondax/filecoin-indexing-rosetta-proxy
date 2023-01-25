package parser

import (
	"bytes"
	"encoding/base64"
	"github.com/filecoin-project/go-state-types/builtin/v9/market" // TODO: v10 does not support ComputeDataCommitmentParams and OnMinerSectorsTerminateParams
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func (p *Parser) parseStoragemarket(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "Constructor":
	case "AddBalance":
	case "WithdrawBalance":
		return p.withdrawBalance(msg.Params, msgRct.Return)
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
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) withdrawBalance(raw, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params market.WithdrawBalanceParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	if rawReturn != nil {
		metadata[tools.ReturnKey] = base64.StdEncoding.EncodeToString(rawReturn)
	}
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
	metadata[tools.ParamsKey] = params
	reader = bytes.NewReader(rawReturn)
	var publishReturn market.PublishStorageDealsReturn
	err = publishReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = publishReturn
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
	metadata[tools.ParamsKey] = params
	reader = bytes.NewReader(rawReturn)
	var dealsReturn market.VerifyDealsForActivationReturn
	err = dealsReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = dealsReturn
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
	metadata[tools.ParamsKey] = params
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
	metadata[tools.ParamsKey] = params
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
	metadata[tools.ParamsKey] = params
	reader = bytes.NewReader(rawReturn)
	var computeReturn market.ComputeDataCommitmentReturn
	err = computeReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = computeReturn
	return metadata, nil
}
