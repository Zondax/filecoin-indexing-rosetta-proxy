package tools

import (
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-address"
	methods "github.com/filecoin-project/go-state-types/builtin"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"reflect"
)

const UnknownStr = "unknown"

func IsOpSupported(op string) bool {
	supported, ok := SupportedOperations[op]
	if ok && supported {
		return true
	}

	return false
}

func SetupSupportedOperations(ops []string) {
	for s := range SupportedOperations {
		for _, op := range ops {
			found := false
			if s == op {
				found = true
			}
			SupportedOperations[s] = found
			if found {
				break
			}
		}
	}
}

func GetActorNameFromAddress(address address.Address, height int64, key filTypes.TipSetKey, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) string {
	var actorCode cid.Cid
	// Search for actor in cache
	var err error
	actorCode, err = database.ActorsDB.GetActorCode(address, height, key)
	if err != nil {
		return actors.UnknownStr
	}

	actorName, err := lib.BuiltinActors.GetActorNameFromCid(actorCode)
	if err != nil {
		return actors.UnknownStr
	}

	return actorName
}

func GetMethodName(msg *filTypes.Message, height int64, key filTypes.TipSetKey, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) (string, *rosettaTypes.Error) {

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

	actorName := GetActorNameFromAddress(msg.To, height, key, lib)

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
		return UnknownStr, nil
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
	return UnknownStr, nil
}

func GetActorAddressInfo(add address.Address, height int64, key filTypes.TipSetKey, lib *rosettaFilecoinLib.RosettaConstructionFilecoin) types.AddressInfo {
	// TODO: add support for eth
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
		addInfo.ActorType, _ = lib.BuiltinActors.GetActorNameFromCid(addInfo.ActorCid)
	}

	return addInfo
}
