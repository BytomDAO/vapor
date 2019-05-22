package asset

import (
	"context"
	stdjson "encoding/json"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/protocol/bc"
	// chainjson "github.com/vapor/encoding/json"
	// "github.com/vapor/testutil"
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
	AssetDefinition map[string]interface{} `json:"asset_definition"`
	UpdateAssetDef  bool                   `json:"update_asset_definition"`
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
	return nil
	/*  var missing []string
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
	    rawDefinition, err := asset.SerializeAssetDef(a.AssetDefinition)
	    if err != nil {
	        return asset.ErrSerializing
	    }
	    // TODO: may need to skip here
	    if !chainjson.IsValidJSON(rawDefinition) {
	        return errors.New("asset definition is not in valid json format")
	    }
	    if preAsset, _ := a.accounts.assetReg.GetAsset(a.AssetId.String()); preAsset != nil {
	        // GetAsset() doesn't unmashall for RawDefinitionBytes
	        preRawDefinition, err := asset.SerializeAssetDef(preAsset.DefinitionMap)
	        if err != nil {
	            return asset.ErrSerializing
	        }

	        if !testutil.DeepEqual(preRawDefinition, rawDefinition) && !UpdateAssetDef {
	            return errors.New("asset definition mismatch with previous definition")
	        }
	        // TODO: update asset def here?
	    }

	    // TODO: IssuanceProgram vs arguments?
	    // TODO: also need to hard-code mapTx
	    // TODO: save AssetDefinition

	    // in :=  types.NewCrossChainInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount, sourcePos uint64, controlProgram, assetDefinition []byte)
	    // txin := types.NewIssuanceInput(nonce[:], a.Amount, asset.IssuanceProgram, nil, asset.RawDefinitionByte)
	    // input's arguments will be set when signing
	    sourceID := testutil.MustDecodeHash(a.SourceID)
	    txin := types.NewCrossChainInput(nil, sourceID, *a.AssetId, a.Amount, a.SourcePos, nil, rawDefinition)
	    tplIn := &txbuilder.SigningInstruction{}
	    if false {
	        // if asset.Signer != nil {
	        // path := signers.GetBip0032Path(asset.Signer, signers.AssetKeySpace)
	        // tplIn.AddRawWitnessKeys(asset.Signer.XPubs, path, asset.Signer.Quorum)
	    } else if a.Arguments != nil {
	        if err := txbuilder.AddContractArgs(tplIn, a.Arguments); err != nil {
	            return err
	        }
	    }

	    log.Info("cross-chain input action build")
	    builder.RestrictMinTime(time.Now())
	    return builder.AddInput(txin, tplIn)
	*/
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}
