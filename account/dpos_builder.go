package account

import (
	"context"
	"encoding/json"

	"github.com/vapor/config"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
)

func (m *Manager) DecodeDposAction(data []byte) (txbuilder.Action, error) {
	a := &DopsAction{Accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type DopsAction struct {
	Accounts *Manager
	bc.AssetAmount
	From           string `json:"from"`
	To             string `json:"to"`
	Fee            uint64 `json:"fee"`
	UseUnconfirmed bool   `json:"use_unconfirmed"`
}

func (a *DopsAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string

	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.From == "" {
		missing = append(missing, "from")
	}
	if a.To == "" {
		missing = append(missing, "to")
	}

	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}
	res, err := a.Accounts.utxoKeeper.ReserveByAddress(a.From, a.AssetId, a.Amount, a.UseUnconfirmed, false)
	if err != nil {
		return errors.Wrap(err, "reserving utxos")
	}

	// Cancel the reservation if the build gets rolled back.
	b.OnRollback(func() { a.Accounts.utxoKeeper.Cancel(res.id) })
	for _, r := range res.utxos {
		txInput, sigInst, err := DposTx(a.From, a.To, a.Amount, r)
		if err != nil {
			return errors.Wrap(err, "creating inputs")
		}
		if err = b.AddInput(txInput, sigInst); err != nil {
			return errors.Wrap(err, "adding inputs")
		}
	}

	res, err = a.Accounts.utxoKeeper.ReserveByAddress(a.From, a.AssetId, a.Fee, a.UseUnconfirmed, true)
	if err != nil {
		return errors.Wrap(err, "reserving utxos")
	}

	// Cancel the reservation if the build gets rolled back.
	b.OnRollback(func() { a.Accounts.utxoKeeper.Cancel(res.id) })
	for _, r := range res.utxos {
		txSpendInput, sigInst, err := spendInput(r)
		if err != nil {
			return errors.Wrap(err, "creating inputs")
		}

		if err = b.AddInput(txSpendInput, sigInst); err != nil {
			return errors.Wrap(err, "adding inputs")
		}
	}
	if res.change >= 0 {
		address, err := common.DecodeAddress(a.From, &consensus.ActiveNetParams)
		if err != nil {
			return err
		}
		redeemContract := address.ScriptAddress()
		program, err := vmutil.P2WPKHProgram(redeemContract)
		if err != nil {
			return err
		}
		if err = b.AddOutput(types.NewTxOutput(*consensus.BTMAssetID, res.change, program)); err != nil {
			return errors.Wrap(err, "adding change output")
		}
	}

	return nil
}

func (a *DopsAction) ActionType() string {
	return "dpos"
}

// DposInputs convert an utxo to the txinput
func DposTx(from, to string, stake uint64, u *UTXO) (*types.TxInput, *txbuilder.SigningInstruction, error) {
	txInput := types.NewDpos(nil, from, to, u.SourceID, u.AssetID, stake, u.Amount, u.SourcePos, u.ControlProgram, types.Delegate)
	sigInst := &txbuilder.SigningInstruction{}
	var xpubs []chainkd.XPub
	var xprv chainkd.XPrv
	xprv.UnmarshalText([]byte(config.CommonConfig.Consensus.Dpos.XPrv))
	xpubs = append(xpubs, xprv.XPub())
	quorum := len(xpubs)
	if u.Address == "" {
		sigInst.AddWitnessKeysWithOutPath(xpubs, quorum)
		return txInput, sigInst, nil
	}

	address, err := common.DecodeAddress(u.Address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, nil, err
	}
	sigInst.AddRawWitnessKeysWithoutPath(xpubs, quorum)
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		derivedPK := xpubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		derivedXPubs := xpubs
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, quorum)
		if err != nil {
			return nil, nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(script))

	default:
		return nil, nil, errors.New("unsupport address type")
	}

	return txInput, sigInst, nil
}

// spendInput convert an utxo to the txinput
func spendInput(u *UTXO) (*types.TxInput, *txbuilder.SigningInstruction, error) {
	txSpendInput := types.NewSpendInput(nil, u.SourceID, u.AssetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &txbuilder.SigningInstruction{}
	var xpubs []chainkd.XPub
	var xprv chainkd.XPrv
	xprv.UnmarshalText([]byte(config.CommonConfig.Consensus.Dpos.XPrv))
	xpubs = append(xpubs, xprv.XPub())
	quorum := len(xpubs)
	if u.Address == "" {
		sigInst.AddWitnessKeysWithOutPath(xpubs, quorum)
		return txSpendInput, sigInst, nil
	}

	address, err := common.DecodeAddress(u.Address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, nil, err
	}
	sigInst.AddRawWitnessKeysWithoutPath(xpubs, quorum)
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		derivedPK := xpubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		derivedXPubs := xpubs
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, quorum)
		if err != nil {
			return nil, nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(script))

	default:
		return nil, nil, errors.New("unsupport address type")
	}

	return txSpendInput, sigInst, nil
}
