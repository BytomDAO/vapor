package mock

import (
	"bytes"
	"container/list"
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	acc "github.com/vapor/account"
	"github.com/vapor/protocol/bc"
)

const (
	desireUtxoCount = 5
	logModule       = "account"
)

type Reservation struct {
	ID     uint64
	UTXOs  []*acc.UTXO
	Change uint64
	Expiry time.Time
}

type UTXOKeeper struct {
	// `sync/atomic` expects the first word in an allocated struct to be 64-bit
	// aligned on both ARM and x86-32. See https://goo.gl/zW7dgq for more details.
	NextIndex     uint64
	Store         acc.AccountStorer
	mtx           sync.RWMutex
	CurrentHeight func() uint64

	Unconfirmed  map[bc.Hash]*acc.UTXO
	Reserved     map[bc.Hash]uint64
	Reservations map[uint64]*Reservation
}

func NewUtxoKeeper(f func() uint64, store acc.AccountStorer) *UTXOKeeper {
	uk := &UTXOKeeper{
		Store:         store,
		CurrentHeight: f,
		Unconfirmed:   make(map[bc.Hash]*acc.UTXO),
		Reserved:      make(map[bc.Hash]uint64),
		Reservations:  make(map[uint64]*Reservation),
	}
	go uk.expireWorker()
	return uk
}

func (uk *UTXOKeeper) expireWorker() {
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for now := range ticker.C {
		uk.expireReservation(now)
	}
}

func (uk *UTXOKeeper) expireReservation(t time.Time) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	for rid, res := range uk.Reservations {
		if res.Expiry.Before(t) {
			uk.cancel(rid)
		}
	}
}

func (uk *UTXOKeeper) cancel(rid uint64) {
	res, ok := uk.Reservations[rid]
	if !ok {
		return
	}

	delete(uk.Reservations, rid)
	for _, utxo := range res.UTXOs {
		delete(uk.Reserved, utxo.OutputID)
	}
}

func (uk *UTXOKeeper) Reserve(accountID string, assetID *bc.AssetID, amount uint64, useUnconfirmed bool, vote []byte, exp time.Time) (*Reservation, error) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	utxos, immatureAmount := uk.FindUtxos(accountID, assetID, useUnconfirmed, vote)
	optUtxos, optAmount, reservedAmount := uk.optUTXOs(utxos, amount)
	if optAmount+reservedAmount+immatureAmount < amount {
		return nil, acc.ErrInsufficient
	}

	if optAmount+reservedAmount < amount {
		if vote != nil {
			return nil, acc.ErrVoteLock
		}
		return nil, acc.ErrImmature
	}

	if optAmount < amount {
		return nil, acc.ErrReserved
	}

	result := &Reservation{
		ID:     atomic.AddUint64(&uk.NextIndex, 1),
		UTXOs:  optUtxos,
		Change: optAmount - amount,
		Expiry: exp,
	}

	uk.Reservations[result.ID] = result
	for _, u := range optUtxos {
		uk.Reserved[u.OutputID] = result.ID
	}
	return result, nil
}

func (uk *UTXOKeeper) ReserveParticular(outHash bc.Hash, useUnconfirmed bool, exp time.Time) (*Reservation, error) {
	uk.mtx.Lock()
	defer uk.mtx.Unlock()

	if _, ok := uk.Reserved[outHash]; ok {
		return nil, acc.ErrReserved
	}

	u, err := uk.FindUtxo(outHash, useUnconfirmed)
	if err != nil {
		return nil, err
	}

	if u.ValidHeight > uk.CurrentHeight() {
		return nil, acc.ErrImmature
	}

	result := &Reservation{
		ID:     atomic.AddUint64(&uk.NextIndex, 1),
		UTXOs:  []*acc.UTXO{u},
		Expiry: exp,
	}
	uk.Reservations[result.ID] = result
	uk.Reserved[u.OutputID] = result.ID
	return result, nil
}

func (uk *UTXOKeeper) FindUtxo(outHash bc.Hash, useUnconfirmed bool) (*acc.UTXO, error) {
	if u, ok := uk.Unconfirmed[outHash]; useUnconfirmed && ok {
		return u, nil
	}

	u := &acc.UTXO{}
	if data := uk.Store.GetStandardUTXO(outHash); data != nil {
		return u, json.Unmarshal(data, u)
	}
	if data := uk.Store.GetContractUTXO(outHash); data != nil {
		return u, json.Unmarshal(data, u)
	}
	return nil, acc.ErrMatchUTXO
}

func (uk *UTXOKeeper) FindUtxos(accountID string, assetID *bc.AssetID, useUnconfirmed bool, vote []byte) ([]*acc.UTXO, uint64) {
	immatureAmount := uint64(0)
	currentHeight := uk.CurrentHeight()
	utxos := []*acc.UTXO{}
	appendUtxo := func(u *acc.UTXO) {
		if u.AccountID != accountID || u.AssetID != *assetID || !bytes.Equal(u.Vote, vote) {
			return
		}
		if u.ValidHeight > currentHeight {
			immatureAmount += u.Amount
		} else {
			utxos = append(utxos, u)
		}
	}

	UTXOs := uk.Store.GetUTXOs()
	for _, UTXO := range UTXOs {
		appendUtxo(UTXO)
	}

	if !useUnconfirmed {
		return utxos, immatureAmount
	}

	for _, u := range uk.Unconfirmed {
		appendUtxo(u)
	}
	return utxos, immatureAmount
}

func (uk *UTXOKeeper) optUTXOs(utxos []*acc.UTXO, amount uint64) ([]*acc.UTXO, uint64, uint64) {
	//sort the utxo by amount, bigger amount in front
	var optAmount, reservedAmount uint64
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Amount > utxos[j].Amount
	})

	//push all the available utxos into list
	utxoList := list.New()
	for _, u := range utxos {
		if _, ok := uk.Reserved[u.OutputID]; ok {
			reservedAmount += u.Amount
			continue
		}
		utxoList.PushBack(u)
	}

	optList := list.New()
	for node := utxoList.Front(); node != nil; node = node.Next() {
		//append utxo if we haven't reached the required amount
		if optAmount < amount {
			optList.PushBack(node.Value)
			optAmount += node.Value.(*acc.UTXO).Amount
			continue
		}

		largestNode := optList.Front()
		replaceList := list.New()
		replaceAmount := optAmount - largestNode.Value.(*acc.UTXO).Amount

		for ; node != nil && replaceList.Len() <= desireUtxoCount-optList.Len(); node = node.Next() {
			replaceList.PushBack(node.Value)
			if replaceAmount += node.Value.(*acc.UTXO).Amount; replaceAmount >= amount {
				optList.Remove(largestNode)
				optList.PushBackList(replaceList)
				optAmount = replaceAmount
				break
			}
		}

		//largestNode remaining the same means that there is nothing to be replaced
		if largestNode == optList.Front() {
			break
		}
	}

	optUtxos := []*acc.UTXO{}
	for e := optList.Front(); e != nil; e = e.Next() {
		optUtxos = append(optUtxos, e.Value.(*acc.UTXO))
	}
	return optUtxos, optAmount, reservedAmount
}
