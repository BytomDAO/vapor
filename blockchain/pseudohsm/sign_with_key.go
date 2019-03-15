package pseudohsm

import (
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
)

func SignWithKey(tmpl *txbuilder.Template, xprv chainkd.XPrv) error {
	for i, sigInst := range tmpl.SigningInstructions {
		for j, wc := range sigInst.WitnessComponents {
			switch sw := wc.(type) {
			case *txbuilder.SignatureWitness:
				err := sw.Sign(tmpl, uint32(i), xprv)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to signature witness component %d of input %d", j, i)
				}
			case *txbuilder.RawTxSigWitness:
				err := sw.Sign(tmpl, uint32(i), xprv)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to raw-signature witness component %d of input %d", j, i)
				}
			}
		}
	}
	return materializeWitnessesWithKey(tmpl)
}

func materializeWitnessesWithKey(txTemplate *txbuilder.Template) error {
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
