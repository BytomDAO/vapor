package monitor

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	"github.com/vapor/toolbar/precog/common"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

// create or update: https://github.com/jinzhu/gorm/issues/1307
func (m *monitor) upSertNode(node *config.Node) error {
	if node.XPub != nil {
		node.PublicKey = fmt.Sprintf("%v", node.XPub.PublicKey().String())
	}

	ormNode := &orm.Node{PublicKey: node.PublicKey}
	if err := m.db.Where(&orm.Node{PublicKey: node.PublicKey}).First(ormNode).Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if node.Alias != "" {
		ormNode.Alias = node.Alias
	}
	if node.XPub != nil {
		ormNode.Xpub = node.XPub.String()
	}
	ormNode.IP = node.IP
	ormNode.Port = node.Port
	return m.db.Where(&orm.Node{PublicKey: ormNode.PublicKey}).
		Assign(&orm.Node{
			Xpub:  ormNode.Xpub,
			Alias: ormNode.Alias,
			IP:    ormNode.IP,
			Port:  ormNode.Port,
		}).FirstOrCreate(ormNode).Error
}

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
			log.Error(err)
			continue
		}

		publicKey := xPub.PublicKey().String()
		connMap[publicKey] = true
		if err := m.processConnectedPeer(publicKeyMap[publicKey], peer); err != nil {
			log.Error(err)
		}
	}

	// offline peers
	for _, ormNode := range ormNodes {
		if _, ok := connMap[ormNode.PublicKey]; ok {
			continue
		}

		if err := m.processOfflinePeer(ormNode); err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (m *monitor) processConnectedPeer(ormNode *orm.Node, peer *p2p.Peer) error {
	ormNodeLiveness := &orm.NodeLiveness{}
	err := m.db.Model(&orm.NodeLiveness{}).Joins("join nodes on nodes.id = node_livenesses.node_id").
		Where("nodes.public_key = ? AND status != ?", ormNode.PublicKey, common.NodeOfflineStatus).Last(ormNodeLiveness).Error
	if err == nil {
		return m.db.Model(&orm.NodeLiveness{}).Where(ormNodeLiveness).UpdateColumn(&orm.NodeLiveness{
			PingTimes: ormNodeLiveness.PingTimes + 1,
		}).Error
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// gorm.ErrRecordNotFound
	return m.db.Create(&orm.NodeLiveness{
		NodeID:    ormNode.ID,
		PingTimes: 1,
		Status:    common.NodeUnknownStatus,
	}).Error
}

func (m *monitor) processOfflinePeer(ormNode *orm.Node) error {
	return m.db.Model(&orm.NodeLiveness{}).
		Where(&orm.NodeLiveness{NodeID: ormNode.ID}).
		UpdateColumn(&orm.NodeLiveness{
			Status: common.NodeOfflineStatus,
		}).Error
}

func (m *monitor) processPeerInfos(peerInfos []*peers.PeerInfo) error {
	for _, peerInfo := range peerInfos {
		dbTx := m.db.Begin()
		if err := m.processPeerInfo(dbTx, peerInfo); err != nil {
			log.Error(err)
			dbTx.Rollback()
		} else {
			dbTx.Commit()
		}
	}

	return nil
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

	log.Debugf("peerInfo ping: %v", peerInfo.Ping)
	ping, err := time.ParseDuration(peerInfo.Ping)
	if err != nil {
		return err
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	var ormNodeLivenesses []*orm.NodeLiveness
	if err := dbTx.Model(&orm.NodeLiveness{}).
		Where("node_id = ? AND updated_at >= ?", ormNode.ID, yesterday).
		Order(fmt.Sprintf("created_at %s", "DESC")).
		Find(&ormNodeLivenesses).Error; err != nil {
		return err
	}

	// update latest liveness
	latestLiveness := ormNodeLivenesses[0]
	if latestLiveness.Status == common.NodeOfflineStatus {
		return fmt.Errorf("node %s latest liveness status error", ormNode.PublicKey)
	}

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
