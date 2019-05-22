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
	SourceID        string                       `json:"source_id"` // AnnotatedUTXO
	SourcePos       uint64                       `json:"source_pos"`
	Program         json.HexBytes                `json:"control_program"`
	AssetDefinition map[string]interface{}       `json:"asset_definition"`
	UpdateAssetDef  bool                         `json:"update_asset_definition"`
	Arguments       []txbuilder.ContractArgument `json:"arguments"`
}

// type AnnotatedInput struct {
//  Type             string               `json:"type"`
//  AssetID          bc.AssetID           `json:"asset_id"`
//  AssetAlias       string               `json:"asset_alias,omitempty"`
//  AssetDefinition  *json.RawMessage     `json:"asset_definition,omitempty"`
//  Amount           uint64               `json:"amount"`
//  ControlProgram   chainjson.HexBytes   `json:"control_program,omitempty"`
//  Address          string               `json:"address,omitempty"`
//  SpentOutputID    *bc.Hash             `json:"spent_output_id,omitempty"`
//  AccountID        string               `json:"account_id,omitempty"`
//  AccountAlias     string               `json:"account_alias,omitempty"`
//  Arbitrary        chainjson.HexBytes   `json:"arbitrary,omitempty"`
//  InputID          bc.Hash              `json:"input_id"`
//  WitnessArguments []chainjson.HexBytes `json:"witness_arguments"`
// }

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

	sourceID := testutil.MustDecodeHash(a.SourceID)
	// input's arguments will be set when signing
	// arguments?
	// in :=  types.NewCrossChainInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram, assetDefinition []byte)
	txin := types.NewCrossChainInput(nil, sourceID, *a.AssetId, a.Amount, a.SourcePos, a.Program, rawDefinition)
	log.Info("cross-chain input action build")
	builder.RestrictMinTime(time.Now())
	return builder.AddInput(txin, &txbuilder.SigningInstruction{})
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}
