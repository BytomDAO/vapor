package security

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	cfg "github.com/vapor/config"
	dbm "github.com/vapor/database/leveldb"
)

const (
	defaultBanDuration = time.Hour * 1
	blacklistKey       = "BlacklistPeers"
)

var (
	ErrConnectBannedPeer = errors.New("connect banned peer")
)

type Blacklist struct {
	peers map[string]time.Time
	db    dbm.DB

	mtx sync.Mutex
}

func NewBlacklist(config *cfg.Config) *Blacklist {
	return &Blacklist{
		peers: make(map[string]time.Time),
		db:    dbm.NewDB("blacklist", config.DBBackend, config.DBDir()),
	}
}

//addBannedPeer add peer to blacklist
func (bl *Blacklist) addPeer(ip string) error {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	bl.peers[ip] = time.Now().Add(defaultBanDuration)
	dataJSON, err := json.Marshal(bl.peers)
	if err != nil {
		return err
	}

	bl.db.Set([]byte(blacklistKey), dataJSON)
	return nil
}

func (bl *Blacklist) delBannedPeer(ip string) error {
	delete(bl.peers, ip)
	dataJson, err := json.Marshal(bl.peers)
	if err != nil {
		return err
	}

	bl.db.Set([]byte(blacklistKey), dataJson)
	return nil
}

func (bl *Blacklist) DoFilter(ip string, pubKey string) error {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()
	if banEnd, ok := bl.peers[ip]; ok {
		if time.Now().Before(banEnd) {
			return ErrConnectBannedPeer
		}

		if err := bl.delBannedPeer(ip); err != nil {
			return err
		}
	}

	return nil
}

// loadBannedPeers load banned peers from db
func (bl *Blacklist) loadBannedPeers() error {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	if dataJSON := bl.db.Get([]byte(blacklistKey)); dataJSON != nil {
		if err := json.Unmarshal(dataJSON, &bl.peers); err != nil {
			return err
		}
	}

	return nil
}
