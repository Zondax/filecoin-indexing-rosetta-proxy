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

	ParamsKey = "Params"
	ReturnKey = "Return"

	UnknownStr = "unknown"

	// Methods
	MethodSend                        = "Send"                        // Common
	MethodFee                         = "Fee"                         // Common
	MethodConstructor                 = "Constructor"                 // Common
	MethodCronTick                    = "CronTick"                    // Common
	MethodEpochTick                   = "EpochTick"                   // Cron
	MethodExec                        = "Exec"                        // MethodsInit
	MethodSwapSigner                  = "SwapSigner"                  // MethodsMultisig
	MethodAddSigner                   = "AddSigner"                   // MethodsMultisig
	MethodRemoveSigner                = "RemoveSigner"                // MethodsMultisig
	MethodPropose                     = "Propose"                     // MethodsMultisig
	MethodApprove                     = "Approve"                     // MethodsMultisig
	MethodCancel                      = "Cancel"                      // MethodsMultisig
	MethodChangeNumApprovalsThreshold = "ChangeNumApprovalsThreshold" // MethodsMultisig
	MethodLockBalance                 = "LockBalance"                 // MethodsMultisig
	MethodAddVerifies                 = "AddVerifies"                 // MethodsMultisig
	MethodAwardBlockReward            = "AwardBlockReward"            // MethodsReward
	MethodUpdateNetworkKPI            = "UpdateNetworkKPI"            // MethodsReward
	MethodThisEpochReward             = "ThisEpochReward"             // MethodsReward
	MethodCreateMiner                 = "CreateMiner"                 // MethodsPower
	MethodUpdateClaimedPower          = "UpdateClaimedPower"          // MethodsPower
	MethodEnrollCronEvent             = "EnrollCronEvent"             // MethodsPower
	MethodSubmitPoRepForBulkVerify    = "SubmitPoRepForBulkVerify"    // MethodsPower
	MethodCurrentTotalPower           = "CurrentTotalPower"           // MethodsPower
	MethodUpdatePledgeTotal           = "UpdatePledgeTotal"           // MethodsPower
	MethodDeprecated1                 = "Deprecated1"                 // MethodsPower
	MethodOnDeferredCronEvent         = "OnDeferredCronEvent"         // MethodsMiner
	MethodPreCommitSector             = "PreCommitSector"             // MethodsMiner
	MethodProveCommitSector           = "ProveCommitSector"           // MethodsMiner
	MethodSubmitWindowedPoSt          = "SubmitWindowedPoSt"          // MethodsMiner
	MethodApplyRewards                = "ApplyRewards"                // MethodsMiner
	MethodWithdrawBalance             = "WithdrawBalance"             // MethodsMiner
	MethodChangeOwnerAddress          = "ChangeOwnerAddress"          // MethodsMiner
	MethodChangeWorkerAddress         = "ChangeWorkerAddress"         // MethodsMiner
	MethodConfirmUpdateWorkerKey      = "ConfirmUpdateWorkerKey"      // MethodsMiner
	MethodDeclareFaultsRecovered      = "DeclareFaultsRecovered"      // MethodsMiner
	MethodPreCommitSectorBatch        = "PreCommitSectorBatch"        // MethodsMiner
	MethodProveCommitAggregate        = "ProveCommitAggregate"        // MethodsMiner
	MethodProveReplicaUpdates         = "ProveReplicaUpdates"         // MethodsMiner
	MethodChangeMultiaddrs            = "ChangeMultiaddrs"            // MethodsMiner
	MethodChangePeerID                = "ChangePeerID"                // MethodsMiner
	MethodExtendSectorExpiration      = "ExtendSectorExpiration"      // MethodsMiner
	MethodControlAddresses            = "ControlAddresses"            // MethodsMiner
	MethodTerminateSectors            = "TerminateSectors"            // MethodsMiner
	MethodDeclareFaults               = "DeclareFaults"               // MethodsMiner
	MethodCheckSectorProven           = "CheckSectorProven"           // MethodsMiner
	MethodReportConsensusFault        = "ReportConsensusFault"        // MethodsMiner
	MethodConfirmSectorProofsValid    = "ConfirmSectorProofsValid"    // MethodsMiner
	MethodCompactPartitions           = "CompactPartitions"           // MethodsMiner
	MethodCompactSectorNumbers        = "CompactSectorNumbers"        // MethodsMiner
	MethodRepayDebt                   = "RepayDebt"                   // MethodsMiner
	MethodDisputeWindowedPoSt         = "DisputeWindowedPoSt"         // MethodsMiner
	MethodChangeBeneficiary           = "ChangeBeneficiary"           // MethodsMiner
	MethodGetBeneficiary              = "GetBeneficiary"              // MethodsMiner
	MethodPublishStorageDeals         = "PublishStorageDeals"         // MethodsMarket
	MethodAddBalance                  = "AddBalance"                  // MethodsMarket
	MethodVerifyDealsForActivation    = "VerifyDealsForActivation"    // MethodsMarket
	MethodActivateDeals               = "ActivateDeals"               // MethodsMarket
	MethodOnMinerSectorsTerminate     = "OnMinerSectorsTerminate"     // MethodsMarket
	MethodComputeDataCommitment       = "ComputeDataCommitment"       // MethodsMarket
	MethodUpdateChannelState          = "UpdateChannelState"          // MethodsPaymentChannel
	MethodSettle                      = "Settle"                      // MethodsPaymentChannel
	MethodCollect                     = "Collect"                     // MethodsPaymentChannel
	MethodAddVerifiedClient           = "AddVerifiedClient"           // MethodsVerifiedRegistry
	MethodAddVerifier                 = "AddVerifier"                 // MethodsVerifiedRegistry
	MethodRemoveVerifier              = "RemoveVerifier"              // MethodsVerifiedRegistry
	MethodUseBytes                    = "UseBytes"                    // MethodsVerifiedRegistry
	MethodRestoreBytes                = "RestoreBytes"                // MethodsVerifiedRegistry
	MethodRemoveExpiredAllocations    = "RemoveExpiredAllocations"    // MethodsVerifiedRegistry
	MethodRemoveVerifiedClientDataCap = "RemoveVerifiedClientDataCap" // MethodsVerifiedRegistry
	MethodInvokeContract              = "InvokeContract"              // MethodsEVM
	MethodGetBytecode                 = "GetBytecode"                 // MethodsEVM
	MethodGetStorageAt                = "GetStorageAt"                // MethodsEVM
	MethodInvokeContractReadOnly      = "InvokeContractReadOnly"      // MethodsEVM
	MethodInvokeContractDelegate      = "InvokeContractDelegate"      // MethodsEVM
)

// SupportedOperations operations that will be parsed
var SupportedOperations = map[string]bool{
	MethodSend:                   true,
	MethodFee:                    true,
	MethodExec:                   true,
	MethodSwapSigner:             true,
	MethodAddSigner:              true,
	MethodRemoveSigner:           true,
	MethodPropose:                true,
	MethodApprove:                true,
	MethodCancel:                 true,
	MethodAwardBlockReward:       true,
	MethodOnDeferredCronEvent:    true,
	MethodPreCommitSector:        true,
	MethodProveCommitSector:      true,
	MethodSubmitWindowedPoSt:     true,
	MethodApplyRewards:           true,
	MethodWithdrawBalance:        true,
	MethodChangeOwnerAddress:     true,
	MethodChangeWorkerAddress:    true,
	MethodConfirmUpdateWorkerKey: true,
	MethodDeclareFaultsRecovered: true,
	MethodPreCommitSectorBatch:   true,
	MethodProveCommitAggregate:   true,
	MethodProveReplicaUpdates:    true,
	MethodCreateMiner:            true,
	MethodChangeMultiaddrs:       true,
	MethodChangePeerID:           true,
	MethodExtendSectorExpiration: true,
	MethodPublishStorageDeals:    true,
	MethodAddBalance:             true,
	MethodAddVerifiedClient:      true,
	MethodAddVerifier:            true,
	MethodRemoveVerifier:         true,
	MethodInvokeContract:         true,
	MethodGetBytecode:            true,
	MethodGetStorageAt:           true,
	MethodInvokeContractReadOnly: true,
	MethodInvokeContractDelegate: true,
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
