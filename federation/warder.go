package federation

import (
	"github.com/vapor/federation/database/orm"
)

type warder struct {
	txCh chan *orm.CrossTransaction
}

func NewWarder(txCh chan *orm.CrossTransaction) *warder {
	return &warder{
		txCh: txCh,
	}
}

func (w *warder) Run() {
	for tx := range w.txCh {
		w.proposeDestTx(tx)
	}
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) {}
