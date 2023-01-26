package parser

import (
	"bytes"
	"github.com/filecoin-project/go-state-types/cbor"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"io"
)

func (p *Parser) parseStoragepower(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt,
	height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	switch txType {
	case tools.MethodSend:
		return p.parseSend(msg), nil
	case tools.MethodConstructor:
		return p.powerConstructor(msg.Params)
	case tools.MethodCreateMiner:
		return p.parseCreateMiner(msg, msgRct, height, key)
	case tools.MethodUpdateClaimedPower:
		return p.updateClaimedPower(msg.Params)
	case tools.MethodEnrollCronEvent:
		return p.enrollCronEvent(msg.Params)
	case tools.MethodCronTick:
	case tools.MethodUpdatePledgeTotal: // TODO
	case tools.MethodDeprecated1:
	case tools.MethodSubmitPoRepForBulkVerify:
		return p.submitPoRepForBulkVerify(msg.Params)
	case tools.MethodCurrentTotalPower:
		return p.currentTotalPower(msgRct.Return)

	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) currentTotalPower(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params power.CurrentTotalPowerReturn
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = params
	return metadata, nil
}

func (p *Parser) submitPoRepForBulkVerify(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params proof.SealVerifyInfo
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) powerConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params power.MinerConstructorParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) parseCreateMiner(msg *filTypes.Message, msgRct *filTypes.MessageReceipt,
	height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	createdActor, err := p.searchForActorCreation(msg, msgRct, height, key)
	if err != nil {
		return map[string]interface{}{}, err
	}
	p.appendToAddresses(*createdActor)
	metadata[tools.ReturnKey] = createdActor
	reader := bytes.NewReader(msg.Params)
	var params power.CreateMinerParams
	err = params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) enrollCronEvent(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params power.EnrollCronEventParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) updateClaimedPower(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params power.UpdateClaimedPowerParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) unmarshalParams(reader io.Reader, unmarshaler cbor.Unmarshaler) (cbor.Unmarshaler, error) {
	err := unmarshaler.UnmarshalCBOR(reader)
	return unmarshaler, err
}
