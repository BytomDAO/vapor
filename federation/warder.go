package federation

import (
	"database/sql"
	"time"

	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

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
	go w.collectPendingTx()

	for ormTx := range w.txCh {
		if err := w.validateCrossTx(ormTx); err != nil {
			log.Warnln("invalid cross-chain tx", ormTx)
			continue
		}

		destTx, destTxID, err := w.proposeDestTx(ormTx)
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
			submittedTxID, err := w.submitTx(destTx)
			if err != nil {
				log.WithFields(log.Fields{
					"err":            err,
					"cross-chain tx": ormTx,
					"dest tx":        destTx,
				}).Warnln("submitTx")
				continue
			}

			if submittedTxID != destTxID {
				log.WithFields(log.Fields{
					"err":            err,
					"cross-chain tx": ormTx,
					"built tx ID":    destTxID,
					"submittedTx ID": submittedTxID,
				}).Warnln("submitTx ID mismatch")
				continue

			}

			// TODO: what to update? what about others?
			if err := w.updateSubmission(ormTx); err != nil {
				log.WithFields(log.Fields{
					"err":            err,
					"cross-chain tx": ormTx,
				}).Warnln("updateSubmission")
				continue
			}
		}
	}
}

func (w *warder) collectPendingTx() {
	ticker := time.NewTicker(w.colletInterval)
	for ; true; <-ticker.C {
		txs := []*orm.CrossTransaction{}
		if err := w.db.Preload("Chain").Preload("Reqs").
			// do not use "Where(&orm.CrossTransaction{Status: common.CrossTxPendingStatus})" directly
			// otherwise the field "status" is ignored
			Model(&orm.CrossTransaction{}).Where("status = ?", common.CrossTxPendingStatus).
			Find(&txs).Error; err == gorm.ErrRecordNotFound {
			continue
		} else if err != nil {
			log.Warnln("collectPendingTx", err)
		}

		for _, tx := range txs {
			w.txCh <- tx
		}
	}
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

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) (interface{}, string, error) {
	switch tx.Chain.Name {
	case "bytom":
		return w.buildSidechainTx(tx)
	case "vapor":
		return w.buildMainchainTx(tx)
	default:
		return nil, "", errors.New("unknown source chain")
	}
}

// TODO: build it
func (w *warder) buildSidechainTx(tx *orm.CrossTransaction) (interface{}, string, error) {
	sidechainTx := &vaporTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{sidechainTx.ID.String(), true},
	}).Error; err != nil {
		return nil, "", err
	}

	return sidechainTx, sidechainTx.ID.String(), nil
}

// TODO: build it
func (w *warder) buildMainchainTx(tx *orm.CrossTransaction) (interface{}, string, error) {
	mainchainTx := &btmTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{mainchainTx.ID.String(), true},
	}).Error; err != nil {
		return nil, "", err
	}

	return mainchainTx, mainchainTx.ID.String(), nil
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
func (w *warder) submitTx(destTx interface{}) (string, error) {
	switch tx := destTx.(type) {
	case *btmTypes.Tx:
		return w.mainchainNode.SubmitTx(tx /*, true*/)

	case *vaporTypes.Tx:
		return w.sidechainNode.SubmitTx(tx /*, false*/)

	default:
		return "", errors.New("unknown destTx type")
	}
}

// TODO:
func (w *warder) updateSubmission(tx *orm.CrossTransaction) error {
	return nil
}
