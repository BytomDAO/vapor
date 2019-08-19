package monitor

import (
	"database/sql"
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/toolbar/precog/common"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

// TODO: get lantency
// TODO: get best_height
// TODO: decide check_height("best best_height" - "confirmations")
// TODO: get blockhash by check_height, get latency
// TODO: update lantency, active_time and status

// create or update: https://github.com/jinzhu/gorm/issues/1307
func (m *monitor) upSertNode(dbTx *gorm.DB, node *config.Node) error {
	if node.XPub != nil {
		node.PublicKey = fmt.Sprintf("%v", node.XPub.PublicKey().String())
	}

	ormNode := &orm.Node{PublicKey: node.PublicKey}
	if err := dbTx.Where(&orm.Node{PublicKey: node.PublicKey}).First(ormNode).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if node.Alias != "" {
		ormNode.Alias = node.Alias
	}
	if node.XPub != nil {
		ormNode.Xpub = node.XPub.String()
	}
	ormNode.Host = node.Host
	ormNode.Port = node.Port
	return dbTx.Where(&orm.Node{PublicKey: ormNode.PublicKey}).
		Assign(&orm.Node{
			Xpub:  ormNode.Xpub,
			Alias: ormNode.Alias,
			Host:  ormNode.Host,
			Port:  ormNode.Port,
		}).FirstOrCreate(ormNode).Error
}

func (m *monitor) savePeerInfo(dbTx *gorm.DB, peerInfo *peers.PeerInfo) error {
	xPub := &chainkd.XPub{}
	if err := xPub.UnmarshalText([]byte(peerInfo.ID)); err != nil {
		return err
	}

	ormNode := &orm.Node{}
	if err := dbTx.Model(&orm.Node{}).Where(&orm.Node{PublicKey: xPub.PublicKey().String()}).
		UpdateColumn(&orm.Node{
			Alias:      peerInfo.Moniker,
			Xpub:       peerInfo.ID,
			BestHeight: peerInfo.Height,
			// LatestDailyUptimeMinutes uint64
		}).First(ormNode).Error; err != nil {
		return err
	}

	log.Debug("peerInfo.Ping:", peerInfo.Ping)

	ormNodeLiveness := &orm.NodeLiveness{
		NodeID:        ormNode.ID,
		BestHeight:    ormNode.BestHeight,
		AvgLantencyMS: sql.NullInt64{Int64: 1, Valid: true},
		// PingTimes     uint64
		// PongTimes     uint64
	}
	if err := dbTx.Model(&orm.NodeLiveness{}).Where("node_id = ? AND status != ?", ormNode.ID, common.NodeOfflineStatus).
		UpdateColumn(&orm.NodeLiveness{
			BestHeight:    ormNodeLiveness.BestHeight,
			AvgLantencyMS: ormNodeLiveness.AvgLantencyMS,
		}).FirstOrCreate(ormNodeLiveness).Error; err != nil {
		return err
	}

	return nil
}
