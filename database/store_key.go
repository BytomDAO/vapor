package database

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/vapor/blockchain/signers"
	"github.com/vapor/common"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

const (
	UTXOPrefix          = "ACU:" //UTXOPrefix is StandardUTXOKey prefix
	SUTXOPrefix         = "SCU:" //SUTXOPrefix is ContractUTXOKey prefix
	ContractPrefix      = "Contract:"
	ContractIndexPrefix = "ContractIndex:"
	AccountPrefix       = "Account:" // AccountPrefix is account ID prefix
	AccountAliasPrefix  = "AccountAlias:"
	AccountIndexPrefix  = "AccountIndex:"
	TxPrefix            = "TXS:"  //TxPrefix is wallet database transactions prefix
	TxIndexPrefix       = "TID:"  //TxIndexPrefix is wallet database tx index prefix
	UnconfirmedTxPrefix = "UTXS:" //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	GlobalTxIndexPrefix = "GTID:" //GlobalTxIndexPrefix is wallet database global tx index prefix
	WalletKey           = "walletInfo"
	MiningAddressKey    = "MiningAddress"
	CoinbaseAbKey       = "CoinbaseArbitrary"
)

var (
	ErrFindAccount = errors.New("Failed to find account")
)

func AccountIndexKey(xpubs []chainkd.XPub) []byte {
	var hash [32]byte
	var xPubs []byte
	cpy := append([]chainkd.XPub{}, xpubs[:]...)
	sort.Sort(signers.SortKeys(cpy))
	for _, xpub := range cpy {
		xPubs = append(xPubs, xpub[:]...)
	}
	sha3pool.Sum256(hash[:], xPubs)
	return append([]byte(AccountIndexPrefix), hash[:]...)
}

func Bip44ContractIndexKey(accountID string, change bool) []byte {
	key := append([]byte(ContractIndexPrefix), accountID...)
	if change {
		return append(key, []byte{1}...)
	}
	return append(key, []byte{0}...)
}

// ContractKey account control promgram store prefix
func ContractKey(hash common.Hash) []byte {
	return append([]byte(ContractPrefix), hash[:]...)
}

// AccountIDKey account id store prefix
func AccountIDKey(accountID string) []byte {
	return append([]byte(AccountPrefix), []byte(accountID)...)
}

// StandardUTXOKey makes an account unspent outputs key to store
func StandardUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return []byte(UTXOPrefix + name)
}

// ContractUTXOKey makes a smart contract unspent outputs key to store
func ContractUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return []byte(SUTXOPrefix + name)
}

func calcDeleteKey(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPrefix, blockHeight))
}

func calcTxIndexKey(txID string) []byte {
	return []byte(TxIndexPrefix + txID)
}

func calcAnnotatedKey(formatKey string) []byte {
	return []byte(TxPrefix + formatKey)
}

func calcUnconfirmedTxKey(formatKey string) []byte {
	return []byte(UnconfirmedTxPrefix + formatKey)
}

func calcGlobalTxIndexKey(txID string) []byte {
	return []byte(GlobalTxIndexPrefix + txID)
}

func CalcGlobalTxIndex(blockHash *bc.Hash, position uint64) []byte {
	txIdx := make([]byte, 40)
	copy(txIdx[:32], blockHash.Bytes())
	binary.BigEndian.PutUint64(txIdx[32:], position)
	return txIdx
}

func formatKey(blockHeight uint64, position uint32) string {
	return fmt.Sprintf("%016x%08x", blockHeight, position)
}

func ContractIndexKey(accountID string) []byte {
	return append([]byte(ContractIndexPrefix), []byte(accountID)...)
}

func AccountAliasKey(name string) []byte {
	return append([]byte(AccountAliasPrefix), []byte(name)...)
}
