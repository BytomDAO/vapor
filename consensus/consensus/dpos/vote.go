package dpos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"

	cmn "github.com/tendermint/tmlibs/common"

	"github.com/vapor/config"
	"github.com/vapor/consensus"
	engine "github.com/vapor/consensus/consensus"
	"github.com/vapor/protocol/bc"
)

const (
	VoteFile                 = "vote.dat"
	DelegateFile             = "delegate.dat"
	BalanceFile              = "balance.dat"
	ControlFile              = "control.dat"
	InvalidVoteTxFile        = "invalidvotetx.dat"
	DelegateMultiAddressFile = "delegatemultiaddress.dat"

	ForgerFile = "forger.data"
)

type Vote struct {
	DelegateVoters            map[string]map[string]bool
	VoterDelegates            map[string]map[string]bool
	lockVoter                 sync.Mutex
	DelegateName              map[string]string
	NameDelegate              map[string]string
	lockRegister              sync.Mutex
	HashHeightInvalidVote     map[bc.Hash]uint64
	lockHashHeightInvalidVote sync.Mutex
	AddressBalances           map[string]uint64
	DelegateMultiaddress      map[string]uint64

	filePath                 string
	delegateFileName         string
	voteFileName             string
	balanceFileName          string
	controlFileName          string
	invalidVoteTxFileName    string
	delegateMultiaddressName string

	forgerFileName string

	oldBlockHeight uint64
	oldBlockHash   bc.Hash
}

func newVote(blockHeight uint64, blockHash bc.Hash) (*Vote, error) {
	vote := &Vote{
		DelegateVoters:        make(map[string]map[string]bool),
		VoterDelegates:        make(map[string]map[string]bool),
		DelegateName:          make(map[string]string),
		NameDelegate:          make(map[string]string),
		HashHeightInvalidVote: make(map[bc.Hash]uint64),
		AddressBalances:       make(map[string]uint64),
		DelegateMultiaddress:  make(map[string]uint64),
	}

	err := vote.New(blockHeight, blockHash)
	return vote, err
}

func (v *Vote) New(blockHeight uint64, blockHash bc.Hash) error {
	v.filePath = filepath.Join(config.CommonConfig.RootDir, "dpos")
	v.delegateFileName = filepath.Join(v.filePath, DelegateFile)
	v.balanceFileName = filepath.Join(v.filePath, BalanceFile)
	v.controlFileName = filepath.Join(v.filePath, ControlFile)
	v.invalidVoteTxFileName = filepath.Join(v.filePath, InvalidVoteTxFile)
	v.delegateMultiaddressName = filepath.Join(v.filePath, DelegateMultiAddressFile)
	v.forgerFileName = filepath.Join(v.filePath, ForgerFile)
	if blockHeight == 0 {
		if err := cmn.EnsureDir(v.filePath, 0700); err != nil {
			return err
		}
	} else {
		if err := v.load(blockHeight, blockHash); err != nil {
			return err
		}
	}
	return nil
}

func (v *Vote) ProcessRegister(delegateAddress string, delegateName string, hash bc.Hash, height uint64) bool {
	v.lockRegister.Lock()
	defer v.lockRegister.Unlock()

	if _, ok := v.DelegateName[delegateAddress]; ok {
		v.AddInvalidVote(hash, height)
		return false
	}
	if _, ok := v.NameDelegate[delegateName]; ok {
		v.AddInvalidVote(hash, height)
		return false
	}
	v.DelegateName[delegateAddress] = delegateName
	v.NameDelegate[delegateName] = delegateAddress
	return true
}

func (v *Vote) ProcessVote(voterAddress string, delegates []string, hash bc.Hash, height uint64) bool {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	votes := 0

	if delegates, ok := v.VoterDelegates[voterAddress]; ok {
		votes = len(delegates)
	}

	if votes+len(delegates) > consensus.MaxNumberOfVotes {
		v.AddInvalidVote(hash, height)
		return false
	}

	for _, delegate := range delegates {
		if _, ok := v.DelegateName[delegate]; !ok {
			v.AddInvalidVote(hash, height)
			return false
		}
		if voters, ok := v.DelegateVoters[delegate]; ok {
			if _, ok = voters[voterAddress]; ok {
				v.AddInvalidVote(hash, height)
				return false
			} else {
				voters[voterAddress] = true
				v.DelegateVoters[delegate] = voters
			}
		} else {
			voters := make(map[string]bool)
			voters[voterAddress] = true
			v.DelegateVoters[delegate] = voters
		}
		if dg, ok := v.VoterDelegates[voterAddress]; ok {
			dg[delegate] = true
			v.VoterDelegates[voterAddress] = dg
		} else {
			dg := make(map[string]bool)
			dg[delegate] = true
			v.VoterDelegates[voterAddress] = dg
		}
	}
	return true
}

func (v *Vote) ProcessCancelVote(voterAddress string, delegates []string, hash bc.Hash, height uint64) bool {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	for _, delegate := range delegates {
		if voters, ok := v.DelegateVoters[delegate]; ok {
			if _, ok = voters[voterAddress]; !ok {
				v.AddInvalidVote(hash, height)
				return false
			} else {
				if len(voters) == 1 {
					delete(v.DelegateVoters, delegate)
				} else {
					delete(v.DelegateVoters[delegate], voterAddress)
				}
			}
		} else {
			v.AddInvalidVote(hash, height)
			return false
		}
	}

	if item, ok := v.VoterDelegates[voterAddress]; ok {
		for _, delegate := range delegates {
			delete(v.VoterDelegates[voterAddress], delegate)
		}
		if len(item) == 0 {
			delete(v.VoterDelegates, voterAddress)
		}
	}

	return true
}

func (v *Vote) load(blockHeight uint64, blockHash bc.Hash) error {
	if err := v.repairFile(blockHeight, blockHash); err != nil {
		return err
	}
	return v.read()
}

func (v *Vote) Delete(blockHash bc.Hash) {
	os.Remove(v.delegateFileName + "-" + blockHash.String())
	os.Remove(v.voteFileName + "-" + blockHash.String())
	os.Remove(v.balanceFileName + "-" + blockHash.String())
}

func (v *Vote) Store(blockHeight uint64, blockHash bc.Hash) error {
	if blockHeight == 0 {
		return nil
	}
	if err := v.Write(blockHash); err != nil {
		return err
	}

	if err := v.WriteControlFile(blockHeight, blockHash, v.controlFileName+"-temp"); err != nil {
		v.Delete(blockHash)
		return err
	}

	var (
		blockHeightTemp uint64
		blockHashTemp   bc.Hash
	)

	if err := v.ReadControlFile(&blockHeightTemp, &blockHashTemp, v.controlFileName); err != nil {
		os.Rename(v.controlFileName, v.controlFileName+"-old")
		os.Rename(v.controlFileName+"-temp", v.controlFileName)
		os.Remove(v.controlFileName + "-old")
	} else {
		v.Delete(blockHashTemp)
	}

	return nil
}

type forger struct {
	DelegateVoters        map[string]map[string]bool `json:"delegate_voters"`
	VoterDelegates        map[string]map[string]bool `json:"voter_delegates"`
	DelegateName          map[string]string          `json:"delegate_name"`
	NameDelegate          map[string]string          `json:"name_delegate"`
	HashHeightInvalidVote map[bc.Hash]uint64         `json:"hash_height_invalid_vote"`
	AddressBalances       map[string]uint64          `json:"address_balance"`
	DelegateMultiaddress  map[string]uint64          `json:"delegate_multiaddress"`
}

func (v *Vote) Write(blockHash bc.Hash) error {

	f := forger{
		DelegateVoters:        v.DelegateVoters,
		VoterDelegates:        v.VoterDelegates,
		DelegateName:          v.DelegateName,
		NameDelegate:          v.NameDelegate,
		HashHeightInvalidVote: v.HashHeightInvalidVote,
		AddressBalances:       v.AddressBalances,
		DelegateMultiaddress:  v.DelegateMultiaddress,
	}
	fileObj, err := os.OpenFile(v.forgerFileName+"-"+blockHash.String(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fileObj.Close()

	var data []byte

	if data, err = json.Marshal(&f); err != nil {
		return err
	}

	if _, err = fileObj.Write(data); err != nil {
		return err
	}

	return nil
}

func (v *Vote) read() error {

	if err := v.ReadControlFile(&v.oldBlockHeight, &v.oldBlockHash, v.controlFileName); err != nil {
		return err
	}

	data, err := ioutil.ReadFile(v.forgerFileName + "-" + v.oldBlockHash.String())
	if err != nil {
		return err
	}

	f := &forger{}

	if err = json.Unmarshal(data, f); err != nil {
		return err
	}

	v.DelegateVoters = f.DelegateVoters
	v.VoterDelegates = f.VoterDelegates
	v.DelegateName = f.DelegateName
	v.NameDelegate = f.NameDelegate
	v.HashHeightInvalidVote = f.HashHeightInvalidVote
	v.AddressBalances = f.AddressBalances
	v.DelegateMultiaddress = f.DelegateMultiaddress

	return nil
}

func (v *Vote) repairFile(blockHeight uint64, blockHash bc.Hash) error {

	cmn.EnsureDir(v.filePath, 0700)

	fileName := v.controlFileName + "-temp"

	var (
		blockHeightTmp uint64
		blockHashTmp   bc.Hash
	)

	if cmn.FileExists(fileName) {
		if err := v.ReadControlFile(&blockHeightTmp, &blockHashTmp, fileName); err != nil {
			return err
		}
		if cmn.FileExists(v.forgerFileName + "-" + blockHashTmp.String()) {
			os.Rename(fileName, v.controlFileName)
			return nil
		}
		os.Remove(fileName)
	}

	fileName = v.controlFileName + "-old"

	if cmn.FileExists(fileName) {
		if err := v.ReadControlFile(&blockHeightTmp, &blockHashTmp, fileName); err != nil {
			return err
		}
		if cmn.FileExists(v.forgerFileName + "-" + blockHashTmp.String()) {
			os.Rename(fileName, v.controlFileName)
			return nil
		}
		os.Remove(fileName)
	}

	fileName = v.controlFileName
	if cmn.FileExists(fileName) {
		if err := v.ReadControlFile(&blockHeightTmp, &blockHashTmp, fileName); err != nil {
			return err
		}
		if cmn.FileExists(v.forgerFileName + "-" + blockHashTmp.String()) {
			return nil
		}
	}

	return fmt.Errorf("repairFile fail in %d height", blockHeightTmp)
}

func (v *Vote) GetTopDelegateInfo(minHoldBalance uint64, delegateNum uint64) []Delegate {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	var result []Delegate
	for k, value := range v.DelegateVoters {
		votes := uint64(0)
		for address := range value {
			votes += v.GetAddressBalance(address)
		}
		if v.GetAddressBalance(k) >= minHoldBalance {
			result = append(result, Delegate{k, votes})
		}
	}
	sort.Sort(DelegateWrapper{result, func(p, q *Delegate) bool {
		if p.Votes < q.Votes {
			return false
		} else if p.Votes > q.Votes {
			return true
		}
		return bytes.Compare([]byte(p.DelegateAddress), []byte(q.DelegateAddress)) > 0
	}})

	for k := range v.DelegateName {
		if uint64(len(result)) >= delegateNum {
			break
		}
		if v.GetAddressBalance(k) < consensus.MinHoldBalance {
			continue
		}
		if _, ok := v.DelegateVoters[k]; !ok {
			result = append(result, Delegate{k, 0})
		}
	}
	if uint64(len(result)) <= delegateNum {
		return result
	}
	result = result[:delegateNum]
	return result
}

func (v *Vote) GetAddressBalance(address string) uint64 {

	if votes, ok := v.AddressBalances[address]; ok {
		return votes
	}

	return 0
}

type control struct {
	BlockHeight uint64  `json:"block_height"`
	BlockHash   bc.Hash `json:"block_hash"`
}

func (v *Vote) ReadControlFile(blockHeight *uint64, blockHash *bc.Hash, fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	c := &control{}

	if err = json.Unmarshal(data, c); err != nil {
		return err
	}

	*blockHash = c.BlockHash
	*blockHeight = c.BlockHeight
	return nil
}

func (v *Vote) WriteControlFile(blockHeight uint64, blockHash bc.Hash, fileName string) error {

	fileObj, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fileObj.Close()

	c := control{
		BlockHeight: blockHeight,
		BlockHash:   blockHash,
	}

	var data []byte

	if data, err = json.Marshal(&c); err != nil {
		return err
	}

	if _, err = fileObj.Write(data); err != nil {
		return err
	}

	return nil
}

func (v *Vote) UpdateAddressBalance(addressBalance []engine.AddressBalance) {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	mapBalance := make(map[string]int64)

	for _, value := range addressBalance {
		if value.Balance == 0 {
			continue
		}
		mapBalance[value.Address] += value.Balance
	}
	for addr, balance := range mapBalance {
		v.updateAddressBalance(addr, balance)
	}
}

func (v *Vote) updateAddressBalance(address string, value int64) {
	if val, ok := v.AddressBalances[address]; ok {
		banlance := int64(val) + value
		if banlance < 0 {
			cmn.Exit("The balance was negative")
		}
		if banlance == 0 {
			delete(v.AddressBalances, address)
		} else {
			v.AddressBalances[address] = uint64(banlance)
		}
	} else {
		if value < 0 {
			cmn.Exit("The balance was negative")
		}
		if value > 0 {
			v.AddressBalances[address] = uint64(value)
		}
	}
}

func (v *Vote) AddInvalidVote(hash bc.Hash, height uint64) {
	v.lockHashHeightInvalidVote.Lock()
	defer v.lockHashHeightInvalidVote.Unlock()
	v.HashHeightInvalidVote[hash] = height
}
func (v *Vote) DeleteInvalidVote(height uint64) {
	v.lockHashHeightInvalidVote.Lock()
	defer v.lockHashHeightInvalidVote.Unlock()
	for k, value := range v.HashHeightInvalidVote {
		if value <= height {
			delete(v.HashHeightInvalidVote, k)
		}
	}
}

func (v *Vote) GetOldBlockHeight() uint64 {
	return v.oldBlockHeight
}

func (v *Vote) GetOldBlockHash() bc.Hash {
	return v.oldBlockHash
}

func (v *Vote) GetDelegate(name string) string {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	if delegate, ok := v.NameDelegate[name]; ok {
		return delegate
	}
	return ""
}

func (v *Vote) GetDelegateName(address string) string {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()
	if name, ok := v.DelegateName[address]; ok {
		return name
	}
	return ""
}

func (v *Vote) HaveVote(voter string, delegate string) bool {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	if voters, ok := v.DelegateVoters[delegate]; ok {
		if _, ok := voters[voter]; ok {
			return true
		}
	}

	return false
}

func (v *Vote) HaveDelegate(name string, delegate string) bool {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	if n, ok := v.DelegateName[delegate]; ok {
		if n == name {
			return true
		}
	}

	return false
}

func (v *Vote) GetVotedDelegates(voter string) []string {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()
	var results []string
	if delegates, ok := v.VoterDelegates[voter]; ok {
		for delegate, _ := range delegates {
			results = append(results, delegate)
		}
	}
	return results
}

func (v *Vote) ListDelegates() map[string]string {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()
	return v.NameDelegate
}

func (v *Vote) GetDelegateVotes(delegate string) uint64 {
	votes := uint64(0)
	if voters, ok := v.DelegateVoters[delegate]; ok {
		for voter := range voters {
			votes += v.GetAddressBalance(voter)
		}
	}
	return votes
}

func (v *Vote) GetDelegateVoters(delegate string) []string {
	v.lockVoter.Lock()
	defer v.lockVoter.Unlock()

	var result []string

	if voters, ok := v.DelegateVoters[delegate]; ok {
		for voter := range voters {
			result = append(result, voter)
		}
	}

	return result
}
