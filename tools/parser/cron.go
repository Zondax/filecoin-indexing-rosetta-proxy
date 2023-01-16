package parser

import (
	"bytes"
	"errors"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/cron"
)

func (p *Parser) parseCron(txType string, msg *filTypes.Message) (map[string]interface{}, error) {
	switch txType {
	case "Constructor":
		metadata := make(map[string]interface{})
		reader := bytes.NewReader(msg.Params)
		var constructor cron.ConstructorParams
		err := constructor.UnmarshalCBOR(reader)
		if err != nil {
			return metadata, err
		}
		metadata["Params"] = constructor
		return metadata, nil
	case "EpochTick":
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) cronConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var constructor cron.ConstructorParams
	err := constructor.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = constructor
	return metadata, nil
}
