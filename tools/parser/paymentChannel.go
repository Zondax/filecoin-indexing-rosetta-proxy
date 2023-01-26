package parser

import (
	"bytes"
	"github.com/filecoin-project/go-state-types/builtin/v8/paych"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func (p *Parser) parsePaymentchannel(txType string, msg *filTypes.Message) (map[string]interface{}, error) {
	switch txType {
	case tools.MethodSend:
		return p.parseSend(msg), nil
	case tools.MethodConstructor:
		return p.paymentChannelConstructor(msg.Params)
	case tools.MethodUpdateChannelState:
		return p.updateChannelState(msg.Params)
	case tools.MethodSettle:
	case tools.MethodCollect:
	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) paymentChannelConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var constructor paych.ConstructorParams
	err := constructor.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = constructor
	return metadata, nil
}

func (p *Parser) updateChannelState(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var constructor paych.UpdateChannelStateParams
	err := constructor.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = constructor
	return metadata, nil
}
