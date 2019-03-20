package consensus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/vapor/config"

	"github.com/vapor/consensus"
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

type AddressBalance struct {
	Address string
	Balance int64
}

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

var DposVote = Vote{}

func (v *Vote) new(blockHeight uint64, blockHash bc.Hash) error {
	//DefaultDataDir
	v.filePath = filepath.Join(config.DefaultDataDir(), "dpos")
	v.delegateFileName = filepath.Join(v.filePath, DelegateFile)
	v.balanceFileName = filepath.Join(v.filePath, BalanceFile)
	v.controlFileName = filepath.Join(v.filePath, ControlFile)
	v.invalidVoteTxFileName = filepath.Join(v.filePath, InvalidVoteTxFile)
	v.delegateMultiaddressName = filepath.Join(v.filePath, DelegateMultiAddressFile)
	v.forgerFileName = filepath.Join(v.filePath, ForgerFile)
	if blockHeight == 0 {
		if _, err := os.Stat(v.filePath); os.IsNotExist(err) {
			err := os.MkdirAll(v.filePath, 0700)
			if err != nil {
				//return fmt.Errorf("Could not create directory %v. %v", dir, err)
				return err
			}
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
	if _, ok := v.DelegateName[delegateAddress]; !ok {
		v.AddInvalidVote(hash, height)
		return false
	}

	if _, ok := v.NameDelegate[delegateName]; !ok {
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
				v.DelegateVoters[delegate][voterAddress] = true
			}
		} else {
			v.DelegateVoters[delegate][voterAddress] = true
		}
		v.VoterDelegates[voterAddress][delegate] = true
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

func (v *Vote) AddInvalidVote(hash bc.Hash, height uint64) {
	v.lockHashHeightInvalidVote.Lock()
	defer v.lockHashHeightInvalidVote.Unlock()
	v.HashHeightInvalidVote[hash] = height
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
		return err
	}

	var (
		blockHeightTemp uint64
		blockHashTemp   bc.Hash
	)

	if err := v.ReadControlFile(&blockHeightTemp, &blockHashTemp, v.controlFileName); err != nil {
		return err
	}

	os.Rename(v.controlFileName, v.controlFileName+"-old")
	os.Rename(v.controlFileName+"-temp", v.controlFileName)
	os.Remove(v.controlFileName + "-old")

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

	fileObj, err := os.OpenFile(v.forgerFileName+"-"+v.oldBlockHash.String(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	var data []byte

	if _, err = fileObj.Read(data); err != nil {
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
	return nil
}

func (v *Vote) GetTopDelegateInfo(minHoldBalance uint64, delegateNum uint64) []Delegate {
	var result []Delegate

	return result
}

type control struct {
	BlockHeight uint64  `json:"block_height"`
	BlockHash   bc.Hash `json:"block_hash"`
}

func (v *Vote) ReadControlFile(blockHeight *uint64, blockHash *bc.Hash, fileName string) error {

	fileObj, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	var data []byte

	if _, err = fileObj.Read(data); err != nil {
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

func (v *Vote) UpdateAddressBalance(AddressBalance []AddressBalance) {

}
