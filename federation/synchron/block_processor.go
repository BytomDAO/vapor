package synchron

import (
	"encoding/hex"
	// "encoding/json"
	// "fmt"
	// "math/big"
	// "sort"

	// "github.com/bytom/consensus"
	// "github.com/bytom/consensus/segwit"
	// "github.com/bytom/errors"
	// "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	// "github.com/bytom/protocol/vm/vmutil"
	"github.com/jinzhu/gorm"
	// log "github.com/sirupsen/logrus"
	// "github.com/blockcenter/coin/btm"
	// "github.com/blockcenter/config"
	// "github.com/blockcenter/types"
	"github.com/vapor/federation/database/orm"
)

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
