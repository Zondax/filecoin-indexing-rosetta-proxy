package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v10/miner"
	"github.com/filecoin-project/go-state-types/builtin/v10/multisig"
	"github.com/filecoin-project/go-state-types/cbor"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

func (p *Parser) parseMultisig(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	switch txType {
	case tools.MethodConstructor:
	case tools.MethodSend:
		return p.parseSend(msg), nil
	case tools.MethodPropose:
		return p.propose(msg, msgRct)
	case tools.MethodApprove:
		return p.approve(msg, msgRct, height, key)
	case tools.MethodCancel:
		return p.cancel(msg, height, key)
	case tools.MethodAddSigner, tools.MethodSwapSigner:
		return p.msigParams(msg, height, key)
	case tools.MethodRemoveSigner:
		return p.removeSigner(msg, height, key)
	case tools.MethodChangeNumApprovalsThreshold:
		return p.changeNumApprovalsThreshold(msg.Params)
	case tools.MethodAddVerifies:
	case tools.MethodLockBalance:
		return p.lockBalance(msg.Params)
	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) msigParams(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	params, err := p.parseMsigParams(msg, height, key)
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
	params, err := p.parseMsigParams(msg, height, key)
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
	params, err := p.parseMsigParams(msg, height, key)
	if err != nil {
		return map[string]interface{}{}, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) removeSigner(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	params, err := p.parseMsigParams(msg, height, key)
	if err != nil {
		return map[string]interface{}{}, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) changeNumApprovalsThreshold(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	var params multisig.ChangeNumApprovalsThresholdParams
	reader := bytes.NewReader(raw)
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) lockBalance(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	var params multisig.LockBalanceParams
	reader := bytes.NewReader(raw)
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) parseMsigParams(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (string, error) {
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return "", err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To, height, key)
	if err != nil {
		return "", err
	}

	if !p.lib.BuiltinActors.IsActor(actorCode, actors.ActorMultisigName) {
		return "", fmt.Errorf("this id doesn't correspond to a multisig actor")
	}

	parsedParams, err := p.lib.ParseParamsMultisigTx(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse params. ParseParamsMultisigTx returned with error:", err.Error())
		return "", err
	}

	return parsedParams, nil
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
