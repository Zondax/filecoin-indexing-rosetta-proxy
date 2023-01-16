package parser

import (
	"bytes"
	"encoding/base64"
	"errors"
	builtinInit "github.com/filecoin-project/go-state-types/builtin/v10/init"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	filInit "github.com/filecoin-project/specs-actors/actors/builtin/init"
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

func (p *Parser) parseExec(msg *filTypes.Message, msgRct *filTypes.MessageReceipt,
	height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	// Check if this Exec contains actor creation event
	createdActor, err := searchForActorCreation(msg, msgRct, height, key, p.lib)
	if err != nil {
		return map[string]interface{}{}, err
	}

	if createdActor == nil {
		return map[string]interface{}{}, errors.New("not an actor creation event")
	}
	p.appendToAddresses(*createdActor)
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(msg.Params)
	var params filInit.ExecParams
	err = params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = execParams{
		CodeCid:           params.CodeCID.String(),
		ConstructorParams: base64.StdEncoding.EncodeToString(params.ConstructorParams),
	}
	metadata["Return"] = createdActor
	return metadata, nil
}
