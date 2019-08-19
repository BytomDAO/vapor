package monitor

import (
	"fmt"

	"github.com/jinzhu/gorm"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/toolbar/precog/config"
	"github.com/vapor/toolbar/precog/database/orm"
)

// TODO: get lantency
// TODO: get best_height
// TODO: decide check_height("best best_height" - "confirmations")
// TODO: get blockhash by check_height, get latency
// TODO: update lantency, active_time and status

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
	ormNode.Host = node.Host
	ormNode.Port = node.Port
	return m.db.Where(&orm.Node{PublicKey: ormNode.PublicKey}).
		Assign(&orm.Node{
			Xpub:  ormNode.Xpub,
			Alias: ormNode.Alias,
			Host:  ormNode.Host,
			Port:  ormNode.Port,
		}).FirstOrCreate(ormNode).Error
}

func (m *monitor) savePeerInfo(peerInfo *peers.PeerInfo) error {
	xPub := &chainkd.XPub{}
	if err := xPub.UnmarshalText([]peerInfo.ID()); err != nil {
		return err
	}

	ormNode := &orm.Node{}
	if err := m.db.Model(&orm.Node{}).Where(&orm.Node{PublicKey: xPub.PublicKey.String()}).
		UpdateColumn(&orm.Node{
			BestHeight: peerInfo.Height,
			// LatestDailyUptimeMinutes uint64
		}).First(ormNode).Error; err != nil {
		return err
	}

	return nil
}
