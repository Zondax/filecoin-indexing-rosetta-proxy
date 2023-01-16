package parser

import (
	"bytes"
	"errors"
	"github.com/filecoin-project/go-state-types/builtin/v10/verifreg"
	filTypes "github.com/filecoin-project/lotus/chain/types"
)

func (p *Parser) parseVerifiedregistry(txType string, msg *filTypes.Message) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "Constructor":
	case "AddVerifier":
		return p.addVerifier(msg.Params)
	case "RemoveVerifier":
	case "AddVerifiedClient":
		return p.addVerifiedClient(msg.Params)
	case "UseBytes":
		return p.useBytes(msg.Params)
	case "RestoreBytes":
		return p.restoreBytes(msg.Params)
	case "RemoveVerifiedClientDataCap":
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) addVerifier(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params verifreg.AddVerifierParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}

func (p *Parser) addVerifiedClient(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params verifreg.AddVerifiedClientParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}

func (p *Parser) useBytes(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params verifreg.UseBytesParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}

func (p *Parser) restoreBytes(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params verifreg.RestoreBytesParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
	return metadata, nil
}
