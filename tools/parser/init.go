package parser

import (
	"bytes"
	"errors"
	builtinInit "github.com/filecoin-project/go-state-types/builtin/v10/init"
	filTypes "github.com/filecoin-project/lotus/chain/types"
)

func (p *Parser) parseInit(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt, height int64,
	key filTypes.TipSetKey) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "Constructor":
		return p.initConstructor(msg.Params)
	case "Exec":
		return p.parseExec(msg, msgRct, height, key)
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) initConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var constructor builtinInit.ConstructorParams
	err := constructor.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = constructor
	return metadata, nil
}
