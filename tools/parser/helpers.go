package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	initActor "github.com/filecoin-project/specs-actors/v7/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/power"
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

func ParseProposeParams(msg *filTypes.Message, height int64, rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin) (map[string]interface{}, error) {
	var params map[string]interface{}
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return params, err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To, height)
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

	targetActorCode, err := database.ActorsDB.GetActorCode(parsedParams.To, height)
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

func ParseMsigParams(msg *filTypes.Message, height int64, rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin) (string, error) {
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return "", err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To, height)
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
