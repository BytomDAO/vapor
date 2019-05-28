package asset

import (
	"context"
	stdjson "encoding/json"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/consensus/federation"
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
	SourceID        bc.Hash                `json:"source_id"`
	SourcePos       uint64                 `json:"source_pos"`
	AssetDefinition map[string]interface{} `json:"asset_definition"`
}

func (a *crossInAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.SourceID.IsZero() {
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

	rawDefinitionByte, err := serializeAssetDef(a.AssetDefinition)
	if err != nil {
		return ErrSerializing
	}

	// arguments will be set when materializeWitnesses
	txin := types.NewCrossChainInput(nil, a.SourceID, *a.AssetId, a.Amount, a.SourcePos, nil, rawDefinitionByte)
	tplIn := &txbuilder.SigningInstruction{}
	fed := federation.GetFederation()
	tplIn.AddRawWitnessKeys(fed.XPubs, nil, fed.Quorum)
	return builder.AddInput(txin, tplIn)
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}
