package parser

import (
	"bytes"
	"encoding/hex"
	"github.com/filecoin-project/go-state-types/builtin/v10/evm"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func (p *Parser) parseEvm(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt, ethLogs []EthLog) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	switch txType {
	case "Constructor":
		return p.evmConstructor(msg.Params)
	case "InvokeContract", "InvokeContractReadOnly", "InvokeContractDelegate":
		metadata[tools.ParamsKey] = "0x" + hex.EncodeToString(msg.Params)
		metadata[tools.ReturnKey] = "0x" + hex.EncodeToString(msgRct.Return)
		logs, err := searchEthLogs(ethLogs, msg.Cid().String())
		if err != nil {
			return metadata, err
		}
		metadata["ethLogs"] = logs
	case "GetBytecode":
	}
	return metadata, nil
}

func (p *Parser) evmConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params evm.ConstructorParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func searchEthLogs(logs []EthLog, msgCid string) ([]EthLog, error) {
	res := make([]EthLog, 0)
	for _, log := range logs {
		if log["transactionCid"] == msgCid {
			res = append(res, log)
		}
	}
	return res, nil
}
