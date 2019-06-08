package synchron

import (
	"bytes"
	"encoding/hex"
	// "encoding/json"
	// "fmt"
	// "math/big"
	// "sort"

	// "github.com/bytom/consensus"
	// "github.com/bytom/consensus/segwit"
	// "github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/vm/vmutil"
	// "github.com/blockcenter/coin/btm"
	// "github.com/blockcenter/config"
	// "github.com/blockcenter/types"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

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
	processIssuing(db *gorm.DB, txs []*btmTypes.Tx) error
	processChainInfo() error
	// getCoin() *orm.Coin
	// getTxStatus() *bc.TransactionStatus
	// processAddressTransaction(mappings []*addressTxMapping) error
	// processSpendBalance(input *btmTypes.TxInput, deltaBalance *deltaBalance)
	// processReceiveBalance(output *btmTypes.TxOutput, deltaBalance *deltaBalance)
	// processSpendUTXO(utxoIDList []string) error
	// processReceiveUTXO(m *addressTxMapping) error
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

		for _, depositTx := range filterDepositFromMainchain(block) {
			crossChainInputs := getRawCrossChainInputs(depositTx)
			log.Info(crossChainInputs)
		}

		filterWithdrawalToMainchain(block)

	default:
		block := bp.getBlock().(*vaporTypes.Block)
		filterDepositToSidechain(block)
		filterWithdrawalFromSidechain(block)
	}

	// txs := bp.getBlock().Transactions
	// addressTxMappings, err := GetAddressTxMappings(cfg, txs, bp.getTxStatus(), db)
	// if err != nil {
	// 	return err
	// }

	// if err := bp.processAddressTransaction(addressTxMappings); err != nil {
	// 	return err
	// }

	// if err := updateBalanceAndUTXO(db, addressTxMappings, bp); err != nil {
	// 	return err
	// }

	if err := updateDeletedTransaction(db); err != nil {
		return err
	}

	return bp.processChainInfo()
}

func filterDepositFromMainchain(block *btmTypes.Block) []*btmTypes.Tx {
	depositTxs := []*btmTypes.Tx{}
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			fedProg := vaporCfg.FederationProgrom(vaporCfg.CommonConfig)
			if bytes.Equal(output.OutputCommitment.ControlProgram, fedProg) {
				depositTxs = append(depositTxs, tx)
				break
			}
		}
	}
	return depositTxs
}

func filterWithdrawalToMainchain(block *btmTypes.Block) []*btmTypes.Tx {
	withdrawalTxs := []*btmTypes.Tx{}
	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			fedProg := vaporCfg.FederationProgrom(vaporCfg.CommonConfig)
			if bytes.Equal(input.ControlProgram(), fedProg) {
				withdrawalTxs = append(withdrawalTxs, tx)
				break
			}
		}
	}
	return withdrawalTxs
}

func filterDepositToSidechain(block *vaporTypes.Block) []*vaporTypes.Tx {
	depositTxs := []*vaporTypes.Tx{}
	for _, tx := range block.Transactions {
		for _, input := range tx.Inputs {
			if input.InputType() == vaporTypes.CrossChainInputType {
				break
			}
		}
	}
	return depositTxs
}

func filterWithdrawalFromSidechain(block *vaporTypes.Block) []*vaporTypes.Tx {
	withdrawalTxs := []*vaporTypes.Tx{}
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			if output.OutputType() == vaporTypes.CrossChainOutputType {
				break
			}
		}
	}
	return withdrawalTxs
}

func getRawCrossChainInputs(tx *btmTypes.Tx) []*orm.CrossTransactionInput {
	// break
	script := hex.EncodeToString(tx.Inputs[0].ControlProgram())
	inputs := []*orm.CrossTransactionInput{}
	for i, rawOutput := range tx.Outputs {
		input := &orm.CrossTransactionInput{
			// MainchainTxID uint64
			// SidechainTxID sql.NullInt64
			SourcePos: uint64(i),
			// AssetID: rawOutput.OutputCommitment.AssetAmount.assetID,
			AssetAmount: rawOutput.OutputCommitment.AssetAmount.Amount,
			Script:      script,
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func getRefCrossChainInputs(tx *vaporTypes.Tx) []*orm.CrossTransactionInput {
	inputs := []*orm.CrossTransactionInput{}
	for i, rawInput := range tx.Inputs {
		if rawInput.InputType() != vaporTypes.CrossChainInputType {
			continue
		}

		input := &orm.CrossTransactionInput{
			// MainchainTxID uint64
			// SidechainTxID sql.NullInt64
			SourcePos: uint64(i),
			// AssetID:  rawInput.AssetID(),
			AssetAmount: rawInput.Amount(),
			// Script:      "",
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func getRawCrossChainOutputs(tx *vaporTypes.Tx) []*orm.CrossTransactionOutput {
	outputs := []*orm.CrossTransactionOutput{}
	for i, rawOutput := range tx.Outputs {
		if rawOutput.OutputType() != vaporTypes.CrossChainOutputType {
			continue
		}

		output := &orm.CrossTransactionOutput{
			// SidechainTxID uint64
			// MainchainTxID sql.NullInt64
			SourcePos: uint64(i),
			// AssetID       uint64
			AssetAmount: rawOutput.AssetAmount().Amount,
			Script:      hex.EncodeToString(rawOutput.ControlProgram()),
		}
		outputs = append(outputs, output)
	}
	return outputs
}

func getRefCrossChainOutputs(tx *btmTypes.Tx) []*orm.CrossTransactionOutput {
	outputs := []*orm.CrossTransactionOutput{}
	return outputs
}

// An expired unconfirmed transaction will be marked as deleted, but the latter transaction was packaged into block,
// the deleted_at flag must be removed. In addition, the gorm can't support update deleted_at field directly, can only use raw sql.
func updateDeletedTransaction(db *gorm.DB) error {
	return db.Exec("UPDATE cross_transactions SET deleted_at = NULL WHERE block_height > 0 AND deleted_at IS NOT NULL").Error
}
