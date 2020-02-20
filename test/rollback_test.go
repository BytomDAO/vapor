package test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/bytom/vapor/account"
	"github.com/bytom/vapor/application/mov"
	"github.com/bytom/vapor/asset"
	"github.com/bytom/vapor/blockchain/pseudohsm"
	"github.com/bytom/vapor/config"
	cfg "github.com/bytom/vapor/config"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/database"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/event"
	"github.com/bytom/vapor/proposal"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
	w "github.com/bytom/vapor/wallet"
)

const (
	n = 1 // 初始化用的block数量
)

var fedConsensusPath = [][]byte{
	[]byte{0xff, 0xff, 0xff, 0xff},
	[]byte{0xff, 0x00, 0x00, 0x00},
	[]byte{0xff, 0xff, 0xff, 0xff},
	[]byte{0xff, 0x00, 0x00, 0x00},
	[]byte{0xff, 0x00, 0x00, 0x00},
}

type byTime []*protocol.TxDesc

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].Added.Before(a[j].Added) }

func xpub(str string) (xpub chainkd.XPub) {
	if err := xpub.UnmarshalText([]byte(str)); err != nil {
		log.Panicf("Fail converts a string to xpub")
	}
	return xpub
}

func xprv(str string) (xprv chainkd.XPrv) {
	if err := xprv.UnmarshalText([]byte(str)); err != nil {
		log.Panicf("Fail converts a string to xprv")
	}
	return xprv
}

var Xprvs = []chainkd.XPrv{
	xprv("c87f8d0f4bb4b0acbb7f69f1954c4f34d4476e114fffa7b0c853992474a9954a273c2d8f2642a7baf94ebac88f1625af9f5eaf3b13a90de27eec3de78b9fb9ca"),
	xprv("c80fbc34475fc9447753c00820d8448851c87f07e6bdde349260862c9bca5b4bb2e62c15e129067af869ebdf66e5829e61d6f2e47447395cc18c4166b06e8473"),
}

const (
	warnTimeNum       = 2
	warnTimeDenom     = 5
	criticalTimeNum   = 4
	criticalTimeDenom = 5
)

// number 1
// private key: 483355b66c0e15b0913829d709b04557749b871b3bf56ad1de8fda13d3a4954aa2a56121b8eab313b8f36939e8190fe8f267f19496decb91be5644e92b669914
// public key: 32fe453097591f288315ef47b1ebdabf20e8bced8ede670f999980205cacddd4a2a56121b8eab313b8f36939e8190fe8f267f19496decb91be5644e92b669914
// derivied private key: c87f8d0f4bb4b0acbb7f69f1954c4f34d4476e114fffa7b0c853992474a9954a273c2d8f2642a7baf94ebac88f1625af9f5eaf3b13a90de27eec3de78b9fb9ca
// derivied public key: 4d6f710dae8094c111450ca20e054c3aed59dfcb2d29543c29901a5903755e69273c2d8f2642a7baf94ebac88f1625af9f5eaf3b13a90de27eec3de78b9fb9ca

// number 2
// private key: d8e786a4eafa3456e35b2a1467d37dd84f64ba36604f8076015b76a8eec55b4b83d4fac0f94d157cfc720b77602f21b6b8a7e86f95c571e4d7986210dbce44c9
// public key: ebe1060254ec43bd7883e94583ff0a71ef0ec0e1ada4cd0f5ed7e9d37f1d244e83d4fac0f94d157cfc720b77602f21b6b8a7e86f95c571e4d7986210dbce44c9
// derivied private key: c80fbc34475fc9447753c00820d8448851c87f07e6bdde349260862c9bca5b4bb2e62c15e129067af869ebdf66e5829e61d6f2e47447395cc18c4166b06e8473
// derivied public key: 59184c0f1f4f13b8b256ac82df30dc12cfd66b6e09a28054933f848dc51b9a89b2e62c15e129067af869ebdf66e5829e61d6f2e47447395cc18c4166b06e8473

func getKey() {
	xprv, _ := chainkd.NewXPrv(nil)
	fmt.Println("secretKey:", xprv)

	xpub := xprv.XPub()
	fmt.Println("publicKey:", xpub)

	derivateKey := xprv.Derive(fedConsensusPath)
	fmt.Println("derivateSecretKey:", derivateKey)

	derivatePublicKey := derivateKey.XPub()
	fmt.Println("derivatePublicKey", derivatePublicKey)
}

func newFederationConfig() *cfg.FederationConfig {
	return &cfg.FederationConfig{
		Xpubs: []chainkd.XPub{
			xpub("32fe453097591f288315ef47b1ebdabf20e8bced8ede670f999980205cacddd4a2a56121b8eab313b8f36939e8190fe8f267f19496decb91be5644e92b669914"),
			xpub("ebe1060254ec43bd7883e94583ff0a71ef0ec0e1ada4cd0f5ed7e9d37f1d244e83d4fac0f94d157cfc720b77602f21b6b8a7e86f95c571e4d7986210dbce44c9"),
		},
		Quorum: 1,
	}
}

func getBlockerOrder(startTimestamp, blockTimestamp, numOfConsensusNode uint64) uint64 {
	// One round of product block time for all consensus nodes
	roundBlockTime := consensus.ActiveNetParams.BlockNumEachNode * numOfConsensusNode * consensus.ActiveNetParams.BlockTimeInterval
	// The start time of the last round of product block
	lastRoundStartTime := startTimestamp + (blockTimestamp-startTimestamp)/roundBlockTime*roundBlockTime
	// Order of blocker
	return (blockTimestamp - lastRoundStartTime) / (consensus.ActiveNetParams.BlockNumEachNode * consensus.ActiveNetParams.BlockTimeInterval)
}

func getPrevRoundLastBlock(c *protocol.Chain, store protocol.Store, prevBlockHash *bc.Hash) (*types.BlockHeader, error) {
	blockHeader, err := store.GetBlockHeader(prevBlockHash)
	if err != nil {
		return nil, err
	}

	for blockHeader.Height%consensus.ActiveNetParams.RoundVoteBlockNums != 0 {
		blockHeader, err = store.GetBlockHeader(&blockHeader.PreviousBlockHash)
		if err != nil {
			return nil, err
		}
	}
	return blockHeader, nil
}

// according to getOrder
func getXprv(c *protocol.Chain, store protocol.Store, timeStamp uint64) (*chainkd.XPrv, error) {
	prevVoteRoundLastBlock, err := getPrevRoundLastBlock(c, store, c.BestBlockHash())
	if err != nil {
		return &(Xprvs[0]), err
	}

	startTimestamp := prevVoteRoundLastBlock.Timestamp + consensus.ActiveNetParams.BlockTimeInterval
	order := getBlockerOrder(startTimestamp, timeStamp, uint64(len(Xprvs)))
	if order >= uint64(len(Xprvs)) {
		return nil, errors.New("bad order")
	}
	return &(Xprvs[order]), nil
}

func getConsensusResult(c *protocol.Chain, store *database.Store, seq uint64, blockHeader *types.BlockHeader) (*state.ConsensusResult, error) {
	consensusResult, err := store.GetConsensusResult(seq)
	if err != nil {
		return nil, err
	}

	return consensusResult, nil
}

func TestRollback(t *testing.T) {
	// genesisBlock := config.GenesisBlock()

	db := dbm.NewDB("block_test_db", "leveldb", "block_test_db")
	defer os.RemoveAll("block_test_db")

	cfg.CommonConfig = cfg.DefaultConfig()
	cfg.CommonConfig.Federation = newFederationConfig()

	xp := xprv("c87f8d0f4bb4b0acbb7f69f1954c4f34d4476e114fffa7b0c853992474a9954a273c2d8f2642a7baf94ebac88f1625af9f5eaf3b13a90de27eec3de78b9fb9ca")
	cfg.CommonConfig.XPrv = &xp
	consensus.ActiveNetParams.RoundVoteBlockNums = 3

	store := database.NewStore(db)
	dispatcher := event.NewDispatcher()

	movCore := mov.NewMovCore(cfg.CommonConfig.DBBackend, cfg.CommonConfig.DBDir(), consensus.ActiveNetParams.MovStartHeight)
	txPool := protocol.NewTxPool(store, []protocol.DustFilterer{movCore}, dispatcher)
	chain, err := protocol.NewChain(store, txPool, []protocol.Protocoler{movCore}, dispatcher)

	hsm, err := pseudohsm.New(cfg.CommonConfig.KeysDir())
	walletDB := dbm.NewDB("wallet", cfg.CommonConfig.DBBackend, cfg.CommonConfig.DBDir())
	walletStore := database.NewWalletStore(walletDB)
	accountStore := database.NewAccountStore(walletDB)
	accounts := account.NewManager(accountStore, chain)
	assets := asset.NewRegistry(walletDB, chain)
	wallet, err := w.NewWallet(walletStore, accounts, assets, hsm, chain, dispatcher, cfg.CommonConfig.Wallet.TxIndex)
	if err != nil {
		t.Fatal("init NewWallet")
	}

	// trigger rescan wallet
	if cfg.CommonConfig.Wallet.Rescan {
		wallet.RescanBlocks()
	}

	cases := []struct {
		desc        string
		startRunNum int
		runBlockNum int
	}{
		{
			desc:        "first round block",
			startRunNum: 5,
			runBlockNum: 5,
		},
		// {
		// 	desc:        "second add blocks",
		// 	startRunNum: 3,
		// 	runBlockNum: 2,
		// },
		// {
		// 	desc:        "third add blocks",
		// 	startRunNum: 100,
		// 	runBlockNum: 100,
		// },
	}

	warnDuration := time.Duration(consensus.ActiveNetParams.BlockTimeInterval*warnTimeNum/warnTimeDenom) * time.Millisecond
	criticalDuration := time.Duration(consensus.ActiveNetParams.BlockTimeInterval*criticalTimeNum/criticalTimeDenom) * time.Millisecond

	for caseIndex, c := range cases {
		beforeBlocks := []*types.Block{}
		afterBlocks := []*types.Block{}
		expectConsensusResultsMap := map[uint64]*state.ConsensusResult{}
		nowConsensusResultsMap := map[uint64]*state.ConsensusResult{}

		for i := 0; i < c.startRunNum; i++ {
			timeStamp := chain.BestBlockHeader().Timestamp + consensus.ActiveNetParams.BlockTimeInterval
			config.CommonConfig.XPrv, err = getXprv(chain, store, timeStamp)
			if err != nil {
				t.Fatal(err)
			}

			block, err := proposal.NewBlockTemplate(chain, accounts, timeStamp, warnDuration, criticalDuration)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := chain.ProcessBlock(block); err != nil {
				t.Fatal(err)
			}

			blockHash := block.Hash()
			gotBlock, err := store.GetBlock(&blockHash)
			beforeBlocks = append(beforeBlocks, gotBlock)
			if err != nil {
				t.Fatal(err)
			}
		}

		for i := 0; i < len(beforeBlocks); i++ {
			block := beforeBlocks[i]
			blockHash := block.Hash()
			consensusResult, err := chain.GetConsensusResultByHash(&blockHash)
			if err != nil {

				t.Fatal(err)
			}

			expectConsensusResultsMap[state.CalcVoteSeq(block.Height)] = consensusResult
		}

		expectChainStatus := store.GetStoreStatus()
		expectHeight := chain.BestBlockHeight()
		for i := 0; i < c.runBlockNum; i++ {
			timeStamp := chain.BestBlockHeader().Timestamp + consensus.ActiveNetParams.BlockTimeInterval
			config.CommonConfig.XPrv, err = getXprv(chain, store, timeStamp)
			if err != nil {
				t.Fatal(err)
			}

			//block, err := proposal.NewBlockTemplate(chain, txPool, nil, timeStamp)
			block, err := proposal.NewBlockTemplate(chain, accounts, timeStamp, warnDuration, criticalDuration)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := chain.ProcessBlock(block); err != nil {
				t.Fatal(err)
			}

			blockHash := block.Hash()
			gotBlock, err := store.GetBlock(&blockHash)
			afterBlocks = append(afterBlocks, gotBlock)
			if err != nil {
				t.Fatal(err)
			}
		}

		if err = chain.Rollback(expectHeight); err != nil {
			t.Fatal(err)
		}

		nowHeight := chain.BestBlockHeight()
		if expectHeight != nowHeight {
			t.Fatalf("%s test failed, expected: %d, now: %d", c.desc, expectHeight, nowHeight)
		}

		if !testutil.DeepEqual(store.GetStoreStatus(), expectChainStatus) {
			t.Errorf("got block status:%v, expect block status:%v", store.GetStoreStatus(), expectChainStatus)
		}

		for i := 0; i < len(beforeBlocks); i++ {
			block := beforeBlocks[i]
			blockHash := block.Hash()
			gotBlock, err := store.GetBlock(&blockHash)
			if err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(gotBlock, block) {
				t.Errorf("case %v,%v: block mismatch: have %x, want %x", caseIndex, i, gotBlock, block)
			}

			gotBlockHeader, err := store.GetBlockHeader(&blockHash)
			if err != nil {
				t.Fatal(err)
			}

			if !testutil.DeepEqual(block.BlockHeader, *gotBlockHeader) {
				t.Errorf("got block header:%v, expect block header:%v", gotBlockHeader, block.BlockHeader)
			}

			consensusResult, err := chain.GetConsensusResultByHash(&blockHash)
			if err != nil {

				t.Fatal(err)
			}

			nowConsensusResultsMap[state.CalcVoteSeq(block.Height)] = consensusResult
		}

		if !testutil.DeepEqual(expectConsensusResultsMap, nowConsensusResultsMap) {
			t.Errorf("consensusResult is not equal!")
		}

		finalSeq := state.CalcVoteSeq(chain.BestBlockHeight())
		for i := 0; i < len(afterBlocks); i++ {
			block := afterBlocks[i]
			blockHash := block.Hash()
			_, err := store.GetBlockHeader(&blockHash)
			if err == nil {
				t.Errorf("this block should not exists!")
			}

			// Code below tests will be update in later PR
			// this code pr is too big
			// to test consensusResult whether right or not
			seq := state.CalcVoteSeq(block.Height)
			if seq > finalSeq {
				consensusResult, err := getConsensusResult(chain, store, seq, &block.BlockHeader)
				if err == nil {
					t.Errorf("why this result existed! %v, %v", consensusResult, err)
				}
			}
		}
	}
}
