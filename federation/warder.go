package federation

import (
	"database/sql"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	btmTypes "github.com/vapor/protocol/bc/types"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/database/orm"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

type warder struct {
	db   *gorm.DB
	txCh chan *orm.CrossTransaction
}

func NewWarder(db *gorm.DB, txCh chan *orm.CrossTransaction) *warder {
	return &warder{
		txCh: txCh,
	}
}

func (w *warder) Run() {
	for tx := range w.txCh {
		if err := w.validateTx(tx); err != nil {
			log.Warnln("invalid cross-chain tx", tx)
			continue
		}

		if err := w.proposeDestTx(tx); err != nil {
			log.WithFields(log.Fields{
				"err":            err,
				"cross-chain tx": tx,
			}).Warnln("proposeDestTx")
			continue
		}

		if err := w.signDestTx(tx); err != nil {
			log.WithFields(log.Fields{
				"err":            err,
				"cross-chain tx": tx,
			}).Warnln("signDestTx")
			continue
		}
	}
}

func (w *warder) validateTx(tx *orm.CrossTransaction) error {
	if tx.Status != common.CrossTxPendingStatus {
		return errors.New("cross-chain tx already proposed")
	}

	crossTxReqs := []*orm.CrossTransactionReq{}
	if err := w.db.Where(&orm.CrossTransactionReq{CrossTransactionID: tx.ID}).Find(&crossTxReqs).Error; err != nil {
		return err
	}

	if len(crossTxReqs) != len(tx.Reqs) {
		return errors.New("cross-chain requests mismatch")
	}

	return nil
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) error {
	switch tx.Chain.Name {
	case "bytom":
		return w.buildSidechainTx(tx)
	case "vapor":
		return w.buildMainchainTx(tx)
	default:
		return errors.New("unknown source chain")
	}
}

func (w *warder) buildSidechainTx(tx *orm.CrossTransaction) error {
	sidechainTx := &vaporTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{sidechainTx.ID.String(), true},
	}).Error; err != nil {
		return err
	}

	return nil
}

func (w *warder) buildMainchainTx(tx *orm.CrossTransaction) error {
	mainchainTx := &btmTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{mainchainTx.ID.String(), true},
	}).Error; err != nil {
		return err
	}

	return nil
}

func (w *warder) signDestTx(tx *orm.CrossTransaction) error {
	return nil
}
