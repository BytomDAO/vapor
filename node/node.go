package node

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/prometheus/util/flock"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"
	browser "github.com/toqueteos/webbrowser"

	"github.com/vapor/accesstoken"
	"github.com/vapor/account"
	"github.com/vapor/api"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/blockchain/txfeed"
	"github.com/vapor/common"
	cfg "github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/database"
	dbm "github.com/vapor/database/db"
	_ "github.com/vapor/database/leveldb"
	"github.com/vapor/database/orm"
	_ "github.com/vapor/database/sqlite"
	"github.com/vapor/env"
	"github.com/vapor/mining/miner"
	"github.com/vapor/net/websocket"
	"github.com/vapor/netsync"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/util"
	w "github.com/vapor/wallet"
)

const (
	webHost           = "http://127.0.0.1"
	maxNewBlockChSize = 1024
)

type Node struct {
	cmn.BaseService

	db dbm.SQLDB
	// config
	config *cfg.Config

	syncManager *netsync.SyncManager

	//bcReactor    *bc.BlockchainReactor
	wallet          *w.Wallet
	accessTokens    *accesstoken.CredentialStore
	notificationMgr *websocket.WSNotificationManager
	api             *api.API
	chain           *protocol.Chain
	txfeed          *txfeed.Tracker
	//cpuMiner        *cpuminer.CPUMiner
	miner *miner.Miner

	miningEnable bool

	newBlockCh chan *bc.Hash
}

func NewNode(config *cfg.Config) *Node {
	ctx := context.Background()
	if err := lockDataDirectory(config); err != nil {
		cmn.Exit("Error: " + err.Error())
	}
	initLogFile(config)
	initActiveNetParams(config)
	initConsensusConfig(config)
	initCommonConfig(config)

	util.MainchainConfig = config.MainChain
	util.ValidatePegin = config.ValidatePegin
	// Get store
	if config.DBBackend != "memdb" && config.DBBackend != "leveldb" {
		cmn.Exit(cmn.Fmt("Param db_backend [%v] is invalid, use leveldb or memdb", config.DBBackend))
	}

	sqlDB := dbm.NewSqlDB("sql", "sqlitedb", config.DBDir())
	initDatabaseTable(sqlDB)
	sqlStore := database.NewSQLStore(sqlDB)

	accessTokens := accesstoken.NewStore(sqlDB)

	txPool := protocol.NewTxPool(sqlStore)
	chain, err := protocol.NewChain(sqlStore, txPool)
	if err != nil {
		cmn.Exit(cmn.Fmt("Failed to create chain structure: %v", err))
	}

	switch config.Consensus.Type {
	case "dpos":
		initDpos(chain, config)
	}

	var accounts *account.Manager = nil
	var assets *asset.Registry = nil
	var wallet *w.Wallet = nil
	var txFeed *txfeed.Tracker = nil

	txFeedDB := dbm.NewDB("txfeeds", config.DBBackend, config.DBDir())
	txFeed = txfeed.NewTracker(txFeedDB, chain)

	if err = txFeed.Prepare(ctx); err != nil {
		log.WithField("error", err).Error("start txfeed")
		return nil
	}

	hsm, err := pseudohsm.New(config.KeysDir())
	if err != nil {
		cmn.Exit(cmn.Fmt("initialize HSM failed: %v", err))
	}

	if !config.Wallet.Disable {
		address, err := common.DecodeAddress(config.Consensus.Coinbase, &consensus.ActiveNetParams)
		if err != nil {
			cmn.Exit(cmn.Fmt("DecodeAddress: %v", err))
		}
		walletDB := dbm.NewDB("wallet", config.DBBackend, config.DBDir())
		accounts = account.NewManager(walletDB, chain)
		assets = asset.NewRegistry(walletDB, chain)
		wallet, err = w.NewWallet(walletDB, accounts, assets, hsm, chain, address)
		if err != nil {
			log.WithField("error", err).Error("init NewWallet")
		}

		// trigger rescan wallet
		if config.Wallet.Rescan {
			wallet.RescanBlocks()
		}
	}
	newBlockCh := make(chan *bc.Hash, maxNewBlockChSize)

	syncManager, _ := netsync.NewSyncManager(config, chain, txPool, newBlockCh)

	notificationMgr := websocket.NewWsNotificationManager(config.Websocket.MaxNumWebsockets, config.Websocket.MaxNumConcurrentReqs, chain)

	// get transaction from txPool and send it to syncManager and wallet
	go newPoolTxListener(txPool, syncManager, wallet, notificationMgr)

	// run the profile server
	profileHost := config.ProfListenAddress
	if profileHost != "" {
		// Profiling bytomd programs.see (https://blog.golang.org/profiling-go-programs)
		// go tool pprof http://profileHose/debug/pprof/heap
		go func() {
			if err = http.ListenAndServe(profileHost, nil); err != nil {
				cmn.Exit(cmn.Fmt("Failed to register tcp profileHost: %v", err))
			}
		}()
	}
	node := &Node{
		db:           sqlDB,
		config:       config,
		syncManager:  syncManager,
		accessTokens: accessTokens,
		wallet:       wallet,
		chain:        chain,
		txfeed:       txFeed,
		miningEnable: config.Mining,

		newBlockCh:      newBlockCh,
		notificationMgr: notificationMgr,
	}

	node.miner = miner.NewMiner(chain, accounts, txPool, newBlockCh)
	node.BaseService = *cmn.NewBaseService(nil, "Node", node)

	return node
}

// newPoolTxListener listener transaction from txPool, and send it to syncManager and wallet
func newPoolTxListener(txPool *protocol.TxPool, syncManager *netsync.SyncManager, wallet *w.Wallet, notificationMgr *websocket.WSNotificationManager) {
	txMsgCh := txPool.GetMsgCh()
	syncManagerTxCh := syncManager.GetNewTxCh()

	for {
		msg := <-txMsgCh
		switch msg.MsgType {
		case protocol.MsgNewTx:
			syncManagerTxCh <- msg.Tx
			if wallet != nil {
				wallet.AddUnconfirmedTx(msg.TxDesc)
			}
			notificationMgr.NotifyMempoolTx(msg.Tx)
		case protocol.MsgRemoveTx:
			if wallet != nil {
				wallet.RemoveUnconfirmedTx(msg.TxDesc)
			}
		default:
			log.Warn("got unknow message type from the txPool channel")
		}
	}
}

// Lock data directory after daemonization
func lockDataDirectory(config *cfg.Config) error {
	_, _, err := flock.New(filepath.Join(config.RootDir, "LOCK"))
	if err != nil {
		return errors.New("datadir already used by another process")
	}
	return nil
}

func initActiveNetParams(config *cfg.Config) {
	var exist bool
	consensus.ActiveNetParams, exist = consensus.NetParams[config.ChainID]
	if !exist {
		cmn.Exit(cmn.Fmt("chain_id[%v] don't exist", config.ChainID))
	}
	if config.Side.FedpegXPubs != "" {
		var federationRedeemXPubs []chainkd.XPub
		fedpegXPubs := strings.Split(config.Side.FedpegXPubs, ",")
		for _, xpubStr := range fedpegXPubs {
			var xpub chainkd.XPub
			xpub.UnmarshalText([]byte(xpubStr))
			federationRedeemXPubs = append(federationRedeemXPubs, xpub)
		}
		consensus.ActiveNetParams.FedpegXPubs = federationRedeemXPubs
	}

	consensus.ActiveNetParams.Signer = config.Signer
	consensus.ActiveNetParams.PeginMinDepth = config.Side.PeginMinDepth
	consensus.ActiveNetParams.ParentGenesisBlockHash = config.Side.ParentGenesisBlockHash
}

func initLogFile(config *cfg.Config) {
	if config.LogFile == "" {
		return
	}
	cmn.EnsureDir(filepath.Dir(config.LogFile), 0700)
	file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.WithField("err", err).Info("using default")
	}

}

func initCommonConfig(config *cfg.Config) {
	cfg.CommonConfig = config
}

// Lanch web broser or not
func launchWebBrowser(port string) {
	webAddress := webHost + ":" + port
	log.Info("Launching System Browser with :", webAddress)
	if err := browser.Open(webAddress); err != nil {
		log.Error(err.Error())
		return
	}
}

func (n *Node) initAndstartApiServer() {
	n.api = api.NewAPI(n.syncManager, n.wallet, n.txfeed, n.miner, n.chain, n.config, n.accessTokens, n.newBlockCh, n.notificationMgr)

	listenAddr := env.String("LISTEN", n.config.ApiAddress)
	env.Parse()
	n.api.StartServer(*listenAddr)
}

func (n *Node) OnStart() error {
	if n.miningEnable {
		if _, err := n.wallet.AccountMgr.GetMiningAddress(); err != nil {
			n.miningEnable = false
			log.Error(err)
		} else {
			//n.cpuMiner.Start()
			n.miner.Start()
		}
	}
	if !n.config.VaultMode {
		n.syncManager.Start()
	}
	n.initAndstartApiServer()
	n.notificationMgr.Start()
	if !n.config.Web.Closed {
		_, port, err := net.SplitHostPort(n.config.ApiAddress)
		if err != nil {
			log.Error("Invalid api address")
			return err
		}
		launchWebBrowser(port)
	}
	go bytomdRPCCheck()
	return nil
}

func (n *Node) OnStop() {
	if err := n.chain.Engine.Finish(); err != nil {
		log.Errorf("OnStop: %v", err)
	}

	n.notificationMgr.Shutdown()
	n.notificationMgr.WaitForShutdown()
	n.BaseService.OnStop()
	if n.miningEnable {
		n.miner.Stop()
	}
	if !n.config.VaultMode {
		n.syncManager.Stop()
	}

	n.db.Db().Close()
}

func (n *Node) RunForever() {
	// Sleep forever and then...
	cmn.TrapSignal(func() {
		n.Stop()
	})
}

func (n *Node) SyncManager() *netsync.SyncManager {
	return n.syncManager
}

/**bytomdRPCCheck Check if bytomd connection via RPC is correctly working*/
func bytomdRPCCheck() bool {
	type Req struct {
		BlockHeight uint64 `json:"block_height"`
	}
	if util.ValidatePegin {
		for {
			resp, err := util.CallRPC("/get-merkle-proof", &Req{BlockHeight: 0})
			if err != nil {
				log.Error("Call mainchain interface get-block-header failed")
				time.Sleep(time.Millisecond * 1000)
				continue
			}
			tmp, _ := json.Marshal(resp)
			var blockHeader api.GetBlockHeaderResp
			json.Unmarshal(tmp, &blockHeader)
			hash := blockHeader.BlockHeader.Hash()
			if strings.Compare(consensus.ActiveNetParams.ParentGenesisBlockHash, hash.String()) != 0 {
				log.Error("Invalid parent genesis block hash response via RPC. Contacting wrong parent daemon?", consensus.ActiveNetParams.ParentGenesisBlockHash, ":", hash.String())
				return false
			}
			break
		}
	}

	return true
}

func initConsensusConfig(config *cfg.Config) {
	if config.ConsensusConfigFile == "" {
		// poa
	} else {
		//
		file, err := os.Open(config.ConsensusConfigFile)
		if err != nil {
			cmn.Exit(cmn.Fmt("Failed to read consensus file: %v", err))
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(config); err != nil {
			cmn.Exit(cmn.Fmt("invalid consensus file: %v", err))
		}

		for _, v := range config.Consensus.SelfVoteSigners {
			address, err := common.DecodeAddress(v, &consensus.ActiveNetParams)
			if err != nil {
				cmn.Exit(cmn.Fmt("Address resolution failed: %v", err))
			}
			config.Consensus.Signers = append(config.Consensus.Signers, address)
		}
	}
}

func initDpos(chain *protocol.Chain, config *cfg.Config) {
	header := chain.BestBlockHeader()
	height := header.Height
	hash := header.Hash()
	maxSignerCount := config.Consensus.MaxSignerCount
	period := config.Consensus.Period
	err := chain.Engine.Init(chain, maxSignerCount, period, height, hash)

	if height > 0 {
		oldBlockHeight := chain.Engine.GetOldBlockHeight()
		oldBlockHash := chain.Engine.GetOldBlockHash()
		if err != nil {
			oldBlockHeight = 0
			header, _ = chain.GetHeaderByHeight(oldBlockHeight)
			oldBlockHash = header.Hash()
		}

		if err := chain.RepairDPoSData(oldBlockHeight, oldBlockHash); err != nil {
			cmn.Exit(cmn.Fmt("initVote failed: %v", err))
		}
	}
}

func initDatabaseTable(db dbm.SQLDB) {
	db.Db().AutoMigrate(&orm.BlockHeader{}, &orm.Transaction{}, &orm.BlockStoreState{}, &orm.ClaimTxState{}, &orm.Utxo{})
}
