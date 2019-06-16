package federation

import (
	"database/sql"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	btmTypes "github.com/vapor/protocol/bc/types"

	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

type warder struct {
	colletInterval time.Duration
	db             *gorm.DB
	txCh           chan *orm.CrossTransaction
	mainchainNode  *service.Node
	sidechainNode  *service.Node
}

func NewWarder(cfg *config.Config, db *gorm.DB, txCh chan *orm.CrossTransaction) *warder {
	return &warder{
		colletInterval: time.Duration(cfg.CollectMinutes) * time.Minute,
		db:             db,
		txCh:           txCh,
		mainchainNode:  service.NewNode(cfg.Mainchain.Upstream),
		sidechainNode:  service.NewNode(cfg.Sidechain.Upstream),
	}
}

func (w *warder) Run() {
	go w.collectUnsubmittedTx()

	for ormTx := range w.txCh {
		if err := w.validateCrossTx(ormTx); err != nil {
			log.Warnln("invalid cross-chain tx", ormTx)
			continue
		}

		destTx, err := w.proposeDestTx(ormTx)
		if err != nil {
			log.WithFields(log.Fields{
				"err":            err,
				"cross-chain tx": ormTx,
			}).Warnln("proposeDestTx")
			continue
		}

		if err := w.signDestTx(destTx, ormTx); err != nil {
			log.WithFields(log.Fields{
				"err":            err,
				"cross-chain tx": ormTx,
			}).Warnln("signDestTx")
			continue
		}

		// TODO: elect signer & request sign

		// TODO: what if submit fail
		if w.isTxSignsReachQuorum(destTx) && w.isLeader() {
			if err := w.submitTx(destTx); err != nil {
				log.WithFields(log.Fields{
					"err":            err,
					"cross-chain tx": ormTx,
					"dest tx":        destTx,
				}).Warnln("submitTx")
				continue
			}
		}
	}
}

func (w *warder) collectUnsubmittedTx() {

}

func (w *warder) validateCrossTx(tx *orm.CrossTransaction) error {
	if tx.Status == common.CrossTxRejectedStatus {
		return errors.New("cross-chain tx rejeted")
	}

	if tx.Status == common.CrossTxRejectedStatus {
		return errors.New("cross-chain tx submitted")
	}

	crossTxReqs := []*orm.CrossTransactionReq{}
	if err := w.db.Where(&orm.CrossTransactionReq{CrossTransactionID: tx.ID}).Find(&crossTxReqs).Error; err != nil {
		return err
	}

	if len(crossTxReqs) != len(tx.Reqs) {
		return errors.New("cross-chain requests count mismatch")
	}

	return nil
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) (interface{}, error) {
	switch tx.Chain.Name {
	case "bytom":
		return w.buildSidechainTx(tx)
	case "vapor":
		return w.buildMainchainTx(tx)
	default:
		return nil, errors.New("unknown source chain")
	}
}

// TODO: build it
func (w *warder) buildSidechainTx(tx *orm.CrossTransaction) (interface{}, error) {
	sidechainTx := &vaporTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{sidechainTx.ID.String(), true},
	}).Error; err != nil {
		return nil, err
	}

	return sidechainTx, nil
}

// TODO: build it
func (w *warder) buildMainchainTx(tx *orm.CrossTransaction) (interface{}, error) {
	mainchainTx := &btmTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{mainchainTx.ID.String(), true},
	}).Error; err != nil {
		return nil, err
	}

	return mainchainTx, nil
}

// TODO: sign it
func (w *warder) signDestTx(destTx interface{}, tx *orm.CrossTransaction) error {
	if tx.Status != common.CrossTxPendingStatus || !tx.DestTxHash.Valid {
		return errors.New("cross-chain tx status error")
	}

	return nil
}

// TODO:
func (w *warder) isTxSignsReachQuorum(destTx interface{}) bool {
	return false
}

// TODO:
func (w *warder) isLeader() bool {
	return false
}

// TODO: submit it
func (w *warder) submitTx(destTx interface{}) error {
	return nil
}
