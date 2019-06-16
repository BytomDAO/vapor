package federation

import (
	"database/sql"
	"time"

	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

var collectInterval = 5 * time.Second

type warder struct {
	position       uint8
	xpub           chainkd.XPub
	colletInterval time.Duration
	db             *gorm.DB
	txCh           chan *orm.CrossTransaction
	mainchainNode  *service.Node
	sidechainNode  *service.Node
	remotes        []*service.Warder
}

func NewWarder(cfg *config.Config, db *gorm.DB) *warder {
	local, remotes := parseWarders(cfg)
	return &warder{
		position:      local.Position,
		xpub:          local.XPub,
		db:            db,
		txCh:          make(chan *orm.CrossTransaction),
		mainchainNode: service.NewNode(cfg.Mainchain.Upstream),
		sidechainNode: service.NewNode(cfg.Sidechain.Upstream),
		remotes:       remotes,
	}
}

func parseWarders(cfg *config.Config) (*service.Warder, []*service.Warder) {
	var local *service.Warder
	var remotes []*service.Warder
	for _, warderCfg := range cfg.Warders {
		if warderCfg.IsLocal {
			local = service.NewWarder(&warderCfg)
		} else {
			remoteWarder := service.NewWarder(&warderCfg)
			remotes = append(remotes, remoteWarder)
		}
	}

	if local == nil {
		log.Fatal("none local warder set")
	}

	return local, remotes
}

func (w *warder) Run() {
	go w.collectPendingTx()
	go w.processCrossTxRoutine()
}

func (w *warder) collectPendingTx() {
	ticker := time.NewTicker(collectInterval)
	for ; true; <-ticker.C {
		txs := []*orm.CrossTransaction{}
		if err := w.db.Preload("Chain").Preload("Reqs").
			// do not use "Where(&orm.CrossTransaction{Status: common.CrossTxPendingStatus})" directly,
			// otherwise the field "status" will be ignored
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

func (w *warder) processCrossTxRoutine() {
	for ormTx := range w.txCh {
		if err := w.validateCrossTx(ormTx); err != nil {
			log.Warnln("invalid cross-chain tx", ormTx)
			continue
		}

		destTx, destTxID, err := w.proposeDestTx(ormTx)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("proposeDestTx")
			continue
		}

		if err := w.signDestTx(destTx, ormTx); err != nil {
			log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("signDestTx")
			continue
		}

		for _, remote := range w.remotes {
			signs, err := remote.RequestSign(destTx, ormTx)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "remote": remote, "cross-chain tx": ormTx}).Warnln("RequestSign")
				continue
			}

			w.attachSignsForTx(destTx, ormTx, remote.Position, signs)
		}

		if w.isTxSignsReachQuorum(destTx) && w.isLeader() {
			submittedTxID, err := w.submitTx(destTx)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx, "dest tx": destTx}).Warnln("submitTx")
				continue
			}

			if submittedTxID != destTxID {
				log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx, "builtTx ID": destTxID, "submittedTx ID": submittedTxID}).Warnln("submitTx ID mismatch")
				continue
			}

			if err := w.updateSubmission(ormTx); err != nil {
				log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("updateSubmission")
				continue
			}
		}
	}
}

func (w *warder) validateCrossTx(tx *orm.CrossTransaction) error {
	switch tx.Status {
	case common.CrossTxRejectedStatus:
		return errors.New("cross-chain tx rejected")
	case common.CrossTxSubmittedStatus:
		return errors.New("cross-chain tx submitted")
	case common.CrossTxCompletedStatus:
		return errors.New("cross-chain tx completed")
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

// TODO:
func (w *warder) buildSidechainTx(tx *orm.CrossTransaction) (*vaporTypes.Tx, string, error) {
	sidechainTx := &vaporTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{sidechainTx.ID.String(), true},
	}).Error; err != nil {
		return nil, "", err
	}

	return sidechainTx, sidechainTx.ID.String(), nil
}

// TODO:
func (w *warder) buildMainchainTx(tx *orm.CrossTransaction) (*btmTypes.Tx, string, error) {
	mainchainTx := &btmTypes.Tx{}

	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		DestTxHash: sql.NullString{mainchainTx.ID.String(), true},
	}).Error; err != nil {
		return nil, "", err
	}

	return mainchainTx, mainchainTx.ID.String(), nil
}

// TODO:
func (w *warder) signDestTx(destTx interface{}, tx *orm.CrossTransaction) error {
	if tx.Status != common.CrossTxPendingStatus || !tx.DestTxHash.Valid {
		return errors.New("cross-chain tx status error")
	}

	return nil
}

// TODO:
func (w *warder) attachSignsForTx(destTx interface{}, ormTx *orm.CrossTransaction, position uint8, signs string) {
}

// TODO:
func (w *warder) isTxSignsReachQuorum(destTx interface{}) bool {
	return false
}

// TODO:
func (w *warder) isLeader() bool {
	return false
}

func (w *warder) submitTx(destTx interface{}) (string, error) {
	switch tx := destTx.(type) {
	case *btmTypes.Tx:
		return w.mainchainNode.SubmitTx(tx)
	case *vaporTypes.Tx:
		return w.sidechainNode.SubmitTx(tx)
	default:
		return "", errors.New("unknown destTx type")
	}
}

func (w *warder) updateSubmission(tx *orm.CrossTransaction) error {
	if err := w.db.Where(tx).UpdateColumn(&orm.CrossTransaction{
		Status: common.CrossTxSubmittedStatus,
	}).Error; err != nil {
		return err
	}

	for _, remote := range w.remotes {
		remote.NotifySubmission(tx)
	}
	return nil
}
