package test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/bytom/vapor/config"
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
	fmt.Println("secretkey_key:", xprv)

	xpub := xprv.XPub()
	fmt.Println("public_key:", xpub)

	derivate_key := xprv.Derive(fedConsensusPath)
	fmt.Println("derivate_secret_key:", derivate_key)

	derivate_public_key := derivate_key.XPub()
	fmt.Println("derivate_public_key", derivate_public_key)
}

func newFederationConfig() *config.FederationConfig {
	return &config.FederationConfig{
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

func TestProposalTemplate(t *testing.T) {
	db := dbm.NewDB("block_test_db", "leveldb", "block_test_db")
	defer os.RemoveAll("block_test_db")

	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Federation = newFederationConfig()

	xp := xprv("c87f8d0f4bb4b0acbb7f69f1954c4f34d4476e114fffa7b0c853992474a9954a273c2d8f2642a7baf94ebac88f1625af9f5eaf3b13a90de27eec3de78b9fb9ca")
	config.CommonConfig.XPrv = &xp

	store := database.NewStore(db)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)
	chain, err := protocol.NewChain(store, txPool, dispatcher)

	cases := []struct {
		desc        string
		startRunNum int
		runBlockNum int
	}{
		{
			desc:        "first round block",
			startRunNum: 3,
			runBlockNum: 2,
		},
	}

	for _, c := range cases {
		for i := 0; i < c.startRunNum; i++ {
			timeStamp := uint64(time.Now().UnixNano()/1e6 + int64(i))
			config.CommonConfig.XPrv, err = getXprv(chain, store, timeStamp)
			if err != nil {
				t.Fatal(err)
			}

			block, err := proposal.NewBlockTemplate(chain, txPool, nil, timeStamp)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := chain.ProcessBlock(block); err != nil {
				t.Fatal(err)
			}

			time.Sleep(time.Duration(1) * time.Second)
		}

		nowHeight := chain.BestBlockHeight()
		for i := 0; i < c.runBlockNum; i++ {
			timeStamp := uint64(time.Now().UnixNano()/1e6 + int64(i))
			config.CommonConfig.XPrv, err = getXprv(chain, store, timeStamp)
			if err != nil {
				t.Fatal(err)
			}

			block, err := proposal.NewBlockTemplate(chain, txPool, nil, timeStamp)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := chain.ProcessBlock(block); err != nil {
				t.Fatal(err)
			}

			time.Sleep(time.Duration(1) * time.Second)
		}

		if err = chain.Rollback(nowHeight); err != nil {
			t.Fatal(err)
		}

		afterHeight := chain.BestBlockHeight()

		if nowHeight != afterHeight {
			t.Fatalf("%s test failed, expected: %d, now: %d", c.desc, nowHeight, afterHeight)
		}
	}
}
