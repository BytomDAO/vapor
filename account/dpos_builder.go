package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vapor/config"

	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	dpos "github.com/vapor/consensus/consensus/dpos"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
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
	DposType       uint32   `json:"dpos_type"`
	Address        string   `json:"address"`
	Name           string   `json:"name"`
	Forgers        []string `json:"forgers"`
	UseUnconfirmed bool     `json:"use_unconfirmed"`
}

func (a *DopsAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string

	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}
	if types.TxType(a.DposType) < types.Binary || types.TxType(a.DposType) > types.CancelVote {
		return errors.New("tx type  of dpos is error")
	}
	var (
		referenceData []byte
		data          []byte
		op            vm.Op
		err           error
	)

	switch types.TxType(a.DposType) {
	case types.Binary:
	case types.Registe:
		if a.Name == "" {
			return errors.New("name is null for dpos Registe")
		}
		if a.Amount < consensus.RegisrerForgerFee {
			return errors.New("The transaction fee is 100000000 for dpos Registe")
		}

		if dpos.GDpos.HaveDelegate(a.Name, a.Address) {
			return errors.New("Forger name has registe")
		}

		data, err = json.Marshal(&dpos.RegisterForgerData{Name: a.Name})
		if err != nil {
			return err
		}
		op = vm.OP_REGISTE
	case types.Vote:
		if len(a.Forgers) == 0 {
			return errors.New("Forgers is null for dpos Vote")
		}

		if a.Amount < consensus.VoteForgerFee {
			return errors.New("The transaction fee is 10000000 for dpos Registe")
		}

		for _, forger := range a.Forgers {
			if dpos.GDpos.HaveVote(a.Address, forger) {
				return fmt.Errorf("delegate name: %s is voted", forger)
			}
		}

		data, err = json.Marshal(&dpos.VoteForgerData{Forgers: a.Forgers})
		if err != nil {
			return err
		}
		op = vm.OP_VOTE
	case types.CancelVote:
		if len(a.Forgers) == 0 {
			return errors.New("Forgers is null for dpos CancelVote")
		}
		if a.Amount < consensus.CancelVoteForgerFee {
			return errors.New("The transaction fee is 10000000 for dpos Registe")
		}

		for _, forger := range a.Forgers {
			if !dpos.GDpos.HaveVote(a.Address, forger) {
				return fmt.Errorf("delegate name: %s is not voted", forger)
			}
		}

		data, err = json.Marshal(&dpos.CancelVoteForgerData{Forgers: a.Forgers})
		if err != nil {
			return err
		}
		op = vm.OP_REVOKE
	}

	msg := dpos.DposMsg{
		Type: op,
		Data: data,
	}

	referenceData, err = json.Marshal(&msg)
	if err != nil {
		return err
	}
	b.SetReferenceData(referenceData)

	res, err := a.Accounts.utxoKeeper.ReserveByAddress(a.Address, a.AssetId, a.Amount, a.UseUnconfirmed, false)
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
		address, err := common.DecodeAddress(a.Address, &consensus.ActiveNetParams)
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

// spendInput convert an utxo to the txinput
func spendInput(u *UTXO) (*types.TxInput, *txbuilder.SigningInstruction, error) {
	txSpendInput := types.NewSpendInput(nil, u.SourceID, u.AssetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &txbuilder.SigningInstruction{}
	var xpubs []chainkd.XPub
	var xprv chainkd.XPrv
	xprv.UnmarshalText([]byte(config.CommonConfig.Consensus.XPrv))
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
