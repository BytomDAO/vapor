package api

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

func (a *API) dpos(ctx context.Context, ins struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Fee    uint64 `json:"fee"`
	Stake  uint64 `json:"stake"`
	TxType uint8  `json:"tx_type"`
}) Response {
	// 找到utxo
	var assetID bc.AssetID
	assetID.UnmarshalText([]byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
	// 生成dpos交易
	dpos := account.DopsAction{
		Accounts: a.wallet.AccountMgr,
		From:     ins.From,
		To:       ins.To,
		Fee:      ins.Fee,
	}
	dpos.Amount = ins.Stake
	dpos.AssetId = &assetID
	builder := txbuilder.NewBuilder(time.Now())
	if err := dpos.Build(ctx, builder); err != nil {
		return NewErrorResponse(err)
	}

	// 签名
	tmpl, _, err := builder.Build()
	if err != nil {
		return NewErrorResponse(err)
	}
	var xprv chainkd.XPrv
	xprv.UnmarshalText([]byte(config.CommonConfig.Consensus.Dpos.XPrv))
	if err := signWithKey(tmpl, xprv); err != nil {
		return NewErrorResponse(err)
	}
	log.Info("Sign Transaction complete.")
	log.Info(txbuilder.SignProgress(tmpl))
	//return NewSuccessResponse(&signTemplateResp{Tx: tmpl, SignComplete: txbuilder.SignProgress(tmpl)})
	// 提交

	if err := txbuilder.FinalizeTx(ctx, a.chain, tmpl.Transaction); err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("tx_id", tmpl.Transaction.ID.String()).Info("submit single tx")
	return NewSuccessResponse(&submitTxResp{TxID: &tmpl.Transaction.ID})

	//return NewSuccessResponse(nil)
}

func signWithKey(tmpl *txbuilder.Template, xprv chainkd.XPrv) error {
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
