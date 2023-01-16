package parser

import (
	"bytes"
	"errors"
	"github.com/filecoin-project/go-state-types/builtin/v10/miner"
	filTypes "github.com/filecoin-project/lotus/chain/types"
)

func (p *Parser) parseStorageminer(txType string, msg *filTypes.Message, height int64, key filTypes.TipSetKey) (map[string]interface{}, error) {
	switch txType {
	case "Send":
		return p.parseSend(msg), nil
	case "Constructor":
		return p.minerConstructor(msg.Params)
	case "AwardBlockReward": // ?
	case "ControlAddresses":
	case "ChangeWorkerAddress":
		return p.changeWorkerAddress(msg.Params)
	case "ChangePeerID":
		return p.changePeerID(msg.Params)
	case "SubmitWindowedPoSt":
		return p.submitWindowedPoSt(msg.Params)
	case "PreCommitSector":
		return p.preCommitSector(msg.Params)
	case "ProveCommitSector":
		return p.proveCommitSector(msg.Params)
	case "ExtendSectorExpiration":
		return p.extendSectorExpiration(msg.Params)
	case "TerminateSectors":
	case "DeclareFaults":
	case "DeclareFaultsRecovered":
	case "OnDeferredCronEvent":
		return p.onDeferredCronEvent(msg.Params)
	case "CheckSectorProven":
		return p.checkSectorProven(msg.Params)
	case "ApplyRewards":
		return p.applyRewards(msg.Params)
	case "ReportConsensusFault":
		return p.reportConsensusFault(msg.Params)
	case "WithdrawBalance":
		return p.parseWithdrawBalance(msg.Params)
	case "ConfirmSectorProofsValid":
		return p.confirmSectorProofsValid(msg.Params)
	case "ChangeMultiaddrs":
		return p.changeMultiaddrs(msg.Params)
	case "CompactPartitions":
		return p.compactPartitions(msg.Params)
	case "CompactSectorNumbers":
		return p.compactSectorNumbers(msg.Params)
	case "ConfirmUpdateWorkerKey":
	case "RepayDebt":
	case "ChangeOwnerAddress":
	case "DisputeWindowedPoSt":
		return p.disputeWindowedPoSt(msg.Params)
	case "PreCommitSectorBatch":
		return p.preCommitSectorBatch(msg.Params)
	case "ProveCommitAggregate":
		return p.proveCommitAggregate(msg.Params)
	case "ProveReplicaUpdates":
		return p.proveReplicaUpdates(msg.Params)
	case "ChangeBeneficiary":
		return p.changeBeneficiary(msg.Params)
	}
	return map[string]interface{}{}, errors.New("not method")
}

func (p *Parser) proveReplicaUpdates(raw []byte) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	reader := bytes.NewReader(raw)
	var params miner.ProveReplicaUpdatesParams
	err := params.UnmarshalCBOR(reader)
	if err != nil {
		return metadata, err
	}
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
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
	metadata["Params"] = params
	return metadata, nil
}
