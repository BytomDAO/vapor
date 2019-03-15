package api

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/account"
	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/config"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/protocol/bc"
)

func (a *API) dpos(ctx context.Context, ins struct {
	To     string `json:"to"`
	Fee    uint64 `json:"fee"`
	Stake  uint64 `json:"stake"`
	TxType uint32 `json:"tx_type"`
}) Response {
	// 找到utxo
	var assetID bc.AssetID
	assetID.UnmarshalText([]byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
	// 生成dpos交易
	dpos := account.DopsAction{
		Accounts: a.wallet.AccountMgr,
		From:     config.CommonConfig.Consensus.Dpos.Coinbase,
		To:       ins.To,
		Fee:      ins.Fee,
		TxType:   ins.TxType,
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
	if err := pseudohsm.SignWithKey(tmpl, xprv); err != nil {
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
}
