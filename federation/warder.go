package federation

import (
	"time"

	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/errors"
	"github.com/vapor/federation/api"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/federation/util"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

var collectInterval = 5 * time.Second

type warder struct {
	cfg     *config.Config
	db      *gorm.DB
	fedProg []byte
	quorum  int
	// position      uint8
	// xpub          chainkd.XPub
	mainchainNode *service.Node
	sidechainNode *service.Node
	// remotes       []*service.Warder
	server *api.Server
}

func NewWarder(db *gorm.DB, cfg *config.Config) *warder {
	// local, remotes := parseWarders(cfg)
	return &warder{
		cfg:     cfg,
		db:      db,
		fedProg: util.ParseFedProg(cfg.Warders, cfg.Quorum),
		quorum:  cfg.Quorum,
		// position:      local.Position,
		// xpub:          local.XPub,
		mainchainNode: service.NewNode(cfg.Mainchain.Upstream),
		sidechainNode: service.NewNode(cfg.Sidechain.Upstream),
		// remotes:       remotes,
		server: api.NewServer(db, cfg),
	}
}

// func parseWarders(cfg *config.Config) (*service.Warder, []*service.Warder) {
// 	var local *service.Warder
// 	var remotes []*service.Warder
// 	for _, warderCfg := range cfg.Warders {
// 		if warderCfg.IsLocal {
// 			local = service.NewWarder(&warderCfg)
// 		} else {
// 			remote := service.NewWarder(&warderCfg)
// 			remotes = append(remotes, remote)
// 		}
// 	}

// 	if local == nil {
// 		log.Fatal("none local warder set")
// 	}

// 	return local, remotes
// }

func (w *warder) Run() {
	go w.server.Run()

	ticker := time.NewTicker(collectInterval)
	for ; true; <-ticker.C {
		txs := []*orm.CrossTransaction{}
		if err := w.db.Preload("Chain").Preload("Reqs").
			// do not use "Where(&orm.CrossTransaction{Status: common.CrossTxInitiatedStatus})" directly,
			// otherwise the field "status" will be ignored
			Model(&orm.CrossTransaction{}).Where("status = ?", common.CrossTxInitiatedStatus).
			Find(&txs).Error; err == gorm.ErrRecordNotFound {
			continue
		} else if err != nil {
			log.Warnln("collectPendingTx", err)
		}

		for _, tx := range txs {
			go w.tryProcessCrossTx(tx)
		}
	}
}

func (w *warder) tryProcessCrossTx(ormTx *orm.CrossTransaction) error {
	dbTx := w.db.Begin()
	if err := w.processCrossTx(ormTx); err != nil {
		dbTx.Rollback()
		return err
	}

	return dbTx.Commit().Error
}

func (w *warder) processCrossTx(ormTx *orm.CrossTransaction) error {
	if err := w.validateCrossTx(ormTx); err != nil {
		log.Warnln("invalid cross-chain tx", ormTx)
		return err
	}

	destTx, _, err := w.proposeDestTx(ormTx)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("proposeDestTx")
		return err
	}

	if err := w.initDestTxSigns(destTx, ormTx); err != nil {
		log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("initDestTxSigns")
		return err
	}

	return nil
}

func (w *warder) validateCrossTx(tx *orm.CrossTransaction) error {
	switch tx.Status {
	case common.CrossTxRejectedStatus:
		return errors.New("cross-chain tx rejected")
	case common.CrossTxSubmittedStatus:
		return errors.New("cross-chain tx submitted")
	case common.CrossTxCompletedStatus:
		return errors.New("cross-chain tx completed")
	default:
		return nil
	}
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) (interface{}, string, error) {
	switch tx.Chain.Name {
	case common.MainchainNameLabel:
		return w.buildSidechainTx(tx)
	case common.SidechainNameLabel:
		return w.buildMainchainTx(tx)
	default:
		return nil, "", errors.New("unknown source chain")
	}
}

// call vapord api to build tx
func (w *warder) buildSidechainTx(ormTx *orm.CrossTransaction) (*vaporTypes.Tx, string, error) {
	return nil, "", errors.New("buildSidechainTx not implemented yet")
}

// call bytomd api to build tx
func (w *warder) buildMainchainTx(ormTx *orm.CrossTransaction) (*btmTypes.Tx, string, error) {
	return nil, "", errors.New("buildMainchainTx not implemented yet")
}

func (w *warder) initDestTxSigns(destTx interface{}, ormTx *orm.CrossTransaction) error {
	for i := 1; i <= len(w.cfg.Warders); i++ {
		if err := w.db.Create(&orm.CrossTransactionSign{
			CrossTransactionID: ormTx.ID,
			WarderID:           uint8(i),
			Status:             common.CrossTxSignPendingStatus,
		}).Error; err != nil {
			return err
		}
	}

	return w.db.Model(&orm.CrossTransaction{}).
		Where(&orm.CrossTransaction{ID: ormTx.ID}).
		UpdateColumn(&orm.CrossTransaction{
			Status: common.CrossTxPendingStatus,
		}).Error
}
