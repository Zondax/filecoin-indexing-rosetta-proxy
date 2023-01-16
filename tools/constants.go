package tools

var (
	// Versions info to be injected on build time
	RosettaSDKVersion = "Unknown"
	LotusVersion      = "Unknown"
	GitRevision       = "Unknown"

	// RosettaServerPort ServerPort to be injected on build time
	RosettaServerPort = "8080"

	// Other configs
	RetryConnectAttempts = "1000000"

	// Populated on main.go
	ConnectedToLotusVersion = UnknownStr

	// Network name (read from api in main)
	NetworkName = ""
)

const (
	// Network
	BlockChainName = "Filecoin"

	// Fees
	TotalFeeOp           = "Fee"
	OverEstimationBurnOp = "OverEstimationBurn"
	MinerFeeOp           = "MinerFee"
	BurnFeeOp            = "BurnFee"

	BurnAddress = "f099"
)

// SupportedOperations operations that will be parsed
var SupportedOperations = map[string]bool{
	"Send":                   true, // Common
	"Fee":                    true, // Common
	"Exec":                   true, // MethodsInit
	"SwapSigner":             true, // MethodsMultisig
	"AddSigner":              true, // MethodsMultisig
	"RemoveSigner":           true, // MethodsMultisig
	"Propose":                true, // MethodsMultisig
	"Approve":                true, // MethodsMultisig
	"Cancel":                 true, // MethodsMultisig
	"AwardBlockReward":       true, // MethodsReward
	"OnDeferredCronEvent":    true, // MethodsMiner
	"PreCommitSector":        true, // MethodsMiner
	"ProveCommitSector":      true, // MethodsMiner
	"SubmitWindowedPoSt":     true, // MethodsMiner
	"ApplyRewards":           true, // MethodsMiner
	"WithdrawBalance":        true, // MethodsMiner
	"ChangeOwnerAddress":     true, // MethodsMiner
	"ChangeWorkerAddress":    true, // MethodsMiner
	"ConfirmUpdateWorkerKey": true, // MethodsMiner
	"DeclareFaultsRecovered": true, // MethodsMiner
	"PreCommitSectorBatch":   true, // MethodsMiner
	"ProveCommitAggregate":   true, // MethodsMiner
	"ProveReplicaUpdates":    true, // MethodsMiner
	"CreateMiner":            true, // MethodsPower
	"AddBalance":             true, // MethodsMarket
	"AddVerifiedClient":      true, // MethodsVerifiedRegistry
	"AddVerifier":            true, // MethodsVerifiedRegistry
	"RemoveVerifier":         true, // MethodsVerifiedRegistry
	"InvokeContract":         true, // MethodsEVM
	"GetBytecode":            true, // MethodsEVM
	"GetStorageAt":           true, // MethodsEVM
	"InvokeContractReadOnly": true, // MethodsEVM
	"InvokeContractDelegate": true, // MethodsEVM
	"ChangeMultiaddrs":       true,
	"ChangePeerID":           true,
	"ExtendSectorExpiration": true,
	"PublishStorageDeals":    true,
}

func GetSupportedOps() []string {
	var result []string
	for k, v := range SupportedOperations {
		if v {
			result = append(result, k)
		}
	}
	return result
}
