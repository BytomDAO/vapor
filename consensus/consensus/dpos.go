package consensus

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/vapor/crypto/ed25519/chainkd"

	"github.com/vapor/consensus"

	"github.com/vapor/chain"
	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type Delegate struct {
	DelegateAddress string `json:"delegate_address"`
	Votes           uint64 `json:"votes"`
}

type DelegateInfo struct {
	Delegates []Delegate `json:"delegates"`
}

//OP_FAIL PUBKEY SIG(block.time) DELEGATE_IDS
func DelegateInfoToScript(delegateInfo DelegateInfo, xpub chainkd.XPub, h bc.Hash) {

}

//ScriptToDelegateInfo

const maxConfirmBlockCount = 2

type IrreversibleBlockInfo struct {
	heights    []int64
	hashs      []bc.Hash
	HeightHash map[int64]bc.Hash
}

func newIrreversibleBlockInfo() *IrreversibleBlockInfo {
	o := &IrreversibleBlockInfo{}
	for i := 0; i < maxConfirmBlockCount; i++ {
		o.heights = append(o.heights, -1)
	}
	return o
}

type DposType struct {
	c                         chain.Chain
	MaxDelegateNumber         uint64
	BlockIntervalTime         uint64
	DposStartHeight           uint64
	DposStartTime             uint64
	cSuperForgerAddress       common.Address
	irreversibleBlockFileName string
	cIrreversibleBlockInfo    IrreversibleBlockInfo
	lockIrreversibleBlockInfo sync.Mutex
}

var GDpos = &DposType{}

func init() {
	GDpos.irreversibleBlockFileName = filepath.Join(config.DefaultDataDir(), "dpos", "irreversible_block.dat")
	GDpos.ReadIrreversibleBlockInfo(&GDpos.cIrreversibleBlockInfo)

}

func (d *DposType) ReadIrreversibleBlockInfo(info *IrreversibleBlockInfo) error {
	return nil
}

func (d *DposType) IsMining(cDelegateInfo *DelegateInfo, address common.Address, t uint64) error {

	header := d.c.BestBlockHeader()
	currentLoopIndex := d.GetLoopIndex(t)
	currentDelegateIndex := d.GetDelegateIndex(t)
	prevLoopIndex := d.GetLoopIndex(header.Timestamp)
	prevDelegateIndex := d.GetDelegateIndex(header.Timestamp)

	if currentLoopIndex > prevLoopIndex {
		*cDelegateInfo = d.GetNextDelegates(t)
		if cDelegateInfo.Delegates[currentLoopIndex].DelegateAddress == address.EncodeAddress() {
			return nil
		}
		return errors.New("Is not the current mining node")
	} else if currentLoopIndex == prevLoopIndex && currentDelegateIndex > prevDelegateIndex {
		//currentDelegateInfo := DelegateInfo{}
		currentDelegateInfo, err := d.GetBlockDelegates(header)
		if err != nil {
			return err
		}
		if currentDelegateIndex+1 > uint64(len(currentDelegateInfo.Delegates)) {
			return errors.New("Out of the block node list")
		} else if currentDelegateInfo.Delegates[currentDelegateIndex].DelegateAddress == address.EncodeAddress() {
			return nil
		} else {
			return errors.New("Is not the current mining node")
		}
	} else {
		return errors.New("Time anomaly")
	}
}

func (d *DposType) GetLoopIndex(time uint64) uint64 {
	if time < d.DposStartTime {
		return 0
	}

	return (time - d.DposStartTime) / (d.MaxDelegateNumber * d.BlockIntervalTime)
}

func (d *DposType) GetDelegateIndex(time uint64) uint64 {
	if time < d.DposStartTime {
		return 0
	}

	return (time - d.DposStartTime) % (d.MaxDelegateNumber * d.BlockIntervalTime) / d.BlockIntervalTime
}

func (d *DposType) GetNextDelegates(t uint64) DelegateInfo {
	delegates := DposVote.GetTopDelegateInfo(consensus.MinHoldBalance, d.MaxDelegateNumber-1)
	delegate := Delegate{
		DelegateAddress: d.cSuperForgerAddress.EncodeAddress(),
		Votes:           7,
	}
	delegates = append(delegates, delegate)
	delegateInfo := DelegateInfo{}
	delegateInfo.Delegates = SortDelegate(delegates, t)
	return delegateInfo
}

func (d *DposType) GetBlockDelegates(header *types.BlockHeader) (*DelegateInfo, error) {
	loopIndex := d.GetLoopIndex(header.Timestamp)
	for {
		preHeader, err := d.c.GetHeaderByHash(&header.PreviousBlockHash)
		if err != nil {
			return nil, err
		}
		if header.Height == d.DposStartHeight || d.GetLoopIndex(preHeader.Timestamp) < loopIndex {
			block, err := d.c.GetBlockByHeight(header.Height)
			if err != nil {
				return nil, err
			}
			delegateInfo, err := d.GetBlockDelegate(block)
			if err != nil {
				return nil, err
			}
			return delegateInfo, nil
		}
		header = preHeader
	}
}

func (d *DposType) GetBlockDelegate(block *types.Block) (*DelegateInfo, error) {
	tx := block.Transactions[0]
	var delegate TypedData
	if len(tx.TxData.Inputs) == 1 && tx.TxData.Inputs[0].InputType() == types.CoinbaseInputType {
		if err := json.Unmarshal(tx.TxData.ReferenceData, delegate); err != nil {
			return nil, err
		}
		if delegateInfo, ok := delegate.(*DelegateInfoList); ok {
			return &delegateInfo.Delegate, nil
		}
	}
	return nil, errors.New("The first transaction is not a coinbase transaction")
}

func (d *DposType) CheckCoinbase(tx types.TxData, t uint64, Height uint64) error {

	return nil
}

func (d *DposType) CheckBlockHeader(header types.BlockHeader) error {
	blockT := time.Unix(int64(header.Timestamp), 0)

	if blockT.Sub(time.Now()).Seconds() > 3 {
		return errors.New("block time is error")
	}

	return nil
}

func (d *DposType) CheckBlock(block types.Block, fIsCheckDelegateInfo bool) error {
	blockT := time.Unix(int64(block.Timestamp), 0)

	if blockT.Sub(time.Now()).Seconds() > 3 {
		return errors.New("block time is error")
	}

	if err := d.CheckCoinbase(block.Transactions[0].TxData, block.Timestamp, block.Height); err != nil {
		return err
	}

	preBlock, err := d.c.GetBlockByHash(&block.PreviousBlockHash)
	if err != nil {
		return err
	}

	currentLoopIndex := d.GetLoopIndex(block.Timestamp)
	currentDelegateIndex := d.GetDelegateIndex(block.Timestamp)
	prevLoopIndex := d.GetLoopIndex(preBlock.Timestamp)
	prevDelegateIndex := d.GetDelegateIndex(preBlock.Timestamp)

	delegateInfo := &DelegateInfo{}

	if currentLoopIndex < prevLoopIndex {
		return errors.New("Block time exception")
	} else if currentLoopIndex > prevLoopIndex {
		if fIsCheckDelegateInfo {
			if err := d.CheckBlockDelegate(block); err != nil {
				return err
			}
			d.ProcessIrreversibleBlock(block.Height, block.Hash())
		}
		delegateInfo, err = d.GetBlockDelegate(&block)
		if err != nil {
			return err
		}
	} else {
		if currentDelegateIndex < prevDelegateIndex {
			return errors.New("Block time exception")
		}

		delegateInfo, err = d.GetBlockDelegates(&preBlock.BlockHeader)
		if err != nil {
			return err
		}
	}

	delegateAddress := d.getBlockForgerAddress(block)
	if currentDelegateIndex < uint64(len(delegateInfo.Delegates)) &&
		delegateInfo.Delegates[currentDelegateIndex].DelegateAddress == delegateAddress.EncodeAddress() {
		return nil
	}
	h := block.Hash()
	return fmt.Errorf("CheckBlock GetDelegateID blockhash:%s error", h.String())
}

func (d *DposType) CheckBlockDelegate(block types.Block) error {
	return nil
}

func (d *DposType) ProcessIrreversibleBlock(height uint64, hash bc.Hash) {

}

func (d *DposType) getBlockForgerAddress(block types.Block) common.Address {
	tx := block.Transactions[0].TxData

	if len(tx.Inputs) == 1 && tx.Inputs[0].InputType() == types.CoinbaseInputType {
		address, err := common.NewAddressWitnessPubKeyHash(tx.Outputs[0].ControlProgram[2:], &consensus.ActiveNetParams)
		if err != nil {
			address, err := common.NewAddressWitnessScriptHash(tx.Outputs[0].ControlProgram[2:], &consensus.ActiveNetParams)
			if err != nil {
				return nil
			}
			return address
		}
		return address
	}

	return nil
}

func (d *DposType) IsValidBlockCheckIrreversibleBlock(height uint64, hash bc.Hash) error {
	d.lockIrreversibleBlockInfo.Lock()
	defer d.lockIrreversibleBlockInfo.Unlock()

	if h, ok := d.cIrreversibleBlockInfo.HeightHash[int64(height)]; ok {
		if h != hash {
			return fmt.Errorf("invalid block[%d:%s]", height, hash.String())
		}
	}

	return nil
}

func SortDelegate(delegates []Delegate, t uint64) []Delegate {
	var result []Delegate
	r := getRand(uint64(len(delegates)), int64(t))
	for _, i := range r {
		result = append(result, delegates[i])
	}
	return result
}

func getRand(num uint64, seed int64) []uint64 {
	rand.Seed(seed)
	var r []uint64
	s := make(map[uint64]bool)
	for {
		v := rand.Uint64()
		v %= num
		if _, ok := s[v]; ok {
			continue
		}
		s[v] = true
		r = append(r, v)
		if uint64(len(r)) >= num {
			break
		}
	}

	return r
}
