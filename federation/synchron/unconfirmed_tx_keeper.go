package synchron

import (
	// "time"
	"encoding/json"

	// "github.com/bytom/errors"
	// TODO:
	btmBc "github.com/bytom/protocol/bc"
	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/federation/config"
	// "github.com/blockcenter/database"
	// "github.com/blockcenter/database/orm"
	"github.com/vapor/federation/service"
)

const maxRawTxSize = 1 << 16

type unconfirmedTxKeeper struct {
	cfg         *config.Chain
	db          *gorm.DB
	processTxCh chan *service.WSResponse
	// coinName    string
}

func (u *unconfirmedTxKeeper) Run() {
	ws := service.NewWSClient(u.cfg.Upstream.WebSocket, u.processTxCh)
	if err := ws.Connect(); err != nil {
		log.WithField("err", err).Fatal("websocket dail fail")
	}

	defer ws.Close()
	if err := ws.Subscribe(service.TopicNotifyNewTransactions); err != nil {
		log.WithField("err", err).Fatal("subscribe new transaction fail")
	}

	u.receiveTransactions()
}

// TODO: FK
type TxDesc struct {
	// TODO:
	Tx         *btmTypes.Tx `json:"transaction"`
	StatusFail bool         `json:"status_fail"`
}

func (u *unconfirmedTxKeeper) receiveTransactions() {
	for resp := range u.processTxCh {
		if resp.NotificationType != service.ResponseNewTransaction {
			log.Warn("receive non new transaction message")
			continue
		}

		txDesc := &TxDesc{}
		if err := json.Unmarshal([]byte(resp.Data), txDesc); err != nil {
			log.WithField("err", err).Error("unmarshal transaction error")
			continue
		}

		// coin := &orm.Coin{Name: u.coinName}
		// if err := u.db.Where(coin).First(coin).Error; err != nil {
		// 	log.WithField("err", err).Error("query coin fail")
		// 	continue
		// }

		// TODO: may still need it
		// if err := addIssueAssets(u.db, []*btmTypes.Tx{txDesc.Tx}, coin.ID); err != nil {
		// 	log.WithField("err", err).Error("fail on adding issue assets")
		// }

		if err := u.AddUnconfirmedTx( /*coin,*/ txDesc); err != nil {
			log.WithField("err", err).Error("fail on adding unconfirmed transaction")
		}
	}
}

func (u *unconfirmedTxKeeper) AddUnconfirmedTx( /*coin *orm.Coin, */ txDesc *TxDesc) error {
	dbTx := u.db.Begin()
	// TODO:
	txStatus := &btmBc.TransactionStatus{VerifyStatus: []*btmBc.TxVerifyResult{&btmBc.TxVerifyResult{StatusFail: txDesc.StatusFail}}}
	bp := &attachBlockProcessor{
		db:       dbTx,
		txStatus: txStatus,
		// coin:     coin,
		block: &btmTypes.Block{BlockHeader: btmTypes.BlockHeader{}},
	}

	txs := []*btmTypes.Tx{txDesc.Tx}
	if err := bp.processIssuing(dbTx, txs); err != nil {
		dbTx.Rollback()
		return err
	}

	// mappings, err := GetAddressTxMappings(u.cfg, txs, txStatus, dbTx)
	// if err != nil {
	// 	dbTx.Rollback()
	// 	return err
	// }

	// if err := bp.processAddressTransaction(mappings); err != nil {
	// 	dbTx.Rollback()
	// 	return err
	// }

	return dbTx.Commit().Error
}
