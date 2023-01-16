package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	initActor "github.com/filecoin-project/specs-actors/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
)

func ParseInitActorExecParams(raw []byte) (initActor.ExecParams, error) {
	reader := bytes.NewReader(raw)
	var params initActor.ExecParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		rosetta.Logger.Error("Could not parse 'Init' actor's 'Exec' parameters:", err.Error())
		return params, err
	}
	return params, nil
}

func ParsePowerActorCreateMinerParams(raw []byte) (power.CreateMinerParams, error) {
	reader := bytes.NewReader(raw)
	var params power.CreateMinerParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		rosetta.Logger.Error("Could not parse 'Power' actor's 'CreateMiner' parameters:", err.Error())
		return params, err
	}
	return params, nil
}

func ParseExecReturn(raw []byte) (initActor.ExecReturn, error) {
	reader := bytes.NewReader(raw)
	var execReturn initActor.ExecReturn
	err := execReturn.UnmarshalCBOR(reader)
	if err != nil {
		return execReturn, err
	}
	return execReturn, nil
}

func ParseProposeParams(msg *filTypes.Message, height int64, key filTypes.TipSetKey, rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin) (map[string]interface{}, error) {
	var params map[string]interface{}
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return params, err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To, height, key)
	if err != nil {
		return params, err
	}

	if !rosettaLib.BuiltinActors.IsActor(actorCode, actors.ActorMultisigName) {
		return params, fmt.Errorf("id %v (address %v) doesn't correspond to a multisig actor", actorCode, msg.To)
	}

	parsedParams, err := rosettaLib.GetInnerProposeTxParams(string(msgSerial))
	if err != nil {
		rosetta.Logger.Errorf("Could not parse params. ParseProposeTxParams returned with error: %s", err.Error())
		return params, err
	}

	targetActorCode, err := database.ActorsDB.GetActorCode(parsedParams.To, height, key)
	if err != nil {
		return params, err
	}

	targetMethod, err := rosettaLib.GetProposedMethod(parsedParams, targetActorCode)
	if err != nil {
		return params, err
	}

	// We do this to turn multisig.ProposeParams into a map[string]interface{} with convenient types
	jsonResponse, err := json.Marshal(parsedParams)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonResponse, &params)
	if err != nil {
		return params, err
	}

	params["Method"] = targetMethod

	innerParams, err := rosettaLib.ParseParamsMultisigTx(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse inner params for propose method:", targetMethod, ". ParseParamsMultisigTx returned with error:", err.Error())
		rosetta.Logger.Debugf("raw serial msg: %s", string(msgSerial))
		return params, err
	}

	innerParamsMap := map[string]interface{}{}
	if err := json.Unmarshal([]byte(innerParams), &innerParamsMap); err != nil {
		rosetta.Logger.Error("Could not unmarshall inner params for propose method:", targetMethod, ". ParseParamsMultisigTx returned with error:", err.Error())
		return params, err
	}

	params["Params"] = innerParamsMap

	return params, nil
}

func ParseMsigParams(msg *filTypes.Message, height int64, key filTypes.TipSetKey, rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin) (string, error) {
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return "", err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To, height, key)
	if err != nil {
		return "", err
	}

	if !rosettaLib.BuiltinActors.IsActor(actorCode, actors.ActorMultisigName) {
		return "", fmt.Errorf("this id doesn't correspond to a multisig actor")
	}

	parsedParams, err := rosettaLib.ParseParamsMultisigTx(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse params. ParseParamsMultisigTx returned with error:", err.Error())
		return "", err
	}

	return parsedParams, nil
}

func (p *Parser) parseAccount(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "PubkeyAddress":
		metadata := make(map[string]interface{})
		metadata["Params"] = base64.StdEncoding.EncodeToString(msg.Params)
		return metadata, nil
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) parseSend(msg *filTypes.Message) map[string]interface{} {
	metadata := make(map[string]interface{})
	metadata["Params"] = msg.Params
	return metadata
}

//func (p *Parser) parseExec(msg *filTypes.Message, msgRct *filTypes.MessageReceipt,
//	height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
//	// Check if this Exec contains actor creation event
//	createdActor, err := searchForActorCreation(msg, msgRct, height, key, p.lib)
//	if err != nil {
//		return map[string]interface{}{}, err
//	}
//
//	if createdActor == nil {
//		return map[string]interface{}{}, errors.New("not an actor creation event")
//	}
//	p.appendToAddresses(*createdActor)
//	metadata := make(map[string]interface{})
//	reader := bytes.NewReader(raw)
//	var params miner.ProveCommitAggregateParams
//	err := params.UnmarshalCBOR(reader)
//	if err != nil {
//		return metadata, err
//	}
//	metadata["Params"] = params
//	return metadata, nil
//
//	return map[string]interface{}{}, nil
//}

func searchEthLogs(logs []EthLog, msg *filTypes.Message) ([]EthLog, error) {
	ethHash, err := api.NewEthHashFromCid(msg.Cid())
	if err != nil {
		return nil, err
	}

	res := make([]EthLog, 0)
	for _, log := range logs {
		if log["transactionHash"] == ethHash.String() {
			res = append(res, log)
		}
	}

	return res, nil
}
