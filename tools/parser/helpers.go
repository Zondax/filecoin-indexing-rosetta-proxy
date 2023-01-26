package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-address"
	methods "github.com/filecoin-project/go-state-types/builtin"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	initActor "github.com/filecoin-project/specs-actors/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/ipfs/go-cid"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"reflect"
)

func (p *Parser) getActorAddressInfo(add address.Address, height int64, key filTypes.TipSetKey) types.AddressInfo {
	var (
		addInfo types.AddressInfo
		err     error
	)
	addInfo.Robust, err = database.ActorsDB.GetRobustAddress(add)
	if err != nil {
		rosetta.Logger.Errorf("could not get robust address for %s. Err: %v", add.String(), err.Error())
	}

	addInfo.Short, err = database.ActorsDB.GetShortAddress(add)
	if err != nil {
		rosetta.Logger.Errorf("could not get short address for %s. Err: %v", add.String(), err.Error())
	}

	addInfo.ActorCid, err = database.ActorsDB.GetActorCode(add, height, key)
	if err != nil {
		rosetta.Logger.Error("could not get actor code from address. Err:", err.Error())
	} else {
		addInfo.ActorType, _ = p.lib.BuiltinActors.GetActorNameFromCid(addInfo.ActorCid)
	}

	return addInfo
}

func (p *Parser) getActorNameFromAddress(address address.Address, height int64, key filTypes.TipSetKey) string {
	var actorCode cid.Cid
	// Search for actor in cache
	var err error
	actorCode, err = database.ActorsDB.GetActorCode(address, height, key)
	if err != nil {
		return actors.UnknownStr
	}

	actorName, err := p.lib.BuiltinActors.GetActorNameFromCid(actorCode)
	if err != nil {
		return actors.UnknownStr
	}

	return actorName
}

func (p *Parser) getMethodName(msg *filTypes.Message, height int64, key filTypes.TipSetKey) (string, *rosettaTypes.Error) {

	if msg == nil {
		return "", rosetta.BuildError(rosetta.ErrMalformedValue, nil, true)
	}

	// Shortcut 1 - Method "0" corresponds to "MethodSend"
	if msg.Method == 0 {
		return "Send", nil
	}

	// Shortcut 2 - Method "1" corresponds to "MethodConstructor"
	if msg.Method == 1 {
		return "Constructor", nil
	}

	actorName := p.getActorNameFromAddress(msg.To, height, key)

	var method interface{}
	switch actorName {
	case "init":
		method = methods.MethodsInit
	case "cron":
		method = methods.MethodsCron
	case "account":
		method = methods.MethodsAccount
	case "storagepower":
		method = methods.MethodsPower
	case "storageminer":
		method = methods.MethodsMiner
	case "storagemarket":
		method = methods.MethodsMarket
	case "paymentchannel":
		method = methods.MethodsPaych
	case "multisig":
		method = methods.MethodsMultisig
	case "reward":
		method = methods.MethodsReward
	case "verifiedregistry":
		method = methods.MethodsVerifiedRegistry
	case "evm":
		method = methods.MethodsEVM
	case "eam":
		method = methods.MethodsEAM
	case "datacap":
		method = methods.MethodsDatacap
	default:
		return tools.UnknownStr, nil
	}

	val := reflect.Indirect(reflect.ValueOf(method))
	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Field(i)
		methodNum := field.Uint()
		if methodNum == uint64(msg.Method) {
			methodName := val.Type().Field(i).Name
			return methodName, nil
		}
	}
	return tools.UnknownStr, nil
}

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

	params[tools.ParamsKey] = innerParamsMap

	return params, nil
}

func (p *Parser) parseAccount(txType string, msg *filTypes.Message) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "PubkeyAddress":
		metadata := make(map[string]interface{})
		metadata[tools.ParamsKey] = base64.StdEncoding.EncodeToString(msg.Params)
		return metadata, nil
	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) parseSend(msg *filTypes.Message) map[string]interface{} {
	metadata := make(map[string]interface{})
	metadata[tools.ParamsKey] = msg.Params
	return metadata
}
