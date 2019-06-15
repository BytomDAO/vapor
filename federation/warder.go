package federation

import (
	"github.com/jinzhu/gorm"

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
		if err := validateTx(tx); err != nil {
			log.Warn("invalid cross-chain tx")
			continue
		}

		w.proposeDestTx(tx)
	}
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) {}

func validateTx(tx *orm.CrossTransaction) error {
	return nil
}
