package dpos

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vapor/chain"
	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	engine "github.com/vapor/consensus/consensus"
	"github.com/vapor/crypto"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm"
)

type Delegate struct {
	DelegateAddress string `json:"delegate_address"`
	Votes           uint64 `json:"votes"`
}

type DelegateWrapper struct {
	delegate []Delegate
	by       func(p, q *Delegate) bool
}

func (dw DelegateWrapper) Len() int {
	return len(dw.delegate)
}
func (dw DelegateWrapper) Swap(i, j int) {
	dw.delegate[i], dw.delegate[j] = dw.delegate[j], dw.delegate[i]
}
func (dw DelegateWrapper) Less(i, j int) bool {
	return dw.by(&dw.delegate[i], &dw.delegate[j])
}

type DelegateInfo struct {
	Delegates []Delegate `json:"delegates"`
}

func (d *DelegateInfo) ConsensusName() string {
	return "dpos"
}

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
		o.hashs = append(o.hashs, bc.Hash{})
	}
	o.HeightHash = make(map[int64]bc.Hash)
	return o
}

type DposType struct {
	c                           chain.Chain
	vote                        *Vote
	MaxDelegateNumber           uint64
	BlockIntervalTime           uint64
	DposStartHeight             uint64
	DposStartTime               uint64
	superForgerAddress          common.Address
	irreversibleBlockFileName   string
	irreversibleBlockInfo       IrreversibleBlockInfo
	lockIrreversibleBlockInfo   sync.Mutex
	maxIrreversibleCount        int
	firstIrreversibleThreshold  uint64
	secondIrreversibleThreshold uint64
}

var GDpos = &DposType{
	maxIrreversibleCount:        10000,
	firstIrreversibleThreshold:  90,
	secondIrreversibleThreshold: 67,
}

func (d *DposType) Init(c chain.Chain, delegateNumber, intervalTime, blockHeight uint64, blockHash bc.Hash) error {
	d.c = c
	vote, err := newVote(blockHeight, blockHash)
	d.vote = vote
	d.MaxDelegateNumber = delegateNumber
	d.BlockIntervalTime = intervalTime
	d.DposStartHeight = 0
	address, _ := common.DecodeAddress("vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep", &consensus.ActiveNetParams)
	d.superForgerAddress = address

	GDpos.irreversibleBlockFileName = filepath.Join(config.CommonConfig.RootDir, "dpos", "irreversible_block.dat")
	GDpos.irreversibleBlockInfo = *newIrreversibleBlockInfo()
	GDpos.ReadIrreversibleBlockInfo(&GDpos.irreversibleBlockInfo)
	header, _ := c.GetHeaderByHeight(d.DposStartHeight)
	d.setStartTime(header.Timestamp)
	return err
}

func (d *DposType) setStartTime(t uint64) {
	d.DposStartTime = t
}

func (d *DposType) IsMining(address common.Address, t uint64) (interface{}, error) {

	header := d.c.BestBlockHeader()
	currentLoopIndex := d.GetLoopIndex(t)
	currentDelegateIndex := d.GetDelegateIndex(t)
	prevLoopIndex := d.GetLoopIndex(header.Timestamp)
	prevDelegateIndex := d.GetDelegateIndex(header.Timestamp)
	if currentLoopIndex > prevLoopIndex {
		delegateInfo := d.GetNextDelegates(t)
		cDelegateInfo := delegateInfo.(*DelegateInfo)
		if uint64(len(cDelegateInfo.Delegates)) < currentDelegateIndex+1 {
			return nil, errors.New("Out of the block node list")
		}
		if cDelegateInfo.Delegates[currentDelegateIndex].DelegateAddress == address.EncodeAddress() {
			return delegateInfo, nil
		}
		return nil, errors.New("Is not the current mining node")
	} else if currentLoopIndex == prevLoopIndex && currentDelegateIndex > prevDelegateIndex {
		currentDelegateInfo, err := d.GetBlockDelegates(header)
		if err != nil {
			return nil, err
		}
		if currentDelegateIndex+1 > uint64(len(currentDelegateInfo.Delegates)) {
			return nil, errors.New("Out of the block node list")
		} else if currentDelegateInfo.Delegates[currentDelegateIndex].DelegateAddress == address.EncodeAddress() {
			return nil, nil
		} else {
			return nil, errors.New("Is not the current mining node")
		}
	} else {
		return nil, errors.New("Time anomaly")
	}
}

func (d *DposType) ProcessRegister(delegateAddress string, delegateName string, hash bc.Hash, height uint64) bool {
	return d.vote.ProcessRegister(delegateAddress, delegateName, hash, height)
}

func (d *DposType) ProcessVote(voterAddress string, delegates []string, hash bc.Hash, height uint64) bool {
	return d.vote.ProcessVote(voterAddress, delegates, hash, height)
}

func (d *DposType) ProcessCancelVote(voterAddress string, delegates []string, hash bc.Hash, height uint64) bool {
	return d.vote.ProcessCancelVote(voterAddress, delegates, hash, height)
}

func (d *DposType) UpdateAddressBalance(addressBalance []engine.AddressBalance) {
	d.vote.UpdateAddressBalance(addressBalance)
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

func (d *DposType) GetNextDelegates(t uint64) interface{} {
	delegates := d.vote.GetTopDelegateInfo(config.CommonConfig.Consensus.MinVoterBalance, d.MaxDelegateNumber-1)
	delegate := Delegate{
		DelegateAddress: d.superForgerAddress.EncodeAddress(),
		Votes:           7,
	}
	delegates = append(delegates, delegate)
	delegateInfo := DelegateInfo{}
	delegateInfo.Delegates = delegates //SortDelegate(delegates, t)
	return &delegateInfo
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
	if len(tx.TxData.Inputs) == 1 && tx.TxData.Inputs[0].InputType() == types.CoinbaseInputType {
		msg := &DposMsg{}
		if err := json.Unmarshal(tx.TxData.ReferenceData, msg); err != nil {
			return nil, err
		}
		if msg.Type == vm.OP_DELEGATE {
			delegateInfo := &DelegateInfoList{}
			if err := json.Unmarshal(msg.Data, delegateInfo); err != nil {
				return nil, err
			}
			return &delegateInfo.Delegate, nil
		}

	}
	return nil, errors.New("The first transaction is not a coinbase transaction")
}

func (d *DposType) CheckCoinbase(tx types.TxData, t uint64, Height uint64) error {
	msg := &DposMsg{}
	if err := json.Unmarshal(tx.ReferenceData, msg); err != nil {
		return err
	}
	if msg.Type == vm.OP_DELEGATE {
		delegateInfo := &DelegateInfoList{}
		if err := json.Unmarshal(msg.Data, delegateInfo); err != nil {
			return err
		}
		buf := [8]byte{}
		binary.LittleEndian.PutUint64(buf[:], t)

		if !delegateInfo.Xpub.Verify(buf[:], delegateInfo.SigTime) {
			return errors.New("CheckBlock CheckCoinbase: Verification signature error")
		}
		var (
			address common.Address
			err     error
		)
		address, err = common.NewAddressWitnessPubKeyHash(tx.Outputs[0].ControlProgram[2:], &consensus.ActiveNetParams)
		if err != nil {
			return err
		}
		derivedPK := delegateInfo.Xpub.PublicKey()
		pubHash := crypto.Ripemd160(derivedPK)

		addressDet, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
		if err != nil {
			return err
		}

		if addressDet.EncodeAddress() == address.EncodeAddress() {
			return nil
		}
	}
	return errors.New("CheckBlock CheckCoinbase error")
}

func (d *DposType) CheckBlockHeader(header types.BlockHeader) error {
	blockT := time.Unix(int64(header.Timestamp), 0)

	if blockT.Sub(time.Now()).Seconds() > float64(d.BlockIntervalTime) {
		return errors.New("block time is error")
	}

	if header.Height > d.DposStartHeight {
		header, _ := d.c.GetHeaderByHeight(d.DposStartHeight)
		d.setStartTime(header.Timestamp)
	}

	preHeader, err := d.c.GetHeaderByHash(&header.PreviousBlockHash)
	if err != nil {
		return err
	}

	currentLoopIndex := d.GetLoopIndex(header.Timestamp)
	currentDelegateIndex := d.GetDelegateIndex(header.Timestamp)
	prevLoopIndex := d.GetLoopIndex(preHeader.Timestamp)
	prevDelegateIndex := d.GetDelegateIndex(preHeader.Timestamp)
	if currentLoopIndex > prevLoopIndex ||
		(currentLoopIndex == prevLoopIndex && currentDelegateIndex > prevDelegateIndex) {
		return nil
	}

	return errors.New("DPoS CheckBlockHeader error")
}

func (d *DposType) CheckBlock(block types.Block, fIsCheckDelegateInfo bool) error {
	if block.Height > d.DposStartHeight {
		header, _ := d.c.GetHeaderByHeight(d.DposStartHeight)
		d.setStartTime(header.Timestamp)
	}

	blockT := time.Unix(int64(block.Timestamp), 0)
	if blockT.Sub(time.Now()).Seconds() > float64(d.BlockIntervalTime) {
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
	delegateInfo, err := d.GetBlockDelegate(&block)
	if err != nil {
		return err
	}
	nextDelegateInfoInterface := d.GetNextDelegates(block.Timestamp)
	nextDelegateInfo := nextDelegateInfoInterface.(*DelegateInfo)
	if len(delegateInfo.Delegates) != len(nextDelegateInfo.Delegates) {
		return errors.New("The delegates num is not correct in block")
	}

	for index, v := range delegateInfo.Delegates {
		if v.DelegateAddress != nextDelegateInfo.Delegates[index].DelegateAddress {
			return errors.New("The delegates address is not correct in block")
		}
	}

	return nil
}

func (d *DposType) ProcessIrreversibleBlock(height uint64, hash bc.Hash) {
	d.lockIrreversibleBlockInfo.Lock()
	defer d.lockIrreversibleBlockInfo.Unlock()
	i := 0
	for i = maxConfirmBlockCount - 1; i >= 0; i-- {
		if d.irreversibleBlockInfo.heights[i] < 0 || int64(height) < d.irreversibleBlockInfo.heights[i] {
			d.irreversibleBlockInfo.heights[i] = -1
		} else {
			level := (height - uint64(d.irreversibleBlockInfo.heights[i])) * 100
			if level >= d.MaxDelegateNumber*d.firstIrreversibleThreshold {
				d.AddIrreversibleBlock(int64(height), hash)
			} else if level >= d.MaxDelegateNumber*d.secondIrreversibleThreshold {
				if i == maxConfirmBlockCount-1 {
					d.AddIrreversibleBlock(int64(height), hash)
					for k := 0; k < maxConfirmBlockCount-1; k++ {
						d.irreversibleBlockInfo.heights[k] = d.irreversibleBlockInfo.heights[k+1]
						d.irreversibleBlockInfo.hashs[k] = d.irreversibleBlockInfo.hashs[k+1]
					}
					d.irreversibleBlockInfo.heights[i] = int64(height)
					d.irreversibleBlockInfo.hashs[i] = hash
					return
				} else {
					d.irreversibleBlockInfo.heights[i+1] = int64(height)
					d.irreversibleBlockInfo.hashs[i+1] = hash
					return
				}

			}
			for k := 0; k < maxConfirmBlockCount; k++ {
				d.irreversibleBlockInfo.heights[k] = -1
			}
			d.irreversibleBlockInfo.heights[0] = int64(height)
			d.irreversibleBlockInfo.hashs[0] = hash
			return

		}
	}
	if i < 0 {
		d.irreversibleBlockInfo.heights[0] = int64(height)
		d.irreversibleBlockInfo.hashs[0] = hash
	}
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

	if h, ok := d.irreversibleBlockInfo.HeightHash[int64(height)]; ok {
		if h != hash {
			return fmt.Errorf("invalid block[%d:%s]", height, hash.String())
		}
	}

	return nil
}

func (d *DposType) ReadIrreversibleBlockInfo(info *IrreversibleBlockInfo) error {
	f, err := os.Open(d.irreversibleBlockFileName)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		var height int64
		var hashString string
		n, err := fmt.Sscanf(line, "%d;%s\n", &height, &hashString)
		if err != nil || n != 2 {
			return errors.New("parse error for ReadIrreversibleBlockInfo ")
		}
		var hash bc.Hash
		if err := hash.UnmarshalText([]byte(hashString)); err != nil {
			return err
		}
		d.AddIrreversibleBlock(height, hash)
	}
}

type Int64Slice []int64

func (a Int64Slice) Len() int {
	return len(a)
}
func (a Int64Slice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a Int64Slice) Less(i, j int) bool {
	return a[i] < a[j]
}

func (d *DposType) WriteIrreversibleBlockInfo() error {
	if len(d.irreversibleBlockInfo.HeightHash) == 0 {
		return nil
	}

	f, err := os.Create(d.irreversibleBlockFileName)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	var keys []int64
	for k := range d.irreversibleBlockInfo.HeightHash {
		keys = append(keys, k)
	}

	sort.Sort(Int64Slice(keys))

	for _, k := range keys {
		data, _ := d.irreversibleBlockInfo.HeightHash[k].MarshalText()
		line := fmt.Sprintf("%d;%s\n", k, string(data))
		w.WriteString(line)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func (d *DposType) AddIrreversibleBlock(height int64, hash bc.Hash) {
	for k, _ := range d.irreversibleBlockInfo.HeightHash {
		if len(d.irreversibleBlockInfo.HeightHash) > d.maxIrreversibleCount {
			delete(d.irreversibleBlockInfo.HeightHash, k)
		} else {
			break
		}
	}
	d.irreversibleBlockInfo.HeightHash[height] = hash
	d.vote.DeleteInvalidVote(uint64(height))
}

func (d *DposType) GetSuperForgerAddress() common.Address {
	return d.superForgerAddress
}

func (d *DposType) GetIrreversibleBlock() {

}

func (d *DposType) GetOldBlockHeight() uint64 {
	return d.vote.GetOldBlockHeight()
}

func (d *DposType) GetOldBlockHash() bc.Hash {
	return d.vote.GetOldBlockHash()
}

func (d *DposType) ListDelegates() map[string]string {
	return d.vote.ListDelegates()
}

func (d *DposType) GetDelegateVotes(delegate string) uint64 {
	return d.vote.GetDelegateVotes(delegate)
}

func (d *DposType) GetDelegateVoters(delegate string) []string {
	return d.vote.GetDelegateVoters(delegate)
}

func (d *DposType) GetDelegate(name string) string {
	return d.vote.GetDelegate(name)

}

func (d *DposType) GetDelegateName(address string) string {
	return d.vote.GetDelegateName(address)
}

func (d *DposType) GetAddressBalance(address string) uint64 {
	return d.vote.GetAddressBalance(address)
}

func (d *DposType) GetVotedDelegates(voter string) []string {
	return d.vote.GetVotedDelegates(voter)
}

func (d *DposType) HaveVote(voter, delegate string) bool {
	return d.vote.HaveVote(voter, delegate)
}

func (d *DposType) HaveDelegate(name, delegate string) bool {
	return d.vote.HaveDelegate(name, delegate)
}

func (d *DposType) Finish() error {
	header := d.c.BestBlockHeader()
	if err := d.vote.Store(header.Height, header.Hash()); err != nil {
		return err
	}

	if err := d.WriteIrreversibleBlockInfo(); err != nil {
		return err
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
