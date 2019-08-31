package monitor

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/toolbar/precog/common"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

// create or update: https://github.com/jinzhu/gorm/issues/1307
func (m *monitor) upsertNode(node *config.Node) error {
	if node.XPub != nil {
		node.PublicKey = fmt.Sprintf("%v", node.XPub.PublicKey().String())
	}

	ormNode := &orm.Node{PublicKey: node.PublicKey}
	if err := m.db.Where(&orm.Node{PublicKey: node.PublicKey}).First(ormNode).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if node.XPub != nil {
		ormNode.Xpub = node.XPub.String()
	}
	ormNode.IP = node.IP
	ormNode.Port = node.Port
	return m.db.Where(&orm.Node{PublicKey: ormNode.PublicKey}).
		Assign(&orm.Node{
			Xpub: ormNode.Xpub,
			IP:   ormNode.IP,
			Port: ormNode.Port,
		}).FirstOrCreate(ormNode).Error
}

// TODO: maybe return connected nodes here for checkStatus
func (m *monitor) processDialResults() error {
	var ormNodes []*orm.Node
	if err := m.db.Model(&orm.Node{}).Find(&ormNodes).Error; err != nil {
		return err
	}

	publicKeyMap := make(map[string]*orm.Node, len(ormNodes))
	for _, ormNode := range ormNodes {
		publicKeyMap[ormNode.PublicKey] = ormNode
	}

	connMap := make(map[string]bool, len(ormNodes))
	// connected peers
	for _, peer := range m.sw.GetPeers().List() {
		xPub := &chainkd.XPub{}
		if err := xPub.UnmarshalText([]byte(peer.Key)); err != nil {
			log.WithFields(log.Fields{"xpub": peer.Key}).Error("unmarshal xpub")
			continue
		}

		publicKey := xPub.PublicKey().String()
		connMap[publicKey] = true
		if err := m.processConnectedPeer(publicKeyMap[publicKey]); err != nil {
			log.WithFields(log.Fields{"peer publicKey": publicKey, "err": err}).Error("processConnectedPeer")
		}
	}

	// offline peers
	for _, ormNode := range ormNodes {
		if _, ok := connMap[ormNode.PublicKey]; ok {
			continue
		}

		if err := m.processOfflinePeer(ormNode); err != nil {
			log.WithFields(log.Fields{"peer publicKey": ormNode.PublicKey, "err": err}).Error("processOfflinePeer")
		}
	}

	return nil
}

func (m *monitor) processConnectedPeer(ormNode *orm.Node) error {
	ormNodeLiveness := &orm.NodeLiveness{NodeID: ormNode.ID}
	err := m.db.Preload("Node").Where(ormNodeLiveness).Last(ormNodeLiveness).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	ormNodeLiveness.PongTimes += 1
	if ormNode.Status == common.NodeOfflineStatus {
		ormNode.Status = common.NodeUnknownStatus
	}
	ormNodeLiveness.Node = ormNode
	return m.db.Save(ormNodeLiveness).Error
}

func (m *monitor) processOfflinePeer(ormNode *orm.Node) error {
	ormNode.Status = common.NodeOfflineStatus
	return m.db.Save(ormNode).Error
}

func (m *monitor) processPeerInfos(peerInfos []*peers.PeerInfo) {
	for _, peerInfo := range peerInfos {
		dbTx := m.db.Begin()
		if err := m.processPeerInfo(dbTx, peerInfo); err != nil {
			log.WithFields(log.Fields{"peerInfo": peerInfo, "err": err}).Error("processPeerInfo")
			dbTx.Rollback()
		} else {
			dbTx.Commit()
		}
	}
}

func (m *monitor) processPeerInfo(dbTx *gorm.DB, peerInfo *peers.PeerInfo) error {
	xPub := &chainkd.XPub{}
	if err := xPub.UnmarshalText([]byte(peerInfo.ID)); err != nil {
		return err
	}

	ormNode := &orm.Node{}
	if err := dbTx.Model(&orm.Node{}).Where(&orm.Node{PublicKey: xPub.PublicKey().String()}).First(ormNode).Error; err != nil {
		return err
	}

	if ormNode.Status == common.NodeOfflineStatus {
		return fmt.Errorf("node %s status error", ormNode.PublicKey)
	}

	log.WithFields(log.Fields{"ping": peerInfo.Ping}).Debug("peerInfo")
	ping, err := time.ParseDuration(peerInfo.Ping)
	if err != nil {
		return err
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	var ormNodeLivenesses []*orm.NodeLiveness
	if err := dbTx.Preload("Node").Model(&orm.NodeLiveness{}).
		Where("node_id = ? AND updated_at >= ?", ormNode.ID, yesterday).
		Order(fmt.Sprintf("created_at %s", "DESC")).
		Find(&ormNodeLivenesses).Error; err != nil {
		return err
	}

	// update latest liveness
	latestLiveness := ormNodeLivenesses[0]
	lantencyMS := ping.Nanoseconds() / 1000
	if lantencyMS != 0 {
		latestLiveness.AvgLantencyMS = sql.NullInt64{
			Int64: (latestLiveness.AvgLantencyMS.Int64*int64(latestLiveness.PongTimes) + lantencyMS) / int64(latestLiveness.PongTimes+1),
			Valid: true,
		}
	}
	latestLiveness.PongTimes += 1
	if peerInfo.Height != 0 {
		latestLiveness.BestHeight = peerInfo.Height
	}
	if err := dbTx.Save(latestLiveness).Error; err != nil {
		return err
	}

	// calc LatestDailyUptimeMinutes
	total := 0 * time.Minute
	ormNodeLivenesses[0].UpdatedAt = now
	for _, ormNodeLiveness := range ormNodeLivenesses {
		if ormNodeLiveness.CreatedAt.Before(yesterday) {
			ormNodeLiveness.CreatedAt = yesterday
		}

		total += ormNodeLiveness.UpdatedAt.Sub(ormNodeLiveness.CreatedAt)
	}

	return dbTx.Model(&orm.Node{}).Where(&orm.Node{PublicKey: xPub.PublicKey().String()}).
		UpdateColumn(&orm.Node{
			Alias:                    peerInfo.Moniker,
			Xpub:                     peerInfo.ID,
			BestHeight:               peerInfo.Height,
			LatestDailyUptimeMinutes: uint64(total.Minutes()),
		}).First(ormNode).Error
}
