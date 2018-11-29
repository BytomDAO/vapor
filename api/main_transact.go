package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/blockchain/txbuilder/mainchain"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/bc/types/bytom"
	bytomtypes "github.com/vapor/protocol/bc/types/bytom/types"
	"github.com/vapor/protocol/vm/vmutil"
)

func (a *API) buildMainChainTxForContract(ins struct {
	Utxo           account.UTXO       `json:"utxo"`
	Tx             types.Tx           `json:"raw_transaction"`
	RootXPubs      []chainkd.XPub     `json:"root_xpubs"`
	ControlProgram string             `json:"control_program"`
	ClaimScript    chainjson.HexBytes `json:"claim_script"`
}) Response {

	var xpubs []chainkd.XPub
	for _, xpub := range ins.RootXPubs {
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(ins.ClaimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		chaildXPub := xpub.Child(tweak)
		xpubs = append(xpubs, chaildXPub)
	}

	txInput, sigInst, err := contractToInputs(a, &ins.Utxo, xpubs)
	builder := mainchain.NewBuilder(time.Now())
	builder.AddInput(txInput, sigInst)
	changeAmount := uint64(0)
	retire := false
	for _, key := range ins.Tx.GetResultIds() {
		output, err := ins.Tx.Retire(*key)
		if err != nil {
			log.WithFields(log.Fields{"moudle": "transact", "err": err}).Warn("buildMainChainTx error")
			continue
		}
		retire = true
		var controlProgram []byte
		retBool := true
		if controlProgram, retBool = getInput(ins.Tx.Entries, *key, ins.ControlProgram); !retBool {
			return NewErrorResponse(errors.New("The corresponding input cannot be found"))
		}

		assetID := bytom.AssetID{
			V0: output.Source.Value.AssetId.GetV0(),
			V1: output.Source.Value.AssetId.GetV1(),
			V2: output.Source.Value.AssetId.GetV2(),
			V3: output.Source.Value.AssetId.GetV3(),
		}
		out := bytomtypes.NewTxOutput(assetID, output.Source.Value.Amount, controlProgram)
		builder.AddOutput(out)
		changeAmount = ins.Utxo.Amount - output.Source.Value.Amount

	}

	if !retire {
		return NewErrorResponse(errors.New("It's not a transaction to retire assets"))
	}

	if changeAmount > 0 {
		u := ins.Utxo
		assetID := bytom.AssetID{
			V0: u.AssetID.GetV0(),
			V1: u.AssetID.GetV1(),
			V2: u.AssetID.GetV2(),
			V3: u.AssetID.GetV3(),
		}
		out := bytomtypes.NewTxOutput(assetID, changeAmount, ins.Utxo.ControlProgram)
		builder.AddOutput(out)
	}

	tmpl, tx, err := builder.Build()
	if err != nil {
		return NewErrorResponse(err)
	}

	for i, out := range tmpl.Transaction.Outputs {
		if bytes.Equal(out.ControlProgram, ins.Utxo.ControlProgram) {
			//tx.Outputs[i].Amount = changeAmount - uint64(txGasResp.TotalNeu)
			tx.Outputs[i].Amount = changeAmount - 100000000
		}
	}
	tmpl.Transaction = bytomtypes.NewTx(*tx)
	return NewSuccessResponse(tmpl)
}

func (a *API) buildMainChainTx(ins struct {
	Utxo           account.UTXO       `json:"utxo"`
	Tx             types.Tx           `json:"raw_transaction"`
	RootXPubs      []chainkd.XPub     `json:"root_xpubs"`
	ControlProgram string             `json:"control_program"`
	ClaimScript    chainjson.HexBytes `json:"claim_script"`
}) Response {

	var xpubs []chainkd.XPub
	for _, xpub := range ins.RootXPubs {
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(ins.ClaimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		chaildXPub := xpub.Child(tweak)
		xpubs = append(xpubs, chaildXPub)
	}

	txInput, sigInst, err := utxoToInputs(xpubs, &ins.Utxo)
	if err != nil {
		return NewErrorResponse(err)
	}

	builder := mainchain.NewBuilder(time.Now())
	builder.AddInput(txInput, sigInst)
	changeAmount := uint64(0)
	retire := false
	for _, key := range ins.Tx.GetResultIds() {
		output, err := ins.Tx.Retire(*key)
		if err != nil {
			log.WithFields(log.Fields{"moudle": "transact", "err": err}).Warn("buildMainChainTx error")
			continue
		}
		retire = true
		var controlProgram []byte
		retBool := true
		if controlProgram, retBool = getInput(ins.Tx.Entries, *key, ins.ControlProgram); !retBool {
			return NewErrorResponse(errors.New("The corresponding input cannot be found"))
		}

		assetID := bytom.AssetID{
			V0: output.Source.Value.AssetId.GetV0(),
			V1: output.Source.Value.AssetId.GetV1(),
			V2: output.Source.Value.AssetId.GetV2(),
			V3: output.Source.Value.AssetId.GetV3(),
		}
		out := bytomtypes.NewTxOutput(assetID, output.Source.Value.Amount, controlProgram)
		builder.AddOutput(out)
		changeAmount = ins.Utxo.Amount - output.Source.Value.Amount

	}

	if !retire {
		return NewErrorResponse(errors.New("It's not a transaction to retire assets"))
	}

	if changeAmount > 0 {
		u := ins.Utxo
		assetID := bytom.AssetID{
			V0: u.AssetID.GetV0(),
			V1: u.AssetID.GetV1(),
			V2: u.AssetID.GetV2(),
			V3: u.AssetID.GetV3(),
		}
		out := bytomtypes.NewTxOutput(assetID, changeAmount, ins.Utxo.ControlProgram)
		builder.AddOutput(out)
	}

	tmpl, tx, err := builder.Build()
	if err != nil {
		return NewErrorResponse(err)
	}
	//交易费估算
	txGasResp, err := EstimateTxGasForMainchain(*tmpl)
	if err != nil {
		return NewErrorResponse(err)
	}
	for i, out := range tmpl.Transaction.Outputs {
		if bytes.Equal(out.ControlProgram, ins.Utxo.ControlProgram) {
			tx.Outputs[i].Amount = changeAmount - uint64(txGasResp.TotalNeu)
		}
	}
	tmpl.Transaction = bytomtypes.NewTx(*tx)
	return NewSuccessResponse(tmpl)
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
	sourceID := bytom.Hash{
		V0: u.SourceID.GetV0(),
		V1: u.SourceID.GetV1(),
		V2: u.SourceID.GetV2(),
		V3: u.SourceID.GetV3(),
	}

	assetID := bytom.AssetID{
		V0: u.AssetID.GetV0(),
		V1: u.AssetID.GetV1(),
		V2: u.AssetID.GetV2(),
		V3: u.AssetID.GetV3(),
	}

	txInput := bytomtypes.NewSpendInput(nil, sourceID, assetID, u.Amount, u.SourcePos, u.ControlProgram)
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

func contractToInputs(a *API, u *account.UTXO, xpubs []chainkd.XPub) (*bytomtypes.TxInput, *mainchain.SigningInstruction, error) {
	sourceID := bytom.Hash{
		V0: u.SourceID.GetV0(),
		V1: u.SourceID.GetV1(),
		V2: u.SourceID.GetV2(),
		V3: u.SourceID.GetV3(),
	}

	assetID := bytom.AssetID{
		V0: u.AssetID.GetV0(),
		V1: u.AssetID.GetV1(),
		V2: u.AssetID.GetV2(),
		V3: u.AssetID.GetV3(),
	}

	txInput := bytomtypes.NewSpendInput(nil, sourceID, assetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &mainchain.SigningInstruction{}
	for _, xpub := range xpubs {
		xpubsTmp := []chainkd.XPub{xpub}
		sigInst.AddRawWitnessKeysWithoutPath(xpubsTmp, 1)
	}
	return txInput, sigInst, nil
}

type signRespForMainchain struct {
	Tx           *mainchain.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (a *API) signWithKey(ins struct {
	Xprv        string             `json:"xprv"`
	XPub        chainkd.XPub       `json:"xpub"`
	Txs         mainchain.Template `json:"transaction"`
	ClaimScript chainjson.HexBytes `json:"claim_script"`
}) Response {
	xprv := &chainkd.XPrv{}
	if err := xprv.UnmarshalText([]byte(ins.Xprv)); err != nil {
		return NewErrorResponse(err)
	}
	// pub + scriptPubKey 生成一个随机数A
	var tmp [32]byte
	h := hmac.New(sha256.New, ins.XPub[:])
	h.Write(ins.ClaimScript)
	tweak := h.Sum(tmp[:])
	// pub +  A 生成一个新的公钥pub_new
	privateKey := xprv.Child(tweak, false)

	if err := sign(&ins.Txs, privateKey); err != nil {
		return NewErrorResponse(err)
	}
	log.Info("Sign Transaction complete.")
	log.Info(mainchain.SignProgress(&ins.Txs))
	return NewSuccessResponse(&signRespForMainchain{Tx: &ins.Txs, SignComplete: mainchain.SignProgress(&ins.Txs)})
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
