package asset

import (
	"context"
	stdjson "encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/blockchain/txbuilder"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

// DecodeCrossInAction convert input data to action struct
func (r *Registry) DecodeCrossInAction(data []byte) (txbuilder.Action, error) {
	a := &crossInAction{reg: r}
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type crossInAction struct {
	reg *Registry
	bc.AssetAmount
	SourceID        string                 `json:"source_id"` // AnnotatedUTXO
	SourcePos       uint64                 `json:"source_pos"`
	Program         chainjson.HexBytes     `json:"control_program"`
	AssetDefinition map[string]interface{} `json:"asset_definition"`
	Arguments       []chainjson.HexBytes   `json:"arguments"`
}

// func (reg *Registry) DefineCrossInAsset(a *Asset, alias string) error {
//     defHash := bc.NewHash(sha3.Sum256(rawDefinition))
//     a := &Asset{
//         DefinitionMap:     definition,
//         RawDefinitionByte: rawDefinition,
//         VMVersion:         vmver,
//         IssuanceProgram:   issuanceProgram,
//         AssetID:           bc.ComputeAssetID(issuanceProgram, vmver, &defHash),
//         Alias:             &alias,
//     }
//     return a, reg.SaveAsset(a, alias)
// }

// TODO: also need to hard-code mapTx
// TODO: iter cross-in and save asset
// TODO: federation can sign? check arguments length?
func (a *crossInAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
	}
	if a.SourceID == "" {
		missing = append(missing, "source_id")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	asset := &Asset{}
	var err error
	// Handle asset definition.
	// Asset issuance's legality is guaranteed by the federation.
	if preAsset, _ := a.reg.GetAsset(a.AssetId.String()); preAsset != nil {
		asset = preAsset
	} else {
		asset.RawDefinitionByte, err = serializeAssetDef(a.AssetDefinition)
		if err != nil {
			return ErrSerializing
		}

		if !chainjson.IsValidJSON(asset.RawDefinitionByte) {
			return errors.New("asset definition is not in valid json format")
		}

		asset.DefinitionMap = a.AssetDefinition
		asset.VMVersion = 1
		// TODO: asset.IssuanceProgram
		asset.AssetID = *a.AssetId
		extAlias := a.AssetId.String()
		asset.Alias = &(extAlias)
		a.reg.SaveAsset(asset, extAlias)
	}

	arguments := [][]byte{}
	for _, argument := range a.Arguments {
		arguments = append(arguments, argument)
	}

	var sourceID bc.Hash
	if err := sourceID.UnmarshalText([]byte(a.SourceID)); err != nil {
		return errors.New("invalid source ID format")
	}

	txin := types.NewCrossChainInput(arguments, sourceID, *a.AssetId, a.Amount, a.SourcePos, a.Program, asset.RawDefinitionByte)
	log.Info("cross-chain input action build")
	builder.RestrictMinTime(time.Now())
	return builder.AddInput(txin, &txbuilder.SigningInstruction{})
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}
