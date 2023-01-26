package parser

import (
	"bytes"
	"github.com/filecoin-project/go-state-types/builtin/v10/verifreg"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func (p *Parser) parseVerifiedRegistry(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	switch txType {
	case tools.MethodSend:
		return p.parseSend(msg), nil
	case tools.MethodConstructor:
	case tools.MethodAddVerifier:
		return p.addVerifier(msg.Params)
	case tools.MethodRemoveVerifier:
	case tools.MethodAddVerifiedClient:
		return p.addVerifiedClient(msg.Params)
	case tools.MethodUseBytes:
		return p.useBytes(msg.Params)
	case tools.MethodRestoreBytes:
		return p.restoreBytes(msg.Params)
	case tools.MethodRemoveVerifiedClientDataCap:
		// TODO: untested
		return p.removeVerifiedClientDataCap(msg.Params)
	case tools.MethodRemoveExpiredAllocations:
		return p.removeExpiredAllocations(msg.Params, msgRct.Return)
	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) addVerifier(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params verifreg.AddVerifierParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
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
	metadata[tools.ParamsKey] = params
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
	metadata[tools.ParamsKey] = params
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
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

// TODO: untested
func (p *Parser) removeVerifiedClientDataCap(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var datacap verifreg.DataCap
	err := datacap.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = datacap
	return metadata, nil
}

func (p *Parser) removeExpiredAllocations(raw, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params verifreg.RemoveExpiredAllocationsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	reader = bytes.NewReader(rawReturn)
	var expiredReturn verifreg.RemoveExpiredAllocationsReturn
	err = expiredReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = expiredReturn
	return metadata, nil
}
