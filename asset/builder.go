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
	"github.com/vapor/testutil"
)

// DecodeCrossInAction convert input data to action struct
func (r *Registry) DecodeCrossInAction(data []byte) (txbuilder.Action, error) {
	a := &crossInAction{assets: r}
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type crossInAction struct {
	assets *Registry
	bc.AssetAmount
	SourceID        string                 `json:"source_id"` // AnnotatedUTXO
	SourcePos       uint64                 `json:"source_pos"`
	Program         chainjson.HexBytes     `json:"control_program"`
	AssetDefinition map[string]interface{} `json:"asset_definition"`
	UpdateAssetDef  bool                   `json:"update_asset_definition"`
	Arguments       []chainjson.HexBytes   `json:"arguments"`
}

func (a *crossInAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
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

	// Handle asset definition.
	// Asset issuance's legality is guaranteed by the federation.
	rawDefinition, err := serializeAssetDef(a.AssetDefinition)
	if err != nil {
		return ErrSerializing
	}
	// TODO: may need to skip here
	if !chainjson.IsValidJSON(rawDefinition) {
		return errors.New("asset definition is not in valid json format")
	}
	if preAsset, _ := a.assets.GetAsset(a.AssetId.String()); preAsset != nil {
		// GetAsset() doesn't unmashall for RawDefinitionBytes
		preRawDefinition, err := serializeAssetDef(preAsset.DefinitionMap)
		if err != nil {
			return ErrSerializing
		}

		if !testutil.DeepEqual(preRawDefinition, rawDefinition) && !a.UpdateAssetDef {
			return errors.New("asset definition mismatch with previous definition")
		}
		// TODO: update asset def here?
	}

	// TODO: also need to hard-code mapTx
	// TODO: save AssetDefinition

	arguments := [][]byte{}
	for _, argument := range a.Arguments {
		arguments = append(arguments, argument)
	}
	sourceID := testutil.MustDecodeHash(a.SourceID)
	txin := types.NewCrossChainInput(arguments, sourceID, *a.AssetId, a.Amount, a.SourcePos, a.Program, rawDefinition)
	log.Info("cross-chain input action build")
	builder.RestrictMinTime(time.Now())
	return builder.AddInput(txin, &txbuilder.SigningInstruction{})
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}
