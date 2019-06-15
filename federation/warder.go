package federation

import (
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/database/orm"
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
				"err":       err,
				"source tx": tx,
			}).Warnln("proposeDestTx")
			continue
		}
	}
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) error {
	return nil
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
