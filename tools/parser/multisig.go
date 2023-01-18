package parser

import (
	"bytes"
	"encoding/json"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v10/miner"
	"github.com/filecoin-project/go-state-types/builtin/v10/multisig"
	"github.com/filecoin-project/go-state-types/cbor"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func (p *Parser) parseMultisig(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	switch txType {
	case "Constructor":
	case "Send":
		return p.parseSend(msg), nil
	case "Propose":
		return p.propose(msg, msgRct)
	case "Approve":
		return p.approve(msg, msgRct, height, key)
	case "Cancel":
		return p.cancel(msg, height, key)
	case "AddSigner", "SwapSigner":
		return p.parseMsigParams(msg, height, key)
	case "RemoveSigner":
		return p.removeSigner(msg, height, key)
	case "ChangeNumApprovalsThreshold":
	case "AddVerifies":
	case "LockBalance":
	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) parseMsigParams(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	params, err := ParseMsigParams(msg, height, key, p.lib)
	if err != nil {
		return map[string]interface{}{}, err
	}
	var paramsMap map[string]interface{}
	err = json.Unmarshal([]byte(params), &paramsMap)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return paramsMap, nil
}

func (p *Parser) propose(msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	var proposeParams multisig.ProposeParams
	reader := bytes.NewReader(msg.Params)
	err := proposeParams.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	innerParams, err := p.innerProposeParams(proposeParams)
	if err != nil {
		// TODO: log.
	}
	metadata[tools.ParamsKey] = propose{
		To:     proposeParams.To.String(),
		Value:  proposeParams.Value.String(),
		Method: uint64(proposeParams.Method),
		Params: innerParams,
	}
	var proposeReturn multisig.ProposeReturn
	reader = bytes.NewReader(msgRct.Return)
	err = proposeReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = proposeReturn
	return metadata, nil
}

func (p *Parser) approve(msg *filTypes.Message, msgRct *filTypes.MessageReceipt, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	params, err := ParseMsigParams(msg, height, key, p.lib)
	if err != nil {
		return map[string]interface{}{}, err
	}
	metadata[tools.ParamsKey] = params
	reader := bytes.NewReader(msgRct.Return)
	var approveReturn multisig.ApproveReturn
	err = approveReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = approveReturn
	return metadata, nil
}

func (p *Parser) cancel(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	params, err := ParseMsigParams(msg, height, key, p.lib)
	if err != nil {
		return map[string]interface{}{}, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) removeSigner(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	params, err := ParseMsigParams(msg, height, key, p.lib)
	if err != nil {
		return map[string]interface{}{}, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) innerProposeParams(propose multisig.ProposeParams) (cbor.Unmarshaler, error) {
	reader := bytes.NewReader(propose.Params)
	switch propose.Method {
	case builtin.MethodSend:
		var params multisig.ProposeParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMultisig.Approve,
		builtin.MethodsMultisig.Cancel:
		var params multisig.TxnIDParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMultisig.AddSigner:
		var params multisig.AddSignerParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMultisig.RemoveSigner:
		var params multisig.RemoveSignerParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMultisig.SwapSigner:
		var params multisig.SwapSignerParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMultisig.ChangeNumApprovalsThreshold:
		var params multisig.ChangeNumApprovalsThresholdParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMultisig.LockBalance:
		var params multisig.LockBalanceParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	case builtin.MethodsMiner.WithdrawBalance:
		var params miner.WithdrawBalanceParams
		err := params.UnmarshalCBOR(reader)
		if err != nil {
			return nil, err
		}
		return &params, nil
	}
	return nil, errUnknownMethod
}
