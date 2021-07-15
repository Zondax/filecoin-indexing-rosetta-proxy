package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	initActor "github.com/filecoin-project/specs-actors/v4/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v5/actors/builtin/power"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	filLib "github.com/zondax/rosetta-filecoin-lib"
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

func ParseProposeParams(msg *filTypes.Message) (map[string]interface{}, error) {
	r := &filLib.RosettaConstructionFilecoin{}
	var params map[string]interface{}
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return params, err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To)
	if err != nil {
		return params, err
	}

	if !builtin.IsMultisigActor(actorCode) {
		return params, fmt.Errorf("this id doesn't correspond to a multisig actor")
	}

	innerMethod, parsedParams, err := r.ParseProposeTxParams(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse params. ParseProposeTxParams returned with error:", err.Error())
		return params, err
	}

	err = json.Unmarshal([]byte(parsedParams), &params)
	if err != nil {
		return params, err
	}

	params["Method"] = innerMethod
	return params, nil
}

func ParseMsigParams(msg *filTypes.Message) (string, error) {
	r := &filLib.RosettaConstructionFilecoin{}
	msgSerial, err := msg.MarshalJSON()
	if err != nil {
		rosetta.Logger.Error("Could not parse params. Cannot serialize lotus message:", err.Error())
		return "", err
	}

	actorCode, err := database.ActorsDB.GetActorCode(msg.To)
	if err != nil {
		return "", err
	}

	if !builtin.IsMultisigActor(actorCode) {
		return "", fmt.Errorf("this id doesn't correspond to a multisig actor")
	}

	parsedParams, err := r.ParseParamsMultisigTx(string(msgSerial), actorCode)
	if err != nil {
		rosetta.Logger.Error("Could not parse params. ParseParamsMultisigTx returned with error:", err.Error())
		return "", err
	}

	return parsedParams, nil
}
