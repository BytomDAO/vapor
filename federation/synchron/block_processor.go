package synchron

import (
	"bytes"
	"encoding/hex"
	// "encoding/json"
	// "fmt"

	// "github.com/bytom/consensus"
	// "github.com/bytom/consensus/segwit"
	// "github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/vm/vmutil"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	// log "github.com/sirupsen/logrus"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/errors"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

var ErrInconsistentDB = errors.New("inconsistent db status")

type blockProcessor interface {
	getCfg() *config.Chain
	getBlock() interface{}
	processWithdrawalToMainchain(uint64, *btmTypes.Tx) error
	processDepositFromMainchain(uint64, *btmTypes.Tx) error
	processDepositToSidechain(uint64, *vaporTypes.Tx) error
	processWithdrawalFromSidechain(uint64, *vaporTypes.Tx) error
	processIssuing(*gorm.DB, []*btmTypes.Tx) error
	processChainInfo() error
	// getTxStatus() *bc.TransactionStatus
}

func addIssueAssets(db *gorm.DB, txs []*btmTypes.Tx) error {
	var assets []*orm.Asset
	assetMap := make(map[string]bool)

	for _, tx := range txs {
		for _, input := range tx.Inputs {
			switch inp := input.TypedInput.(type) {
			case *btmTypes.IssuanceInput:
				assetID := inp.AssetID()
				if _, ok := assetMap[assetID.String()]; ok {
					continue
				}
				assetMap[assetID.String()] = true

				asset := &orm.Asset{
					AssetID:           assetID.String(),
					IssuanceProgram:   hex.EncodeToString(inp.IssuanceProgram),
					VMVersion:         inp.VMVersion,
					RawDefinitionByte: hex.EncodeToString(inp.AssetDefinition),
				}
				assets = append(assets, asset)
			}
		}
	}

	for _, asset := range assets {
		if err := db.Where(&orm.Asset{AssetID: asset.AssetID}).FirstOrCreate(asset).Error; err != nil {
			return err
		}
	}

	return nil
}

func updateBlock(db *gorm.DB, bp blockProcessor) error {
	switch {
	case bp.getCfg().IsMainchain:
		// Issuance can only happen on mainchain
		block := bp.getBlock().(*btmTypes.Block)
		txs := block.Transactions
		if err := bp.processIssuing(db, txs); err != nil {
			return err
		}

		for i, tx := range txs {
			if isDepositFromMainchain(tx) {
				bp.processDepositFromMainchain(uint64(i), tx)
			}
			if isWithdrawalToMainchain(tx) {
				bp.processWithdrawalToMainchain(uint64(i), tx)
			}
		}

	default:
		block := bp.getBlock().(*vaporTypes.Block)
		for i, tx := range block.Transactions {
			if isDepositToSidechain(tx) {
				bp.processDepositToSidechain(uint64(i), tx)
			}
			if isWithdrawalFromSidechain(tx) {
				bp.processWithdrawalFromSidechain(uint64(i), tx)
			}
		}
	}

	return bp.processChainInfo()
}

func isDepositFromMainchain(tx *btmTypes.Tx) bool {
	fedProg := vaporCfg.FederationProgrom(vaporCfg.CommonConfig)
	for _, output := range tx.Outputs {
		if bytes.Equal(output.OutputCommitment.ControlProgram, fedProg) {
			return true
		}
	}
	return false
}

func isWithdrawalToMainchain(tx *btmTypes.Tx) bool {
	fedProg := vaporCfg.FederationProgrom(vaporCfg.CommonConfig)
	for _, input := range tx.Inputs {
		if bytes.Equal(input.ControlProgram(), fedProg) {
			return true
		}
	}
	return false
}

func isDepositToSidechain(tx *vaporTypes.Tx) bool {
	for _, input := range tx.Inputs {
		if input.InputType() == vaporTypes.CrossChainInputType {
			return true
		}
	}
	return false
}

func isWithdrawalFromSidechain(tx *vaporTypes.Tx) bool {
	for _, output := range tx.Outputs {
		if output.OutputType() == vaporTypes.CrossChainOutputType {
			return true
		}
	}
	return false
}

func getCrossChainInputs(mainchainTxID uint64, tx *btmTypes.Tx) []*orm.CrossTransactionInput {
	// assume inputs are from an identical owner
	script := hex.EncodeToString(tx.Inputs[0].ControlProgram())
	inputs := []*orm.CrossTransactionInput{}
	for i, rawOutput := range tx.Outputs {
		fedProg := vaporCfg.FederationProgrom(vaporCfg.CommonConfig)
		// check valid deposit
		if !bytes.Equal(rawOutput.OutputCommitment.ControlProgram, fedProg) {
			continue
		}

		// default null SidechainTxID, which will be set after submitting deposit tx on sidechain
		input := &orm.CrossTransactionInput{
			MainchainTxID: mainchainTxID,
			SourcePos:     uint64(i),
			// AssetID: rawOutput.OutputCommitment.AssetAmount.assetID,
			AssetAmount: rawOutput.OutputCommitment.AssetAmount.Amount,
			Script:      script,
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func getCrossChainOutputs(sidechainTxID uint64, tx *vaporTypes.Tx) []*orm.CrossTransactionOutput {
	outputs := []*orm.CrossTransactionOutput{}
	for i, rawOutput := range tx.Outputs {
		if rawOutput.OutputType() != vaporTypes.CrossChainOutputType {
			continue
		}

		// default null MainchainTxID, which will be set after submitting withdrawal tx on mainchain
		output := &orm.CrossTransactionOutput{
			SidechainTxID: sidechainTxID,
			SourcePos:     uint64(i),
			// AssetID       uint64
			AssetAmount: rawOutput.AssetAmount().Amount,
			Script:      hex.EncodeToString(rawOutput.ControlProgram()),
		}
		outputs = append(outputs, output)
	}
	return outputs
}
