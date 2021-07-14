package tools

import (
	rosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/filecoin-project/go-address"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	methods "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	"github.com/ipfs/go-cid"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools/database"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/types"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"reflect"
)

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

func GetActorNameFromAddress(address address.Address) string {
	var actorCode cid.Cid
	// Search for actor in cache
	var err error
	actorCode, err = database.ActorsDB.GetActorCode(address)
	if err != nil {
		return "unknown"
	}
	return rosetta.GetActorNameFromCid(actorCode)
}

func GetMethodName(msg *filTypes.Message) (string, *rosettaTypes.Error) {

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

	actorName := GetActorNameFromAddress(msg.To)

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
	default:
		return "unknown", nil
	}

	val := reflect.Indirect(reflect.ValueOf(method))
	idx := int(msg.Method)
	if idx > 0 {
		idx--
	}

	if val.Type().NumField() <= idx {
		return "unknown", nil
	}

	methodName := val.Type().Field(idx).Name
	return methodName, nil
}

func GetActorAddressInfo(add address.Address) (types.AddressInfo, error) {

	var addInfo types.AddressInfo

	isRobust, err := database.IsRobustAddress(add)
	if err != nil {
		return addInfo, err
	}

	if isRobust {
		addInfo.Robust = add.String()
		addInfo.Short, err = database.ActorsDB.GetShortAddress(add)
		if err != nil {
			rosetta.Logger.Errorf("could not get short address for %s. Err: %v", add.String(), err.Error())
		}
	} else {
		addInfo.Short = add.String()
		addInfo.Robust, err = database.ActorsDB.GetRobustAddress(add)
		if err != nil {
			rosetta.Logger.Errorf("could not get robust address for %s. Err: %v", add.String(), err.Error())
		}
	}

	actorCode, err := database.ActorsDB.GetActorCode(add)
	if err != nil {
		rosetta.Logger.Error("could not get actor code from address. Err:", err.Error())
	}
	addInfo.ActorCid = actorCode
	addInfo.ActorType = rosetta.GetActorNameFromCid(actorCode)

	return addInfo, nil
}
