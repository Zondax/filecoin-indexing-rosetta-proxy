package parser

import (
	"bytes"
	"encoding/base64"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/builtin/v10/miner"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
)

func (p *Parser) parseStorageminer(txType string, msg *filTypes.Message, msgRct *filTypes.MessageReceipt) (map[string]interface{}, error) {
	switch txType {
	case tools.MethodSend:
		return p.parseSend(msg), nil
	case tools.MethodConstructor:
		return p.minerConstructor(msg.Params)
	case tools.MethodAwardBlockReward: // ?
	case tools.MethodControlAddresses:
		return p.controlAddresses(msg.Params, msgRct.Return)
	case tools.MethodChangeWorkerAddress:
		return p.changeWorkerAddress(msg.Params)
	case tools.MethodChangePeerID:
		return p.changePeerID(msg.Params)
	case tools.MethodSubmitWindowedPoSt:
		return p.submitWindowedPoSt(msg.Params)
	case tools.MethodPreCommitSector:
		return p.preCommitSector(msg.Params)
	case tools.MethodProveCommitSector:
		return p.proveCommitSector(msg.Params)
	case tools.MethodExtendSectorExpiration:
		return p.extendSectorExpiration(msg.Params)
	case tools.MethodTerminateSectors:
		return p.terminateSectors(msg.Params, msgRct.Return)
	case tools.MethodDeclareFaults:
		return p.declareFaults(msg.Params)
	case tools.MethodDeclareFaultsRecovered:
		return p.declareFaultsRecovered(msg.Params)
	case tools.MethodOnDeferredCronEvent:
		return p.onDeferredCronEvent(msg.Params)
	case tools.MethodCheckSectorProven:
		return p.checkSectorProven(msg.Params)
	case tools.MethodApplyRewards:
		return p.applyRewards(msg.Params)
	case tools.MethodReportConsensusFault:
		return p.reportConsensusFault(msg.Params)
	case tools.MethodWithdrawBalance:
		return p.parseWithdrawBalance(msg.Params)
	case tools.MethodConfirmSectorProofsValid:
		return p.confirmSectorProofsValid(msg.Params)
	case tools.MethodChangeMultiaddrs:
		return p.changeMultiaddrs(msg.Params)
	case tools.MethodCompactPartitions:
		return p.compactPartitions(msg.Params)
	case tools.MethodCompactSectorNumbers:
		return p.compactSectorNumbers(msg.Params)
	case tools.MethodConfirmUpdateWorkerKey:
	case tools.MethodRepayDebt:
	case tools.MethodChangeOwnerAddress:
	case tools.MethodDisputeWindowedPoSt:
		return p.disputeWindowedPoSt(msg.Params)
	case tools.MethodPreCommitSectorBatch:
		return p.preCommitSectorBatch(msg.Params)
	case tools.MethodProveCommitAggregate:
		return p.proveCommitAggregate(msg.Params)
	case tools.MethodProveReplicaUpdates:
		return p.proveReplicaUpdates(msg.Params)
	case tools.MethodChangeBeneficiary:
		return p.changeBeneficiary(msg.Params)
	case tools.MethodGetBeneficiary:
		return p.getBeneficiary(msg.Params, msgRct.Return)
	}
	return map[string]interface{}{}, errUnknownMethod
}

func (p *Parser) terminateSectors(rawParams, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(rawParams)
	var params miner.TerminateSectorsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	reader = bytes.NewReader(rawReturn)
	var terminateReturn miner.TerminateSectorsReturn
	err = terminateReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = terminateReturn
	return metadata, nil
}

func (p *Parser) controlAddresses(rawParams, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	if rawParams != nil {
		metadata[tools.ParamsKey] = base64.StdEncoding.EncodeToString(rawParams)
	}
	reader := bytes.NewReader(rawReturn)
	var controlReturn miner.GetControlAddressesReturn
	err := controlReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = controlAddress{
		Owner:        controlReturn.Owner.String(),
		Worker:       controlReturn.Worker.String(),
		ControlAddrs: getControlAddrs(controlReturn.ControlAddrs),
	}
	return metadata, nil
}

func (p *Parser) declareFaults(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.DeclareFaultsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) declareFaultsRecovered(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.DeclareFaultsRecoveredParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) proveReplicaUpdates(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ProveReplicaUpdatesParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) proveCommitAggregate(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ProveCommitAggregateParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) preCommitSectorBatch(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.PreCommitSectorBatchParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) disputeWindowedPoSt(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.DisputeWindowedPoStParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) compactSectorNumbers(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.CompactSectorNumbersParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) compactPartitions(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.CompactPartitionsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) changeMultiaddrs(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ChangeMultiaddrsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) checkSectorProven(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.CheckSectorProvenParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) extendSectorExpiration(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ExtendSectorExpirationParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) changePeerID(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ChangePeerIDParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) changeWorkerAddress(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ChangeWorkerAddressParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) reportConsensusFault(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ReportConsensusFaultParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) changeBeneficiary(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ChangeBeneficiaryParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) confirmSectorProofsValid(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ConfirmSectorProofsParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) minerConstructor(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.MinerConstructorParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) parseWithdrawBalance(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.WithdrawBalanceParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) applyRewards(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ApplyRewardParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) preCommitSector(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.PreCommitSectorParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) proveCommitSector(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ProveCommitSectorParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) submitWindowedPoSt(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.SubmitWindowedPoStParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) onDeferredCronEvent(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.DeferredCronEventParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ParamsKey] = params
	return metadata, nil
}

func (p *Parser) getBeneficiary(rawParams, rawReturn []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	if rawParams != nil {
		metadata[tools.ParamsKey] = base64.StdEncoding.EncodeToString(rawParams)
	}
	reader := bytes.NewReader(rawReturn)
	var beneficiaryReturn miner.GetBeneficiaryReturn
	err := beneficiaryReturn.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata[tools.ReturnKey] = getBeneficiryReturn{
		Active: activeBeneficiary{
			Beneficiary: beneficiaryReturn.Active.Beneficiary.String(),
			Term: beneficiaryTerm{
				Quota:      beneficiaryReturn.Active.Term.Quota.String(),
				UsedQuota:  beneficiaryReturn.Active.Term.UsedQuota.String(),
				Expiration: int64(beneficiaryReturn.Active.Term.Expiration),
			},
		},
		Proposed: proposed{
			NewBeneficiary:        beneficiaryReturn.Proposed.NewBeneficiary.String(),
			NewQuota:              beneficiaryReturn.Proposed.NewQuota.String(),
			NewExpiration:         int64(beneficiaryReturn.Proposed.NewExpiration),
			ApprovedByBeneficiary: beneficiaryReturn.Proposed.ApprovedByBeneficiary,
			ApprovedByNominee:     beneficiaryReturn.Proposed.ApprovedByNominee,
		},
	}
	return metadata, nil
}

func getControlAddrs(addrs []address.Address) []string {
	r := make([]string, len(addrs))
	for i, addr := range addrs {
		r[i] = addr.String()
	}
	return r
}
