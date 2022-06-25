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
	"CreateMiner":            true, // MethodsPower
	"AddBalance":             true, // MethodsMarket
	"AddVerifiedClient":      true, // MethodsVerifiedRegistry
	"AddVerifier":            true, // MethodsVerifiedRegistry
	"RemoveVerifier":         true, // MethodsVerifiedRegistry
}
