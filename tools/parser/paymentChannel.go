package parser

import (
	"bytes"
	"errors"
	"github.com/filecoin-project/go-state-types/builtin/v8/paych"
	filTypes "github.com/filecoin-project/lotus/chain/types"
)

func (p *Parser) parsePaymentchannel(txType string, msg *filTypes.Message) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "Constructor":
		return p.paymentChannelConstructor(msg.Params)
	case "UpdateChannelState":
		return p.updateChannelState(msg.Params)
	case "Settle":
	case "Collect":

	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) paymentChannelConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var constructor paych.ConstructorParams
	err := constructor.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = constructor
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
	metadata["Params"] = constructor
	return metadata, nil
}
