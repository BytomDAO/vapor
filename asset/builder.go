package asset

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/blockchain/signers"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
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
	SourceID        string                 `json:"source_id"`
	SourcePos       uint64                 `json:"source_pos"`
	AssetDefinition map[string]interface{} `json:"asset_definition"`
}

// TODO: also need to hard-code mapTx
func (a *crossInAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
	var missing []string
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

	sourceKey := []byte(fmt.Sprintf("SC:%v:%v", a.SourceID, a.SourcePos))
	a.reg.assetMu.Lock()
	defer a.reg.assetMu.Unlock()
	if existed := a.reg.db.Get(sourceKey); existed != nil {
		return errors.New("mainchain output double spent")
	}

	var err error
	asset := &Asset{}
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
		asset.AssetID = *a.AssetId
		extAlias := a.AssetId.String()
		asset.Alias = &(extAlias)
		a.reg.SaveExtAsset(asset, extAlias)
	}

	assetSigner, err := signers.Create("asset", consensus.Federation().XPubs, consensus.Federation().Quorum, 1, signers.BIP0032)
	if err != nil {
		return err
	}

	path := signers.GetBip0032Path(assetSigner, signers.AssetKeySpace)
	derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	pegInScript, err := buildPegInScript(derivedPKs, assetSigner.Quorum)
	if err != nil {
		return err
	}

	var sourceID bc.Hash
	if err := sourceID.UnmarshalText([]byte(a.SourceID)); err != nil {
		return errors.New("invalid sourceID format")
	}

	// arguments will be set when materializeWitnesses
	txin := types.NewCrossChainInput(nil, sourceID, *a.AssetId, a.Amount, a.SourcePos, pegInScript, asset.RawDefinitionByte)
	log.Info("cross-chain input action built")
	builder.RestrictMinTime(time.Now())
	tplIn := &txbuilder.SigningInstruction{}
	tplIn.AddRawWitnessKeys(assetSigner.XPubs, path, assetSigner.Quorum)
	a.reg.db.Set(sourceKey, []byte("true"))
	return builder.AddInput(txin, tplIn)
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}

func buildPegInScript(pubkeys []ed25519.PublicKey, nrequired int) (program []byte, err error) {
	controlProg, err := vmutil.P2SPMultiSigProgram(pubkeys, nrequired)
	if err != nil {
		return nil, err
	}
	builder := vmutil.NewBuilder()
	builder.AddRawBytes(controlProg)
	prog, err := builder.Build()
	return prog, err
}
