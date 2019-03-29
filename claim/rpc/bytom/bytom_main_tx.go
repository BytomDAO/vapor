package bytom

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/claim/bytom/mainchain"
	bytomtypes "github.com/vapor/claim/bytom/protocolbc/types"
	"github.com/vapor/claim/rpc"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/equity/pegin_contract"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
)

type BytomMainTx struct {
	rpc.MainTxParam
}

func (b *BytomMainTx) BuildMainChainTxForContract() ([]byte, error) {
	var xpubs []chainkd.XPub
	for _, pub := range b.Pubs {
		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(pub)); err != nil {
			return nil, err
		}
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(b.ClaimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		chaildXPub := xpub.Child(tweak)
		xpubs = append(xpubs, chaildXPub)
	}

	utxo := &account.UTXO{}
	if err := json.Unmarshal([]byte(b.Utxo), utxo); err != nil {
		return nil, err
	}
	txInput, sigInst, err := contractToInputs(utxo, xpubs, b.ClaimScript)
	builder := mainchain.NewBuilder(time.Now())
	builder.AddInput(txInput, sigInst)
	changeAmount := uint64(0)
	retire := false
	tx := &types.Tx{}
	if err := tx.UnmarshalText([]byte(b.Tx)); err != nil {
		return nil, err
	}
	for _, key := range tx.GetResultIds() {
		output, err := tx.Retire(*key)
		if err != nil {
			log.WithFields(log.Fields{"moudle": "transact", "err": err}).Warn("buildMainChainTx error")
			continue
		}
		retire = true
		var controlProgram []byte
		retBool := true
		if controlProgram, retBool = getInput(tx.Entries, *key, b.ControlProgram); !retBool {
			return nil, errors.New("The corresponding input cannot be found")
		}

		assetID := *output.Source.Value.AssetId
		out := bytomtypes.NewTxOutput(assetID, output.Source.Value.Amount, controlProgram)
		builder.AddOutput(out)
		changeAmount = utxo.Amount - output.Source.Value.Amount
	}

	if !retire {
		return nil, errors.New("It's not a transaction to retire assets")
	}

	if changeAmount > 100000000 {
		u := utxo
		out := bytomtypes.NewTxOutput(u.AssetID, changeAmount, utxo.ControlProgram)
		builder.AddOutput(out)
	}

	tmpl, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	for i, out := range tmpl.Transaction.Outputs {
		if bytes.Equal(out.ControlProgram, utxo.ControlProgram) {
			if changeAmount-100000000 > 0 {
				txData.Outputs[i].Amount = changeAmount - 100000000
			}
		}
	}
	tmpl.Transaction = bytomtypes.NewTx(*txData)
	resp, err := mainchain.MarshalText(tmpl)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *BytomMainTx) BuildMainChainTx() ([]byte, error) {
	var xpubs []chainkd.XPub
	for _, pub := range b.Pubs {

		xpub := chainkd.XPub{}
		if err := xpub.UnmarshalText([]byte(pub)); err != nil {
			return nil, err
		}
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(b.ClaimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		chaildXPub := xpub.Child(tweak)
		xpubs = append(xpubs, chaildXPub)
	}

	utxo := &account.UTXO{}
	if err := json.Unmarshal([]byte(b.Utxo), utxo); err != nil {
		return nil, err
	}

	txInput, sigInst, err := utxoToInputs(xpubs, utxo)
	if err != nil {
		return nil, err
	}

	builder := mainchain.NewBuilder(time.Now())
	builder.AddInput(txInput, sigInst)
	changeAmount := uint64(0)
	retire := false
	tx := &types.Tx{}
	if err := tx.UnmarshalText([]byte(b.Tx)); err != nil {
		return nil, err
	}
	for _, key := range tx.GetResultIds() {
		output, err := tx.Retire(*key)
		if err != nil {
			log.WithFields(log.Fields{"moudle": "transact", "err": err}).Warn("buildMainChainTx error")
			continue
		}
		retire = true
		var controlProgram []byte
		retBool := true
		if controlProgram, retBool = getInput(tx.Entries, *key, b.ControlProgram); !retBool {
			return nil, errors.New("The corresponding input cannot be found")
		}

		assetID := *output.Source.Value.AssetId
		out := bytomtypes.NewTxOutput(assetID, output.Source.Value.Amount, controlProgram)
		builder.AddOutput(out)
		changeAmount = utxo.Amount - output.Source.Value.Amount

	}

	if !retire {
		return nil, errors.New("It's not a transaction to retire assets")
	}

	if changeAmount > 0 {
		u := utxo
		out := bytomtypes.NewTxOutput(u.AssetID, changeAmount, utxo.ControlProgram)
		builder.AddOutput(out)
	}

	tmpl, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}
	/*
		//交易费估算
		txGasResp, err := EstimateTxGasForMainchain(*tmpl)
		if err != nil {
			return nil, err
		}
	*/
	for i, out := range tmpl.Transaction.Outputs {
		if bytes.Equal(out.ControlProgram, utxo.ControlProgram) {
			//tx.Outputs[i].Amount = changeAmount - uint64(txGasResp.TotalNeu)
			tx.Outputs[i].Amount = changeAmount - 100000000
		}
	}
	tmpl.Transaction = bytomtypes.NewTx(*txData)
	resp, err := json.Marshal(tmpl)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

type BytomMainSign struct {
	rpc.MainTxSignParam
}

func (b *BytomMainSign) SignWithKey() (interface{}, error) {
	xprv := &chainkd.XPrv{}
	if err := xprv.UnmarshalText([]byte(b.Xprv)); err != nil {
		return nil, err
	}

	xpub := &chainkd.XPub{}
	if err := xpub.UnmarshalText([]byte(b.XPub)); err != nil {
		return nil, err
	}

	// pub + scriptPubKey 生成一个随机数A
	var tmp [32]byte
	h := hmac.New(sha256.New, xpub[:])
	h.Write(b.ClaimScript)
	tweak := h.Sum(tmp[:])
	// pub +  A 生成一个新的公钥pub_new
	privateKey := xprv.Child(tweak, false)

	txs := &mainchain.Template{}
	if err := mainchain.UnmarshalText([]byte(b.Txs), txs); err != nil {
		return nil, err
	}

	if err := sign(txs, privateKey); err != nil {
		return nil, err
	}
	log.Info("Sign Transaction complete.")
	return struct {
		Tx           *mainchain.Template `json:"transaction"`
		SignComplete bool                `json:"sign_complete"`
	}{Tx: txs, SignComplete: mainchain.SignProgress(txs)}, nil
}

func sign(tmpl *mainchain.Template, xprv chainkd.XPrv) error {
	for i, sigInst := range tmpl.SigningInstructions {
		for j, wc := range sigInst.WitnessComponents {
			switch sw := wc.(type) {
			case *mainchain.SignatureWitness:
				err := sw.Sign(tmpl, uint32(i), xprv)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to signature witness component %d of input %d", j, i)
				}
			case *mainchain.RawTxSigWitness:
				err := sw.Sign(tmpl, uint32(i), xprv)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to raw-signature witness component %d of input %d", j, i)
				}
			}
		}
	}
	return materializeWitnesses(tmpl)
}

func materializeWitnesses(txTemplate *mainchain.Template) error {
	msg := txTemplate.Transaction

	if msg == nil {
		return errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	if len(txTemplate.SigningInstructions) > len(msg.Inputs) {
		return errors.Wrap(txbuilder.ErrBadInstructionCount)
	}

	for i, sigInst := range txTemplate.SigningInstructions {
		if msg.Inputs[sigInst.Position] == nil {
			return errors.WithDetailf(txbuilder.ErrBadTxInputIdx, "signing instruction %d references missing tx input %d", i, sigInst.Position)
		}

		var witness [][]byte
		for j, wc := range sigInst.WitnessComponents {
			err := wc.Materialize(&witness)
			if err != nil {
				return errors.WithDetailf(err, "error in witness component %d of input %d", j, i)
			}
		}
		msg.SetInputArguments(sigInst.Position, witness)
	}

	return nil
}

//
func getInput(entry map[bc.Hash]bc.Entry, outputID bc.Hash, controlProgram string) ([]byte, bool) {
	output := entry[outputID].(*bc.Retirement)
	mux := entry[*output.Source.Ref].(*bc.Mux)

	for _, valueSource := range mux.GetSources() {
		spend := entry[*valueSource.Ref].(*bc.Spend)
		prevout := entry[*spend.SpentOutputId].(*bc.Output)

		var ctrlProgram chainjson.HexBytes
		ctrlProgram = prevout.ControlProgram.Code
		tmp, _ := ctrlProgram.MarshalText()
		if string(tmp) == controlProgram {
			return ctrlProgram, true
		}
	}
	return nil, false
}

// UtxoToInputs convert an utxo to the txinput
func utxoToInputs(xpubs []chainkd.XPub, u *account.UTXO) (*bytomtypes.TxInput, *mainchain.SigningInstruction, error) {
	txInput := bytomtypes.NewSpendInput(nil, u.SourceID, u.AssetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &mainchain.SigningInstruction{}
	quorum := len(xpubs)
	if u.Address == "" {
		sigInst.AddWitnessKeys(xpubs, quorum)
		return txInput, sigInst, nil
	}

	address, err := common.DecodeBytomAddress(u.Address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, nil, err
	}

	sigInst.AddRawWitnessKeysWithoutPath(xpubs, quorum)

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		derivedPK := xpubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, mainchain.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		derivedXPubs := xpubs
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, quorum)
		if err != nil {
			return nil, nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, mainchain.DataWitness(script))

	default:
		return nil, nil, errors.New("unsupport address type")
	}

	return txInput, sigInst, nil
}

func contractToInputs(u *account.UTXO, xpubs []chainkd.XPub, ClaimScript chainjson.HexBytes) (*bytomtypes.TxInput, *mainchain.SigningInstruction, error) {
	txInput := bytomtypes.NewSpendInput(nil, u.SourceID, u.AssetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &mainchain.SigningInstruction{}

	// raw_tx_signature
	for _, xpub := range xpubs {
		xpubsTmp := []chainkd.XPub{xpub}
		sigInst.AddRawWitnessKeysWithoutPath(xpubsTmp, 1)
	}

	// data
	peginContractPrograms, err := pegin_contract.GetPeginContractPrograms(ClaimScript)
	if err != nil {
		return nil, nil, err
	}
	sigInst.WitnessComponents = append(sigInst.WitnessComponents, mainchain.DataWitness(peginContractPrograms))

	return txInput, sigInst, nil
}
