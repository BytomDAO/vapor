package synchron

import (
	"encoding/hex"
	// "encoding/json"
	// "fmt"
	// "math/big"
	// "sort"

	// "github.com/bytom/consensus"
	// "github.com/bytom/consensus/segwit"
	// "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	// "github.com/bytom/protocol/vm/vmutil"
	"github.com/jinzhu/gorm"
	// log "github.com/sirupsen/logrus"
	// "github.com/blockcenter/coin/btm"
	// "github.com/blockcenter/config"
	// "github.com/blockcenter/types"
	"github.com/vapor/errors"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
)

var ErrInconsistentDB = errors.New("inconsistent db status")

type blockProcessor interface {
	getCfg() *config.Chain
	processIssuing(db *gorm.DB, txs []*btmTypes.Tx) error
	processChainInfo() error
	// getBlock() *btmTypes.Block
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
	// txs := bp.getBlock().Transactions
	if bp.getCfg().IsMainchain {
		// if err := bp.processIssuing(db, txs); err != nil {
		// 	return err
		// }
	}

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

// An expired unconfirmed transaction will be marked as deleted, but the latter transaction was packaged into block,
// the deleted_at flag must be removed. In addition, the gorm can't support update deleted_at field directly, can only use raw sql.
func updateDeletedTransaction(db *gorm.DB) error {
	return db.Exec("UPDATE cross_transactions SET deleted_at = NULL WHERE block_height > 0 AND deleted_at IS NOT NULL").Error
}
