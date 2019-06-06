package synchron

import (
	// "encoding/hex"
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
	// "github.com/blockcenter/database/orm"
	// "github.com/blockcenter/types"
)

func addIssueAssets(db *gorm.DB, txs []*btmTypes.Tx) error {
	/*
		var assets []*orm.Asset
		assetMap := make(map[string]bool)

		type assetDefinition struct {
			Decimals uint64 `json:"decimals"`
		}

		for _, tx := range txs {
			for _, input := range tx.Inputs {
				switch inp := input.TypedInput.(type) {
				case *btmTypes.IssuanceInput:
					assetID := inp.AssetID()
					if _, ok := assetMap[assetID.String()]; ok {
						continue
					}
					assetMap[assetID.String()] = true

					asset := &orm.Asset{}
					definition := &assetDefinition{}
					if err := json.Unmarshal(inp.AssetDefinition, definition); err != nil {
						log.WithFields(log.Fields{
							"err":             err,
							"AssetDefinition": inp.AssetDefinition,
						}).Error("json unmarshal AssetDefinition")
					}

					asset.CoinID = coinID
					asset.Decimals = definition.Decimals
					asset.Asset = assetID.String()
					asset.Definition = string(inp.AssetDefinition)
					assets = append(assets, asset)
				}
			}
		}

		for _, asset := range assets {
			if err := db.Where(&orm.Asset{CoinID: asset.CoinID, Asset: asset.Asset}).FirstOrCreate(asset).Error; err != nil {
				return err
			}
		}
	*/
	return nil
}
