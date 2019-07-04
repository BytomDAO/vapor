package wallet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vapor/account"
	acc "github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/event"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

func TestEncodeDecodeGlobalTxIndex(t *testing.T) {
	want := &struct {
		BlockHash bc.Hash
		Position  uint64
	}{
		BlockHash: bc.NewHash([32]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}),
		Position:  1,
	}

	globalTxIdx := CalcGlobalTxIndex(&want.BlockHash, want.Position)
	blockHashGot, positionGot := parseGlobalTxIdx(globalTxIdx)
	if *blockHashGot != want.BlockHash {
		t.Errorf("blockHash mismatch. Get: %v. Expect: %v", *blockHashGot, want.BlockHash)
	}

	if positionGot != want.Position {
		t.Errorf("position mismatch. Get: %v. Expect: %v", positionGot, want.Position)
	}
}

func TestWalletVersion(t *testing.T) {
	// prepare wallet
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	walletStore := NewMockWalletStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	dispatcher := event.NewDispatcher()
	w := mockWallet(walletStore, nil, nil, nil, dispatcher, false)

	walletStatus := new(StatusInfo)
	if err := w.store.SetWalletInfo(walletStatus); err != nil {
		t.Fatal(err)
	}

	status, err := w.store.GetWalletInfo()
	if err != nil {
		t.Fatal(err)
	}
	w.status = *status

	if err := w.checkWalletInfo(); err != errWalletVersionMismatch {
		t.Fatal("fail to detect legacy wallet version")
	}

	// lower wallet version test case
	lowerVersion := StatusInfo{Version: currentVersion - 1}
	if err := w.store.SetWalletInfo(&lowerVersion); err != nil {
		t.Fatal(err)
	}

	status, err = w.store.GetWalletInfo()
	if err != nil {
		t.Fatal(err)
	}
	w.status = *status

	if err := w.checkWalletInfo(); err != errWalletVersionMismatch {
		t.Fatal("fail to detect expired wallet version")
	}
}

// func TestWalletUpdate(t *testing.T) {
// 	dirPath, err := ioutil.TempDir(".", "")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer os.RemoveAll(dirPath)

// 	config.CommonConfig = config.DefaultConfig()
// 	testDB := dbm.NewDB("testdb", "leveldb", "temp")
// 	defer func() {
// 		testDB.Close()
// 		os.RemoveAll("temp")
// 	}()

// 	store := database.NewStore(testDB)
// 	walletStore := NewMockWalletStore(testDB)
// 	dispatcher := event.NewDispatcher()
// 	txPool := protocol.NewTxPool(store, dispatcher)

// 	chain, err := protocol.NewChain(store, txPool, dispatcher)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	accountStore := NewMockAccountStore(testDB)
// 	accountManager := account.NewManager(accountStore, chain)
// 	hsm, err := pseudohsm.New(dirPath)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount", signers.BIP0044)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	controlProg.KeyIndex = 1

// 	reg := asset.NewRegistry(testDB, chain)
// 	// reg := asset.NewRegistry(testDB, nil)
// 	asset := bc.AssetID{V0: 5}

// 	utxos := []*account.UTXO{}
// 	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)
// 	utxos = append(utxos, btmUtxo)
// 	OtherUtxo := mockUTXO(controlProg, &asset)
// 	utxos = append(utxos, OtherUtxo)

// 	_, txData, err := mockTxData(utxos, testAccount)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	tx := types.NewTx(*txData)
// 	block := mockSingleBlock(tx)
// 	txStatus := bc.NewTransactionStatus()
// 	txStatus.SetStatus(0, false)
// 	txStatus.SetStatus(1, false)
// 	store.SaveBlock(block, txStatus)

// 	w := mockWallet(walletStore, accountManager, reg, chain, dispatcher, true)
// 	err = w.AttachBlock(block)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if _, err := w.GetTransactionByTxID(tx.ID.String()); err != nil {
// 		t.Fatal(err)
// 	}

// 	wants, err := w.GetTransactions(testAccount.ID, "", 1, false)
// 	if len(wants) != 1 {
// 		t.Fatal(err)
// 	}

// 	if wants[0].ID != tx.ID {
// 		t.Fatal("account txID mismatch")
// 	}

// 	for position, tx := range block.Transactions {
// 		get := w.store.GetGlobalTransactionIndex(tx.ID.String())
// 		bh := block.BlockHeader.Hash()
// 		expect := CalcGlobalTxIndex(&bh, uint64(position))
// 		if !reflect.DeepEqual(get, expect) {
// 			t.Fatalf("position#%d: compare retrieved globalTxIdx err", position)
// 		}
// 	}
// }

// func TestRescanWallet(t *testing.T) {
// 	// prepare wallet & db
// 	dirPath, err := ioutil.TempDir(".", "")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer os.RemoveAll(dirPath)

// 	config.CommonConfig = config.DefaultConfig()
// 	testDB := dbm.NewDB("testdb", "leveldb", "temp")
// 	walletStore := NewMockWalletStore(testDB)
// 	defer func() {
// 		testDB.Close()
// 		os.RemoveAll("temp")
// 	}()

// 	store := database.NewStore(testDB)
// 	dispatcher := event.NewDispatcher()
// 	txPool := protocol.NewTxPool(store, dispatcher)
// 	chain, err := protocol.NewChain(store, txPool, dispatcher)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	statusInfo := StatusInfo{
// 		Version:  currentVersion,
// 		WorkHash: bc.Hash{V0: 0xff},
// 	}
// 	rawWallet, err := json.Marshal(statusInfo)
// 	if err != nil {
// 		t.Fatal("save wallet info")
// 	}

// 	w := mockWallet(walletStore, nil, nil, chain, dispatcher, false)
// 	w.store.SetWalletInfo(rawWallet)
// 	rawWallet = w.store.GetWalletInfo()
// 	if rawWallet == nil {
// 		t.Fatal("fail to load wallet StatusInfo")
// 	}

// 	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
// 		t.Fatal(err)
// 	}

// 	// rescan wallet
// 	if err := w.loadWalletInfo(); err != nil {
// 		t.Fatal(err)
// 	}

// 	block := config.GenesisBlock()
// 	if w.status.WorkHash != block.Hash() {
// 		t.Fatal("reattach from genesis block")
// 	}
// }

// func TestMemPoolTxQueryLoop(t *testing.T) {
// 	dirPath, err := ioutil.TempDir(".", "")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	config.CommonConfig = config.DefaultConfig()
// 	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
// 	defer func() {
// 		testDB.Close()
// 		os.RemoveAll(dirPath)
// 	}()

// 	store := database.NewStore(testDB)
// 	// store := newMockStore(testDB)
// 	dispatcher := event.NewDispatcher()
// 	txPool := protocol.NewTxPool(store, dispatcher)

// 	chain, err := protocol.NewChain(store, txPool, dispatcher)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	accountStore := NewMockAccountStore(testDB)
// 	accountManager := account.NewManager(accountStore, chain)
// 	hsm, err := pseudohsm.New(dirPath)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount", signers.BIP0044)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	controlProg.KeyIndex = 1

// 	reg := asset.NewRegistry(testDB, chain)
// 	asset := bc.AssetID{V0: 5}

// 	utxos := []*account.UTXO{}
// 	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)
// 	utxos = append(utxos, btmUtxo)
// 	OtherUtxo := mockUTXO(controlProg, &asset)
// 	utxos = append(utxos, OtherUtxo)

// 	_, txData, err := mockTxData(utxos, testAccount)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	tx := types.NewTx(*txData)
// 	//block := mockSingleBlock(tx)
// 	txStatus := bc.NewTransactionStatus()
// 	txStatus.SetStatus(0, false)
// 	walletStore := NewMockWalletStore(testDB)
// 	w, err := NewWallet(walletStore, accountManager, reg, hsm, chain, dispatcher, false)
// 	go w.memPoolTxQueryLoop()
// 	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: protocol.MsgNewTx}})
// 	time.Sleep(time.Millisecond * 10)
// 	if _, err := w.GetUnconfirmedTxByTxID(tx.ID.String()); err != nil {
// 		t.Fatal("dispatch new tx msg error:", err)
// 	}
// 	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: protocol.MsgRemoveTx}})
// 	time.Sleep(time.Millisecond * 10)
// 	txs, err := w.GetUnconfirmedTxs(testAccount.ID)
// 	if err != nil {
// 		t.Fatal("get unconfirmed tx error:", err)
// 	}

// 	if len(txs) != 0 {
// 		t.Fatal("dispatch remove tx msg error")
// 	}

// 	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: 2}})
// }

func mockUTXO(controlProg *account.CtrlProgram, assetID *bc.AssetID) *account.UTXO {
	utxo := &account.UTXO{}
	utxo.OutputID = bc.Hash{V0: 1}
	utxo.SourceID = bc.Hash{V0: 2}
	utxo.AssetID = *assetID
	utxo.Amount = 1000000000
	utxo.SourcePos = 0
	utxo.ControlProgram = controlProg.ControlProgram
	utxo.AccountID = controlProg.AccountID
	utxo.Address = controlProg.Address
	utxo.ControlProgramIndex = controlProg.KeyIndex
	return utxo
}

func mockTxData(utxos []*account.UTXO, testAccount *account.Account) (*txbuilder.Template, *types.TxData, error) {
	tplBuilder := txbuilder.NewBuilder(time.Now())

	for _, utxo := range utxos {
		txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo)
		if err != nil {
			return nil, nil, err
		}
		tplBuilder.AddInput(txInput, sigInst)

		out := &types.TxOutput{}
		if utxo.AssetID == *consensus.BTMAssetID {
			out = types.NewIntraChainOutput(utxo.AssetID, 100, utxo.ControlProgram)
		} else {
			out = types.NewIntraChainOutput(utxo.AssetID, utxo.Amount, utxo.ControlProgram)
		}
		tplBuilder.AddOutput(out)
	}

	return tplBuilder.Build()
}

func mockWallet(store WalletStore, account *account.Manager, asset *asset.Registry, chain *protocol.Chain, dispatcher *event.Dispatcher, txIndexFlag bool) *Wallet {
	wallet := &Wallet{
		store:           store,
		AccountMgr:      account,
		AssetReg:        asset,
		chain:           chain,
		RecoveryMgr:     newRecoveryManager(store, account),
		eventDispatcher: dispatcher,
		TxIndexFlag:     txIndexFlag,
	}
	wallet.txMsgSub, _ = wallet.eventDispatcher.Subscribe(protocol.TxMsgEvent{})
	return wallet
}

func mockSingleBlock(tx *types.Tx) *types.Block {
	return &types.Block{
		BlockHeader: types.BlockHeader{
			Version: 1,
			Height:  1,
		},
		Transactions: []*types.Tx{config.GenesisTx(), tx},
	}
}

type mockStore struct {
	db dbm.DB
	// cache cache
}

func newMockStore(db dbm.DB) *mockStore {
	return &mockStore{
		db: db,
	}
}

func (s *mockStore) BlockExist(hash *bc.Hash) bool                                { return false }
func (s *mockStore) GetBlock(*bc.Hash) (*types.Block, error)                      { return nil, nil }
func (s *mockStore) GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)          { return nil, nil }
func (s *mockStore) GetStoreStatus() *protocol.BlockStoreState                    { return nil }
func (s *mockStore) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) { return nil, nil }
func (s *mockStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error     { return nil }
func (s *mockStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)                 { return nil, nil }
func (s *mockStore) GetVoteResult(uint64) (*state.VoteResult, error)              { return nil, nil }
func (s *mockStore) GetMainChainHash(uint64) (*bc.Hash, error)                    { return nil, nil }
func (s *mockStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)            { return nil, nil }
func (s *mockStore) SaveBlock(*types.Block, *bc.TransactionStatus) error          { return nil }
func (s *mockStore) SaveBlockHeader(*types.BlockHeader) error                     { return nil }
func (s *mockStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.VoteResult) error {
	return nil
}

type mStore struct {
	blockHeaders map[bc.Hash]*types.BlockHeader
}

func (s *mStore) BlockExist(hash *bc.Hash) bool           { return false }
func (s *mStore) GetBlock(*bc.Hash) (*types.Block, error) { return nil, nil }
func (s *mStore) GetBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	return s.blockHeaders[*hash], nil
}
func (s *mStore) GetStoreStatus() *protocol.BlockStoreState                    { return nil }
func (s *mStore) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) { return nil, nil }
func (s *mStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error     { return nil }
func (s *mStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)                 { return nil, nil }
func (s *mStore) GetVoteResult(uint64) (*state.VoteResult, error)              { return nil, nil }
func (s *mStore) GetMainChainHash(uint64) (*bc.Hash, error)                    { return nil, nil }
func (s *mStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)            { return nil, nil }
func (s *mStore) SaveBlock(*types.Block, *bc.TransactionStatus) error          { return nil }
func (s *mStore) SaveBlockHeader(blockHeader *types.BlockHeader) error {
	s.blockHeaders[blockHeader.Hash()] = blockHeader
	return nil
}
func (s *mStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.VoteResult) error {
	return nil
}

//---------------------

const (
	utxoPrefix  byte = iota //UTXOPrefix is StandardUTXOKey prefix
	sutxoPrefix             //SUTXOPrefix is ContractUTXOKey prefix
	contractPrefix
	contractIndexPrefix
	accountPrefix // AccountPrefix is account ID prefix
	accountAliasPrefix
	accountIndexPrefix
	txPrefix            //TxPrefix is wallet database transactions prefix
	txIndexPrefix       //TxIndexPrefix is wallet database tx index prefix
	unconfirmedTxPrefix //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	globalTxIndexPrefix //GlobalTxIndexPrefix is wallet database global tx index prefix
	walletKey
	miningAddressKey
	coinbaseAbKey
	recoveryKey
)

// leveldb key prefix
var (
	colon       byte = 0x3a
	UTXOPrefix       = []byte{utxoPrefix, colon}
	SUTXOPrefix      = []byte{sutxoPrefix, colon}
	// ContractPrefix = []byte{contractPrefix, contractPrefix, colon}
	ContractPrefix      = "Contract:"
	ContractIndexPrefix = []byte{contractIndexPrefix, colon}
	AccountPrefix       = []byte{accountPrefix, colon} // AccountPrefix is account ID prefix
	AccountAliasPrefix  = []byte{accountAliasPrefix, colon}
	AccountIndexPrefix  = []byte{accountIndexPrefix, colon}
	TxPrefix            = []byte{txPrefix, colon}            //TxPrefix is wallet database transactions prefix
	TxIndexPrefix       = []byte{txIndexPrefix, colon}       //TxIndexPrefix is wallet database tx index prefix
	UnconfirmedTxPrefix = []byte{unconfirmedTxPrefix, colon} //UnconfirmedTxPrefix is txpool unconfirmed transactions prefix
	GlobalTxIndexPrefix = []byte{globalTxIndexPrefix, colon} //GlobalTxIndexPrefix is wallet database global tx index prefix
	WalletKey           = []byte{walletKey}
	MiningAddressKey    = []byte{miningAddressKey}
	CoinbaseAbKey       = []byte{coinbaseAbKey}
	RecoveryKey         = []byte{recoveryKey}
)

// errors
var (
	// ErrFindAccount        = errors.New("Failed to find account")
	errAccntTxIDNotFound = errors.New("account TXID not found")
	errGetAsset          = errors.New("Failed to find asset definition")
)

func accountIndexKey(xpubs []chainkd.XPub) []byte {
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
func ContractKey(hash bc.Hash) []byte {
	return append([]byte(ContractPrefix), hash.Bytes()...)
}

// AccountIDKey account id store prefix
func AccountIDKey(accountID string) []byte {
	return append([]byte(AccountPrefix), []byte(accountID)...)
}

// StandardUTXOKey makes an account unspent outputs key to store
func StandardUTXOKey(id bc.Hash) []byte {
	return append(UTXOPrefix, id.Bytes()...)
}

// ContractUTXOKey makes a smart contract unspent outputs key to store
func ContractUTXOKey(id bc.Hash) []byte {
	return append(SUTXOPrefix, id.Bytes()...)
}

func calcDeleteKey(blockHeight uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", TxPrefix, blockHeight))
}

func calcTxIndexKey(txID string) []byte {
	return append(TxIndexPrefix, []byte(txID)...)
}

func calcAnnotatedKey(formatKey string) []byte {
	return append(TxPrefix, []byte(formatKey)...)
}

func calcUnconfirmedTxKey(formatKey string) []byte {
	return append(UnconfirmedTxPrefix, []byte(formatKey)...)
}

func calcGlobalTxIndexKey(txID string) []byte {
	return append(GlobalTxIndexPrefix, []byte(txID)...)
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

func contractIndexKey(accountID string) []byte {
	return append([]byte(ContractIndexPrefix), []byte(accountID)...)
}

func accountAliasKey(name string) []byte {
	return append([]byte(AccountAliasPrefix), []byte(name)...)
}

// MockWalletStore store wallet using leveldb
type MockWalletStore struct {
	walletDB dbm.DB
	batch    dbm.Batch
}

// NewMockWalletStore create new MockWalletStore struct
func NewMockWalletStore(db dbm.DB) *MockWalletStore {
	return &MockWalletStore{
		walletDB: db,
		batch:    nil,
	}
}

// InitBatch initial batch
func (store *MockWalletStore) InitBatch() error {
	if store.batch != nil {
		return errors.New("MockWalletStore initail fail, store batch is not nil.")
	}
	store.batch = store.walletDB.NewBatch()
	return nil
}

// CommitBatch commit batch
func (store *MockWalletStore) CommitBatch() error {
	if store.batch == nil {
		return errors.New("MockWalletStore commit fail, store batch is nil.")
	}
	store.batch.Write()
	store.batch = nil
	return nil
}

// DeleteContractUTXO delete contract utxo by outputID
func (store *MockWalletStore) DeleteContractUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.walletDB.Delete(ContractUTXOKey(outputID))
	} else {
		store.batch.Delete(ContractUTXOKey(outputID))
	}
}

// DeleteRecoveryStatus delete recovery status
func (store *MockWalletStore) DeleteRecoveryStatus() {
	if store.batch == nil {
		store.walletDB.Delete(RecoveryKey)
	} else {
		store.batch.Delete(RecoveryKey)
	}
}

// DeleteTransactions delete transactions when orphan block rollback
func (store *MockWalletStore) DeleteTransactions(height uint64) {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	txIter := store.walletDB.IteratorPrefix(calcDeleteKey(height))
	defer txIter.Release()

	tmpTx := query.AnnotatedTx{}
	for txIter.Next() {
		if err := json.Unmarshal(txIter.Value(), &tmpTx); err == nil {
			batch.Delete(calcTxIndexKey(tmpTx.ID.String()))
		}
		batch.Delete(txIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteUnconfirmedTransaction delete unconfirmed tx by txID
func (store *MockWalletStore) DeleteUnconfirmedTransaction(txID string) {
	if store.batch == nil {
		store.walletDB.Delete(calcUnconfirmedTxKey(txID))
	} else {
		store.batch.Delete(calcUnconfirmedTxKey(txID))
	}
}

// DeleteWalletTransactions delete all txs in wallet
func (store *MockWalletStore) DeleteWalletTransactions() {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	txIter := store.walletDB.IteratorPrefix([]byte(TxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		batch.Delete(txIter.Key())
	}

	txIndexIter := store.walletDB.IteratorPrefix([]byte(TxIndexPrefix))
	defer txIndexIter.Release()

	for txIndexIter.Next() {
		batch.Delete(txIndexIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// DeleteWalletUTXOs delete all txs in wallet
func (store *MockWalletStore) DeleteWalletUTXOs() {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}
	ruIter := store.walletDB.IteratorPrefix([]byte(UTXOPrefix))
	defer ruIter.Release()
	for ruIter.Next() {
		batch.Delete(ruIter.Key())
	}

	suIter := store.walletDB.IteratorPrefix([]byte(SUTXOPrefix))
	defer suIter.Release()
	for suIter.Next() {
		batch.Delete(suIter.Key())
	}
	if store.batch == nil {
		batch.Write()
	}
}

// GetAsset get asset by assetID
func (store *MockWalletStore) GetAsset(assetID *bc.AssetID) (*asset.Asset, error) {
	definitionByte := store.walletDB.Get(asset.ExtAssetKey(assetID))
	if definitionByte == nil {
		return nil, errGetAsset
	}
	definitionMap := make(map[string]interface{})
	if err := json.Unmarshal(definitionByte, &definitionMap); err != nil {
		return nil, err
	}
	alias := assetID.String()
	externalAsset := &asset.Asset{
		AssetID:           *assetID,
		Alias:             &alias,
		DefinitionMap:     definitionMap,
		RawDefinitionByte: definitionByte,
	}
	return externalAsset, nil
}

// GetControlProgram get raw program by hash
func (store *MockWalletStore) GetControlProgram(hash bc.Hash) (*acc.CtrlProgram, error) {
	rawProgram := store.walletDB.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, acc.ErrFindCtrlProgram
	}
	accountCP := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawProgram, &accountCP); err != nil {
		return nil, err
	}
	return accountCP, nil
}

// GetGlobalTransactionIndex get global tx by txID
func (store *MockWalletStore) GetGlobalTransactionIndex(txID string) []byte {
	return store.walletDB.Get(calcGlobalTxIndexKey(txID))
}

// GetStandardUTXO get standard utxo by id
func (store *MockWalletStore) GetStandardUTXO(outid bc.Hash) (*acc.UTXO, error) {
	rawUTXO := store.walletDB.Get(StandardUTXOKey(outid))
	if rawUTXO == nil {
		return nil, fmt.Errorf("failed get standard UTXO, outputID: %s ", outid.String())
	}
	UTXO := new(acc.UTXO)
	if err := json.Unmarshal(rawUTXO, UTXO); err != nil {
		return nil, err
	}
	return UTXO, nil
}

// GetTransaction get tx by txid
func (store *MockWalletStore) GetTransaction(txID string) (*query.AnnotatedTx, error) {
	formatKey := store.walletDB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, errAccntTxIDNotFound
	}
	rawTx := store.walletDB.Get(calcAnnotatedKey(string(formatKey)))
	tx := new(query.AnnotatedTx)
	if err := json.Unmarshal(rawTx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetUnconfirmedTransaction get unconfirmed tx by txID
func (store *MockWalletStore) GetUnconfirmedTransaction(txID string) (*query.AnnotatedTx, error) {
	rawUnconfirmedTx := store.walletDB.Get(calcUnconfirmedTxKey(txID))
	if rawUnconfirmedTx == nil {
		return nil, fmt.Errorf("failed get unconfirmed tx, txID: %s ", txID)
	}
	tx := new(query.AnnotatedTx)
	if err := json.Unmarshal(rawUnconfirmedTx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetRecoveryStatus delete recovery status
func (store *MockWalletStore) GetRecoveryStatus() (*RecoveryState, error) {
	rawStatus := store.walletDB.Get(dbm.RecoveryKey)
	if rawStatus == nil {
		return nil, ErrGetRecoveryStatus
	}
	state := new(RecoveryState)
	if err := json.Unmarshal(rawStatus, state); err != nil {
		return nil, err
	}
	return state, nil
}

// GetWalletInfo get wallet information
func (store *MockWalletStore) GetWalletInfo() (*StatusInfo, error) {
	rawStatus := store.walletDB.Get([]byte(dbm.WalletKey))
	if rawStatus == nil {
		return nil, fmt.Errorf("failed get wallet info")
	}
	status := new(StatusInfo)
	if err := json.Unmarshal(rawStatus, status); err != nil {
		return nil, err
	}
	return status, nil
}

// ListAccountUTXOs get all account unspent outputs
func (store *MockWalletStore) ListAccountUTXOs(key string) ([]*acc.UTXO, error) {
	accountUtxoIter := store.walletDB.IteratorPrefix([]byte(key))
	defer accountUtxoIter.Release()

	confirmedUTXOs := []*acc.UTXO{}
	for accountUtxoIter.Next() {
		utxo := new(acc.UTXO)
		if err := json.Unmarshal(accountUtxoIter.Value(), utxo); err != nil {
			return nil, err
		}
		confirmedUTXOs = append(confirmedUTXOs, utxo)
	}
	return confirmedUTXOs, nil
}

func (store *MockWalletStore) ListTransactions(accountID string, StartTxID string, count uint, unconfirmed bool) ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	var startKey []byte
	preFix := TxPrefix

	if StartTxID != "" {
		if unconfirmed {
			startKey = calcUnconfirmedTxKey(StartTxID)
		} else {
			formatKey := store.walletDB.Get(calcTxIndexKey(StartTxID))
			if formatKey == nil {
				return nil, errAccntTxIDNotFound
			}
			startKey = calcAnnotatedKey(string(formatKey))
		}
	}

	if unconfirmed {
		preFix = UnconfirmedTxPrefix
	}

	itr := store.walletDB.IteratorPrefixWithStart([]byte(preFix), startKey, true)
	defer itr.Release()

	for txNum := count; itr.Next() && txNum > 0; txNum-- {
		annotatedTx := new(query.AnnotatedTx)
		if err := json.Unmarshal(itr.Value(), &annotatedTx); err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, annotatedTx)
	}

	return annotatedTxs, nil
}

// ListUnconfirmedTransactions get all unconfirmed txs
func (store *MockWalletStore) ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error) {
	annotatedTxs := []*query.AnnotatedTx{}
	txIter := store.walletDB.IteratorPrefix([]byte(UnconfirmedTxPrefix))
	defer txIter.Release()

	for txIter.Next() {
		annotatedTx := &query.AnnotatedTx{}
		if err := json.Unmarshal(txIter.Value(), &annotatedTx); err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, annotatedTx)
	}
	return annotatedTxs, nil
}

// SetAssetDefinition set assetID and definition
func (store *MockWalletStore) SetAssetDefinition(assetID *bc.AssetID, definition []byte) {
	if store.batch == nil {
		store.walletDB.Set(asset.ExtAssetKey(assetID), definition)
	} else {
		store.batch.Set(asset.ExtAssetKey(assetID), definition)
	}
}

// SetContractUTXO set standard utxo
func (store *MockWalletStore) SetContractUTXO(outputID bc.Hash, utxo *acc.UTXO) error {
	data, err := json.Marshal(utxo)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.walletDB.Set(ContractUTXOKey(outputID), data)
	} else {
		store.batch.Set(ContractUTXOKey(outputID), data)
	}
	return nil
}

// SetGlobalTransactionIndex set global tx index by blockhash and position
func (store *MockWalletStore) SetGlobalTransactionIndex(globalTxID string, blockHash *bc.Hash, position uint64) {
	if store.batch == nil {
		store.walletDB.Set(calcGlobalTxIndexKey(globalTxID), CalcGlobalTxIndex(blockHash, position))
	} else {
		store.batch.Set(calcGlobalTxIndexKey(globalTxID), CalcGlobalTxIndex(blockHash, position))
	}
}

// SetRecoveryStatus set recovery status
func (store *MockWalletStore) SetRecoveryStatus(recoveryState *RecoveryState) error {
	rawStatus, err := json.Marshal(recoveryState)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.walletDB.Set(dbm.RecoveryKey, rawStatus)
	} else {
		store.batch.Set(dbm.RecoveryKey, rawStatus)
	}
	return nil
}

// SetTransaction set raw transaction by block height and tx position
func (store *MockWalletStore) SetTransaction(height uint64, tx *query.AnnotatedTx) error {
	batch := store.walletDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	rawTx, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	batch.Set(calcAnnotatedKey(formatKey(height, tx.Position)), rawTx)
	batch.Set(calcTxIndexKey(tx.ID.String()), []byte(formatKey(height, tx.Position)))

	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// SetUnconfirmedTransaction set unconfirmed tx by txID
func (store *MockWalletStore) SetUnconfirmedTransaction(txID string, tx *query.AnnotatedTx) error {
	rawTx, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.walletDB.Set(calcUnconfirmedTxKey(txID), rawTx)
	} else {
		store.batch.Set(calcUnconfirmedTxKey(txID), rawTx)
	}
	return nil
}

// SetWalletInfo get wallet information
func (store *MockWalletStore) SetWalletInfo(status *StatusInfo) error {
	rawWallet, err := json.Marshal(status)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.walletDB.Set([]byte(dbm.WalletKey), rawWallet)
	} else {
		store.batch.Set([]byte(dbm.WalletKey), rawWallet)
	}
	return nil
}

//-------------

type MockAccountStore struct {
	accountDB dbm.DB
	batch     dbm.Batch
}

// NewAccountStore create new MockAccountStore.
func NewMockAccountStore(db dbm.DB) *MockAccountStore {
	return &MockAccountStore{
		accountDB: db,
		batch:     nil,
	}
}

// InitBatch initial batch
func (store *MockAccountStore) InitBatch() error {
	if store.batch != nil {
		return errors.New("MockAccountStore initail fail, store batch is not nil.")
	}
	store.batch = store.accountDB.NewBatch()
	return nil
}

// CommitBatch commit batch
func (store *MockAccountStore) CommitBatch() error {
	if store.batch == nil {
		return errors.New("MockAccountStore commit fail, store batch is nil.")
	}
	store.batch.Write()
	store.batch = nil
	return nil
}

// DeleteAccount set account account ID, account alias and raw account.
func (store *MockAccountStore) DeleteAccount(account *acc.Account) error {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	// delete account utxos
	store.deleteAccountUTXOs(account.ID)

	// delete account control program
	cps, err := store.ListControlPrograms()
	if err != nil {
		return err
	}
	var hash [32]byte
	for _, cp := range cps {
		if cp.AccountID == account.ID {
			sha3pool.Sum256(hash[:], cp.ControlProgram)
			batch.Delete(ContractKey(bc.NewHash(hash)))
		}
	}

	// delete bip44 contract index
	batch.Delete(Bip44ContractIndexKey(account.ID, false))
	batch.Delete(Bip44ContractIndexKey(account.ID, true))

	// delete contract index
	batch.Delete(contractIndexKey(account.ID))

	// delete account id
	batch.Delete(AccountIDKey(account.ID))
	batch.Delete(accountAliasKey(account.Alias))
	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// deleteAccountUTXOs delete account utxos by accountID
func (store *MockAccountStore) deleteAccountUTXOs(accountID string) error {
	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	accountUtxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &acc.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			return err
		}
		if accountID == accountUtxo.AccountID {
			batch.Delete(StandardUTXOKey(accountUtxo.OutputID))
		}
	}

	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// DeleteStandardUTXO delete utxo by outpu id
func (store *MockAccountStore) DeleteStandardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// GetAccountByAlias get account by account alias
func (store *MockAccountStore) GetAccountByAlias(accountAlias string) (*acc.Account, error) {
	accountID := store.accountDB.Get(accountAliasKey(accountAlias))
	if accountID == nil {
		return nil, acc.ErrFindAccount
	}
	return store.GetAccountByID(string(accountID))
}

// GetAccountByID get account by accountID
func (store *MockAccountStore) GetAccountByID(accountID string) (*acc.Account, error) {
	rawAccount := store.accountDB.Get(AccountIDKey(accountID))
	if rawAccount == nil {
		return nil, acc.ErrFindAccount
	}
	account := new(acc.Account)
	if err := json.Unmarshal(rawAccount, account); err != nil {
		return nil, err
	}
	return account, nil
}

// GetAccountIndex get account index by account xpubs
func (store *MockAccountStore) GetAccountIndex(xpubs []chainkd.XPub) uint64 {
	currentIndex := uint64(0)
	if rawIndexBytes := store.accountDB.Get(accountIndexKey(xpubs)); rawIndexBytes != nil {
		currentIndex = common.BytesToUnit64(rawIndexBytes)
	}
	return currentIndex
}

// GetBip44ContractIndex get bip44 contract index
func (store *MockAccountStore) GetBip44ContractIndex(accountID string, change bool) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.accountDB.Get(Bip44ContractIndexKey(accountID, change)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetCoinbaseArbitrary get coinbase arbitrary
func (store *MockAccountStore) GetCoinbaseArbitrary() []byte {
	return store.accountDB.Get([]byte(CoinbaseAbKey))
}

// GetContractIndex get contract index
func (store *MockAccountStore) GetContractIndex(accountID string) uint64 {
	index := uint64(0)
	if rawIndexBytes := store.accountDB.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetControlProgram get control program
func (store *MockAccountStore) GetControlProgram(hash bc.Hash) (*acc.CtrlProgram, error) {
	rawProgram := store.accountDB.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, acc.ErrFindCtrlProgram
	}
	cp := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawProgram, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

// GetMiningAddress get mining address
func (store *MockAccountStore) GetMiningAddress() (*acc.CtrlProgram, error) {
	rawCP := store.accountDB.Get([]byte(MiningAddressKey))
	if rawCP == nil {
		return nil, acc.ErrFindMiningAddress
	}
	cp := new(acc.CtrlProgram)
	if err := json.Unmarshal(rawCP, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

// GetUTXO get standard utxo by id
func (store *MockAccountStore) GetUTXO(outid bc.Hash) (*acc.UTXO, error) {
	u := new(acc.UTXO)
	if data := store.accountDB.Get(StandardUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	if data := store.accountDB.Get(ContractUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	return nil, acc.ErrMatchUTXO
}

// ListAccounts get all accounts which name prfix is id.
func (store *MockAccountStore) ListAccounts(id string) ([]*acc.Account, error) {
	accounts := []*acc.Account{}
	accountIter := store.accountDB.IteratorPrefix(AccountIDKey(strings.TrimSpace(id)))
	defer accountIter.Release()

	for accountIter.Next() {
		account := new(acc.Account)
		if err := json.Unmarshal(accountIter.Value(), &account); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// ListControlPrograms get all local control programs
func (store *MockAccountStore) ListControlPrograms() ([]*acc.CtrlProgram, error) {
	cps := []*acc.CtrlProgram{}
	cpIter := store.accountDB.IteratorPrefix([]byte(ContractPrefix))
	// cpIter := store.accountDB.IteratorPrefix([]byte{0x02, 0x3a})
	defer cpIter.Release()

	for cpIter.Next() {
		cp := new(acc.CtrlProgram)
		if err := json.Unmarshal(cpIter.Value(), cp); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}
	return cps, nil
}

// ListUTXOs get utxos by accountID
func (store *MockAccountStore) ListUTXOs() ([]*acc.UTXO, error) {
	utxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer utxoIter.Release()

	utxos := []*acc.UTXO{}
	for utxoIter.Next() {
		utxo := new(acc.UTXO)
		if err := json.Unmarshal(utxoIter.Value(), utxo); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// SetAccount set account account ID, account alias and raw account.
func (store *MockAccountStore) SetAccount(account *acc.Account) error {
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return acc.ErrMarshalAccount
	}

	batch := store.accountDB.NewBatch()
	if store.batch != nil {
		batch = store.batch
	}

	batch.Set(AccountIDKey(account.ID), rawAccount)
	batch.Set(accountAliasKey(account.Alias), []byte(account.ID))

	if store.batch == nil {
		batch.Write()
	}
	return nil
}

// SetAccountIndex update account index
func (store *MockAccountStore) SetAccountIndex(account *acc.Account) {
	currentIndex := store.GetAccountIndex(account.XPubs)
	if account.KeyIndex > currentIndex {
		if store.batch == nil {
			store.accountDB.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))
		} else {
			store.batch.Set(accountIndexKey(account.XPubs), common.Unit64ToBytes(account.KeyIndex))
		}
	}
}

// SetBip44ContractIndex set contract index
func (store *MockAccountStore) SetBip44ContractIndex(accountID string, change bool, index uint64) {
	if store.batch == nil {
		store.accountDB.Set(Bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(Bip44ContractIndexKey(accountID, change), common.Unit64ToBytes(index))
	}
}

// SetCoinbaseArbitrary set coinbase arbitrary
func (store *MockAccountStore) SetCoinbaseArbitrary(arbitrary []byte) {
	if store.batch == nil {
		store.accountDB.Set([]byte(CoinbaseAbKey), arbitrary)
	} else {
		store.batch.Set([]byte(CoinbaseAbKey), arbitrary)
	}
}

// SetContractIndex set contract index
func (store *MockAccountStore) SetContractIndex(accountID string, index uint64) {
	if store.batch == nil {
		store.accountDB.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
	} else {
		store.batch.Set(contractIndexKey(accountID), common.Unit64ToBytes(index))
	}
}

// SetControlProgram set raw program
func (store *MockAccountStore) SetControlProgram(hash bc.Hash, program *acc.CtrlProgram) error {
	accountCP, err := json.Marshal(program)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.accountDB.Set(ContractKey(hash), accountCP)
	} else {
		store.batch.Set(ContractKey(hash), accountCP)
	}
	return nil
}

// SetMiningAddress set mining address
func (store *MockAccountStore) SetMiningAddress(program *acc.CtrlProgram) error {
	rawProgram, err := json.Marshal(program)
	if err != nil {
		return err
	}

	if store.batch == nil {
		store.accountDB.Set([]byte(MiningAddressKey), rawProgram)
	} else {
		store.batch.Set([]byte(MiningAddressKey), rawProgram)
	}
	return nil
}

// SetStandardUTXO set standard utxo
func (store *MockAccountStore) SetStandardUTXO(outputID bc.Hash, utxo *acc.UTXO) error {
	data, err := json.Marshal(utxo)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.accountDB.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
	return nil
}
