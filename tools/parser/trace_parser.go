package parser

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/filecoin-project/go-state-types/builtin/v10/eam"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/zondax/filecoin-indexing-rosetta-proxy/tools"
	types2 "github.com/zondax/filecoin-indexing-rosetta-proxy/tools/parser/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
	"github.com/zondax/rosetta-filecoin-lib/actors"
	rosetta "github.com/zondax/rosetta-filecoin-proxy/rosetta/services"
	"go.uber.org/zap"
)

type TransactionParser struct {
	rosettaLib *rosettaFilecoinLib.RosettaConstructionFilecoin
	rpcClient  api.FullNode
}

func NewTransactionParser(rpcClient api.FullNode) *TransactionParser {
	lib := rosettaFilecoinLib.NewRosettaConstructionFilecoin(&rpcClient)
	if lib == nil {
		zap.S().Fatalf("could not create instance of rosettaLib")
	}

	return &TransactionParser{
		rosettaLib: lib,
		rpcClient:  rpcClient,
	}
}

func (t *TransactionParser) BuildTransactions(states *api.ComputeStateOutput, rawTipset *types2.ExtendedTipSet) ([]*types2.Transaction, *types2.AddressInfoMap) {
	var discoveredAddresses = types2.NewAddressInfoMap()
	var transactions []*types2.Transaction

	for i := range states.Trace {
		trace := states.Trace[i]

		if trace.Msg == nil {
			continue
		}

		// Analyze full trace recursively
		t.ProcessTrace(&trace.ExecutionTrace, trace.MsgCid, transactions, rawTipset, &discoveredAddresses)
		// Add the corresponding "Fee" operation
		if !trace.GasCost.TotalCost.NilOrZero() {
			feeTxs := t.BuildFeeTx(trace, rawTipset)
			transactions = append(transactions, feeTxs...)
		}
	}
	return transactions, &discoveredAddresses
}

func (t *TransactionParser) BuildFeeTx(trace *api.InvocResult, rawTipset *types2.ExtendedTipSet) []*types2.Transaction {
	var txs = make([]*types2.Transaction, 0)

	feeMeta := types2.TransactionFeeInfo{
		TotalCost:          trace.GasCost.TotalCost.Uint64(),
		OverestimationBurn: trace.GasCost.OverEstimationBurn.Uint64(),
		MinerTip:           trace.GasCost.MinerTip.Uint64(),
		GasUsage:           trace.GasCost.GasUsed.Uint64(),
		GasLimit:           trace.Msg.GasLimit,
		GasPremium:         trace.Msg.GasPremium.Uint64(),
		BaseFeeBurn:        trace.GasCost.BaseFeeBurn.Uint64(),
	}

	feeMetaJson, _ := json.Marshal(feeMeta)

	// Fee tx
	feeTx := &types2.Transaction{
		Height:      int64(rawTipset.TipSet.Height()),
		TxTimestamp: int64(rawTipset.TipSet.MinTimestamp()),
		TxHash:      trace.MsgCid.String(),
		TxFrom:      trace.Msg.From.String(),
		Amount:      GetCastedAmount(trace.GasCost.TotalCost.String()),
		TxType:      tools.TotalFeeOp,
		Status:      TransactionStatusOk,
		TxMetadata:  string(feeMetaJson),
	}

	txs = append(txs, feeTx)

	// Miner tip tx
	// Search the miner who mined this tx
	blocks, ok := rawTipset.BlockMessages[feeTx.TxHash]
	if !ok || len(blocks) == 0 {
		zap.S().Errorf("Warning: could not find tx hash '%s' in rawTipsetData!. Could not build MinerTip tx", feeTx.TxHash)
		return txs
	}

	miner := blocks[0].BlockMiner

	minerTipTx := &types2.Transaction{
		Height:      int64(rawTipset.TipSet.Height()),
		TxTimestamp: int64(rawTipset.TipSet.MinTimestamp()),
		TxHash:      trace.MsgCid.String(),
		TxTo:        miner,
		Amount:      GetCastedAmount(trace.GasCost.MinerTip.String()),
		TxType:      tools.MinerFeeOp,
		Status:      TransactionStatusOk,
	}
	txs = append(txs, minerTipTx)

	return txs
}

func (t *TransactionParser) ProcessTrace(trace *filTypes.ExecutionTrace, msgCid cid.Cid, transactions []*types2.Transaction, rawTipset *types2.ExtendedTipSet,
	addresses *types2.AddressInfoMap) {

	if trace.Msg == nil {
		return
	}

	txStatus := TransactionStatusOk
	if !trace.MsgRct.ExitCode.IsSuccess() {
		txStatus = TransactionStatusFailed
	}

	baseMethod, err := tools.GetMethodName(trace.Msg, int64(rawTipset.Height()), rawTipset.Key(), t.rosettaLib)
	if err != nil {
		zap.S().Errorf("could not get method name: %v", err)
		baseMethod = UnknownStr
	}

	fromAdd := tools.GetActorAddressInfo(trace.Msg.From, int64(rawTipset.Height()), rawTipset.Key(), t.rosettaLib)
	toAdd := tools.GetActorAddressInfo(trace.Msg.To, int64(rawTipset.Height()), rawTipset.Key(), t.rosettaLib)
	appendAddressInfo(addresses, fromAdd, toAdd)

	// Basic tx info
	tx := &types2.Transaction{
		Height:      int64(rawTipset.Height()),
		TxTimestamp: int64(rawTipset.MinTimestamp()),
		TxHash:      trace.Msg.Cid().String(),
		TxFrom:      fromAdd.GetAddress(),
		TxTo:        toAdd.GetAddress(),
		Amount:      GetCastedAmount(trace.Msg.Value.String()),
		TxType:      baseMethod,
		Status:      txStatus,
	}

	switch baseMethod {
	case "InvokeContract", "InvokeContractReadOnly", "InvokeContractDelegate":
		{
			tx.TxParams = "0x" + hex.EncodeToString(trace.Msg.Params)
			tx.TxReturn = "0x" + hex.EncodeToString(trace.MsgRct.Return)
		}
	case "Create", "Create2":
		{
			metadata := make(map[string]interface{})
			var result eam.CreateReturn
			r := bytes.NewReader(trace.MsgRct.Return)
			if err := result.UnmarshalCBOR(r); err != nil {
				zap.S().Errorf("error unmarshaling return value for method '%s': %v", baseMethod, err)
				break
			}

			ethHash, err := api.NewEthHashFromCid(msgCid)
			if err != nil {
				zap.S().Errorf("error getting eth hash from cid for methos '%s': %v", baseMethod, err)
				break
			}

			metadata["robustAdd"] = result.RobustAddress.String()
			metadata["ethAdd"] = "0x" + hex.EncodeToString(result.EthAddress[:])
			metadata["cid"] = msgCid.String()
			metadata["ethHash"] = ethHash.String()
			metaJson, _ := json.Marshal(metadata)
			tx.TxMetadata = string(metaJson)
		}
	case "AddBalance":
		{
			// We don't add anything here to tx
			break
		}
	case "Send":
		{
			tx.TxMetadata = string(trace.Msg.Params)
		}
	case "CreateMiner":
		{
			createdActor, err := t.searchForActorCreation(trace.Msg, trace.MsgRct, int64(rawTipset.Height()), rawTipset.Key())
			if err != nil {
				rosetta.Logger.Errorf("Could not parse 'CreateMiner' params, err: %v", err)
				break
			}
			metadata := make(map[string]interface{})
			metadata["CreatedMiner"] = createdActor.Robust
			metadataJson, _ := json.Marshal(metadata)
			tx.TxMetadata = string(metadataJson)
			appendAddressInfo(addresses, *createdActor)
		}
	case "Exec":
		{
			// Check if this Exec contains actor creation event
			createdActor, err := t.searchForActorCreation(trace.Msg, trace.MsgRct, int64(rawTipset.Height()), rawTipset.Key())
			if err != nil {
				zap.S().Errorf("Could not parse Exec params, err: %v", err)
				break
			}

			if createdActor == nil {
				// This is not an actor creation event
				break
			}

			appendAddressInfo(addresses, *createdActor)

			// Check if the created actor is of multisig type and if it was also funded
			if t.rosettaLib.BuiltinActors.IsActor(createdActor.ActorCid, actors.ActorMultisigName) &&
				!trace.Msg.Value.NilOrZero() {

				txMsigFund := &types2.Transaction{
					Height:      int64(rawTipset.Height()),
					TxTimestamp: int64(rawTipset.MinTimestamp()),
					TxHash:      trace.Msg.Cid().String(),
					TxFrom:      fromAdd.GetAddress(),
					TxTo:        createdActor.Robust,
					Amount:      GetCastedAmount(trace.Msg.Value.String()),
					TxType:      baseMethod,
					Status:      txStatus,
				}

				transactions = append(transactions, txMsigFund)
			}
		}
	case "Propose":
		{
			//params, err := ParseProposeParams(trace.Msg, height, key, lib)
			//if err != nil {
			//	rosetta.Logger.Errorf("Could not parse message params for %v, error: %v", baseMethod, err.Error())
			//	break
			//}
			//
			//*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
			//	"0", txStatus, false, &params)
			//*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
			//	"0", txStatus, true, &params)
		}
	case "SwapSigner", "AddSigner", "RemoveSigner":
		{
			//params, err := ParseMsigParams(trace.Msg, height, key, lib)
			//if err == nil {
			//	var paramsMap map[string]interface{}
			//	if err := json.Unmarshal([]byte(params), &paramsMap); err == nil {
			//		switch baseMethod {
			//		case "SwapSigner":
			//			{
			//				*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
			//					"0", txStatus, false, &paramsMap)
			//				*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
			//					"0", txStatus, true, &paramsMap)
			//			}
			//		case "AddSigner", "RemoveSigner":
			//			{
			//				*transactions = AppendOp(*transactions, baseMethod, fromAdd.GetAddress(),
			//					"0", txStatus, false, &paramsMap)
			//				*transactions = AppendOp(*transactions, baseMethod, toAdd.GetAddress(),
			//					"0", txStatus, true, &paramsMap)
			//			}
			//		}
			//
			//	} else {
			//		rosetta.Logger.Error("Could not parse message params for", baseMethod)
			//	}
			//}
		}
	case "AwardBlockReward", "ApplyRewards", "OnDeferredCronEvent",
		"PreCommitSector", "ProveCommitSector", "SubmitWindowedPoSt",
		"DeclareFaultsRecovered", "ChangeWorkerAddress", "PreCommitSectorBatch",
		"ProveCommitAggregate", "ProveReplicaUpdates":
		{
			// We don't add anything here to tx
			break
		}
	case "Approve", "Cancel":
		{
			// We don't add anything here to tx
			break
		}
	}

	transactions = append(transactions, tx)

	// Only process sub-calls if the parent call was successfully executed
	if txStatus == TransactionStatusOk {
		for i := range trace.Subcalls {
			subTrace := trace.Subcalls[i]
			t.ProcessTrace(&subTrace, msgCid, transactions, rawTipset, addresses)
		}
	}
}

func (t *TransactionParser) searchForActorCreation(msg *filTypes.Message, receipt *filTypes.MessageReceipt,
	height int64, key filTypes.TipSetKey) (*types2.AddressInfo, error) {

	toAddressInfo := tools.GetActorAddressInfo(msg.To, height, key, t.rosettaLib)
	actorName, err := t.rosettaLib.BuiltinActors.GetActorNameFromCid(toAddressInfo.ActorCid)
	if err != nil {
		return nil, err
	}

	switch actorName {
	case "init":
		{
			params, err := ParseInitActorExecParams(msg.Params)
			if err != nil {
				return nil, err
			}
			createdActorName, err := t.rosettaLib.BuiltinActors.GetActorNameFromCid(params.CodeCID)
			if err != nil {
				return nil, err
			}
			switch createdActorName {
			case "multisig", "paymentchannel":
				{
					execReturn, err := ParseExecReturn(receipt.Return)
					if err != nil {
						return nil, err
					}

					return &types2.AddressInfo{
						Short:     execReturn.IDAddress.String(),
						Robust:    execReturn.RobustAddress.String(),
						ActorCid:  params.CodeCID,
						ActorType: createdActorName,
					}, nil
				}
			default:
				return nil, nil
			}
		}
	case "storagepower":
		{
			execReturn, err := ParseExecReturn(receipt.Return)
			if err != nil {
				return nil, err
			}
			return &types2.AddressInfo{
				Short:     execReturn.IDAddress.String(),
				Robust:    execReturn.RobustAddress.String(),
				ActorType: "miner",
			}, nil
		}
	default:
		return nil, nil
	}
}

func appendAddressInfo(addressMap *types2.AddressInfoMap, info ...types2.AddressInfo) {
	if addressMap == nil {
		return
	}
	for _, i := range info {
		if i.Robust != "" && i.Short != "" && i.Robust != i.Short {
			if _, ok := (*addressMap)[i.Short]; !ok {
				(*addressMap)[i.Short] = i
			}
		}
	}
}
