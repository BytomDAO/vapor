package wallet

import (
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/vapor/account"
	acc "github.com/vapor/account"
	"github.com/vapor/asset"
	"github.com/vapor/blockchain/pseudohsm"
	"github.com/vapor/blockchain/query"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/database"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/event"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/test/mock"
)

func TestEncodeDecodeGlobalTxIndex(t *testing.T) {
	want := &struct {
		BlockHash bc.Hash
		Position  uint64
	}{
		BlockHash: bc.NewHash([32]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}),
		Position:  1,
	}

	globalTxIdx := mock.CalcGlobalTxIndex(&want.BlockHash, want.Position)
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
	walletStore := mock.NewMockWalletStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	dispatcher := event.NewDispatcher()
	w := mockWallet(walletStore, nil, nil, nil, dispatcher, false)

	// legacy status test case
	type legacyStatusInfo struct {
		WorkHeight uint64
		WorkHash   bc.Hash
		BestHeight uint64
		BestHash   bc.Hash
	}
	rawWallet, err := json.Marshal(legacyStatusInfo{})
	if err != nil {
		t.Fatal("Marshal legacyStatusInfo")
	}

	w.store.SetWalletInfo(rawWallet)
	rawWallet = w.store.GetWalletInfo()
	if rawWallet == nil {
		t.Fatal("fail to load wallet StatusInfo")
	}

	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
		t.Fatal(err)
	}

	if err := w.checkWalletInfo(); err != errWalletVersionMismatch {
		t.Fatal("fail to detect legacy wallet version")
	}

	// lower wallet version test case
	lowerVersion := StatusInfo{Version: currentVersion - 1}
	rawWallet, err = json.Marshal(lowerVersion)
	if err != nil {
		t.Fatal("save wallet info")
	}

	w.store.SetWalletInfo(rawWallet)
	rawWallet = w.store.GetWalletInfo()
	if rawWallet == nil {
		t.Fatal("fail to load wallet StatusInfo")
	}

	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
		t.Fatal(err)
	}

	if err := w.checkWalletInfo(); err != errWalletVersionMismatch {
		t.Fatal("fail to detect expired wallet version")
	}
}

func TestWalletUpdate(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	config.CommonConfig = config.DefaultConfig()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := database.NewStore(testDB)
	walletStore := mock.NewMockWalletStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)

	chain, err := protocol.NewChain(store, txPool, dispatcher)
	if err != nil {
		t.Fatal(err)
	}

	accountStore := mock.NewMockAccountStore(testDB)
	accountManager := account.NewManager(accountStore, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	controlProg.KeyIndex = 1

	reg := asset.NewRegistry(testDB, chain)
	// reg := asset.NewRegistry(testDB, nil)
	asset := bc.AssetID{V0: 5}

	utxos := []*account.UTXO{}
	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)
	utxos = append(utxos, btmUtxo)
	OtherUtxo := mockUTXO(controlProg, &asset)
	utxos = append(utxos, OtherUtxo)

	_, txData, err := mockTxData(utxos, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)
	block := mockSingleBlock(tx)
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatus.SetStatus(1, false)
	store.SaveBlock(block, txStatus)

	w := mockWallet(walletStore, accountManager, reg, chain, dispatcher, true)
	err = w.AttachBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.GetTransactionByTxID(tx.ID.String()); err != nil {
		t.Fatal(err)
	}

	wants, err := w.GetTransactions(testAccount.ID, "", 1, false)
	if len(wants) != 1 {
		t.Fatal(err)
	}

	if wants[0].ID != tx.ID {
		t.Fatal("account txID mismatch")
	}

	for position, tx := range block.Transactions {
		get := w.store.GetGlobalTransactionIndex(tx.ID.String())
		bh := block.BlockHeader.Hash()
		expect := CalcGlobalTxIndex(&bh, uint64(position))
		if !reflect.DeepEqual(get, expect) {
			t.Fatalf("position#%d: compare retrieved globalTxIdx err", position)
		}
	}
}

func TestRescanWallet(t *testing.T) {
	// prepare wallet & db
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	config.CommonConfig = config.DefaultConfig()
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	walletStore := mock.NewMockWalletStore(testDB)
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	store := database.NewStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)
	chain, err := protocol.NewChain(store, txPool, dispatcher)
	if err != nil {
		t.Fatal(err)
	}

	statusInfo := StatusInfo{
		Version:  currentVersion,
		WorkHash: bc.Hash{V0: 0xff},
	}
	rawWallet, err := json.Marshal(statusInfo)
	if err != nil {
		t.Fatal("save wallet info")
	}

	w := mockWallet(walletStore, nil, nil, chain, dispatcher, false)
	w.store.SetWalletInfo(rawWallet)
	rawWallet = w.store.GetWalletInfo()
	if rawWallet == nil {
		t.Fatal("fail to load wallet StatusInfo")
	}

	if err := json.Unmarshal(rawWallet, &w.status); err != nil {
		t.Fatal(err)
	}

	// rescan wallet
	if err := w.loadWalletInfo(); err != nil {
		t.Fatal(err)
	}

	block := config.GenesisBlock()
	if w.status.WorkHash != block.Hash() {
		t.Fatal("reattach from genesis block")
	}
}

func TestMemPoolTxQueryLoop(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	config.CommonConfig = config.DefaultConfig()
	testDB := dbm.NewDB("testdb", "leveldb", dirPath)
	defer func() {
		testDB.Close()
		os.RemoveAll(dirPath)
	}()

	store := database.NewStore(testDB)
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)

	chain, err := protocol.NewChain(store, txPool, dispatcher)
	if err != nil {
		t.Fatal(err)
	}

	accountStore := mock.NewMockAccountStore(testDB)
	accountManager := account.NewManager(accountStore, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, _, err := hsm.XCreate("test_pub1", "password", "en")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount", signers.BIP0044)
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	controlProg.KeyIndex = 1

	reg := asset.NewRegistry(testDB, chain)
	asset := bc.AssetID{V0: 5}

	utxos := []*account.UTXO{}
	btmUtxo := mockUTXO(controlProg, consensus.BTMAssetID)
	utxos = append(utxos, btmUtxo)
	OtherUtxo := mockUTXO(controlProg, &asset)
	utxos = append(utxos, OtherUtxo)

	_, txData, err := mockTxData(utxos, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)
	//block := mockSingleBlock(tx)
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	walletStore := mock.NewMockWalletStore(testDB)
	w, err := NewWallet(walletStore, accountManager, reg, hsm, chain, dispatcher, false)
	go w.memPoolTxQueryLoop()
	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: protocol.MsgNewTx}})
	time.Sleep(time.Millisecond * 10)
	if _, err := w.GetUnconfirmedTxByTxID(tx.ID.String()); err != nil {
		t.Fatal("dispatch new tx msg error:", err)
	}
	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: protocol.MsgRemoveTx}})
	time.Sleep(time.Millisecond * 10)
	txs, err := w.GetUnconfirmedTxs(testAccount.ID)
	if err != nil {
		t.Fatal("get unconfirmed tx error:", err)
	}

	if len(txs) != 0 {
		t.Fatal("dispatch remove tx msg error")
	}

	w.eventDispatcher.Post(protocol.TxMsgEvent{TxMsg: &protocol.TxPoolMsg{TxDesc: &protocol.TxDesc{Tx: tx}, MsgType: 2}})
}

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

var (
	WalletKey     = []byte{0x00, 0x3a}
	TxIndexPrefix = []byte{0x01, 0x3a}
	TxPrefix      = []byte{0x02, 0x3a}
)

func CalcGlobalTxIndex(blockHash *bc.Hash, position uint64) []byte {
	txIdx := make([]byte, 40)
	copy(txIdx[:32], blockHash.Bytes())
	binary.BigEndian.PutUint64(txIdx[32:], position)
	return txIdx
}

func calcTxIndexKey(txID string) []byte {
	return append(TxIndexPrefix, []byte(txID)...)
}

func calcAnnotatedKey(formatKey string) []byte {
	return append(TxPrefix, []byte(formatKey)...)
}

type mockAccountStore struct {
	accountDB dbm.DB
	batch     dbm.Batch
}

// NewAccountStore create new AccountStore.
func newMockAccountStore(db dbm.DB) *mockAccountStore {
	return &mockAccountStore{
		accountDB: db,
		batch:     nil,
	}
}

func (store *mockAccountStore) InitBatch() error                                   { return nil }
func (store *mockAccountStore) CommitBatch() error                                 { return nil }
func (store *mockAccountStore) DeleteAccount(*account.Account) error               { return nil }
func (store *mockAccountStore) DeleteStandardUTXO(outputID bc.Hash)                { return }
func (store *mockAccountStore) GetAccountByAlias(string) (*account.Account, error) { return nil, nil }
func (store *mockAccountStore) GetAccountByID(string) (*account.Account, error)    { return nil, nil }
func (store *mockAccountStore) GetAccountIndex([]chainkd.XPub) uint64              { return 0 }
func (store *mockAccountStore) GetBip44ContractIndex(string, bool) uint64          { return 0 }
func (store *mockAccountStore) GetCoinbaseArbitrary() []byte                       { return nil }
func (store *mockAccountStore) GetContractIndex(string) uint64                     { return 0 }
func (store *mockAccountStore) GetControlProgram(bc.Hash) (*account.CtrlProgram, error) {
	return nil, nil
}
func (store *mockAccountStore) GetUTXO(outid bc.Hash) (*account.UTXO, error)               { return nil, nil }
func (store *mockAccountStore) GetMiningAddress() (*account.CtrlProgram, error)            { return nil, nil }
func (store *mockAccountStore) ListAccounts(string) ([]*account.Account, error)            { return nil, nil }
func (store *mockAccountStore) ListControlPrograms() ([]*account.CtrlProgram, error)       { return nil, nil }
func (store *mockAccountStore) ListUTXOs() ([]*account.UTXO, error)                        { return nil, nil }
func (store *mockAccountStore) SetAccount(*account.Account) error                          { return nil }
func (store *mockAccountStore) SetAccountIndex(*account.Account)                           { return }
func (store *mockAccountStore) SetBip44ContractIndex(string, bool, uint64)                 { return }
func (store *mockAccountStore) SetCoinbaseArbitrary([]byte)                                { return }
func (store *mockAccountStore) SetContractIndex(string, uint64)                            { return }
func (store *mockAccountStore) SetControlProgram(bc.Hash, *account.CtrlProgram) error      { return nil }
func (store *mockAccountStore) SetMiningAddress(*account.CtrlProgram) error                { return nil }
func (store *mockAccountStore) SetStandardUTXO(outputID bc.Hash, utxo *account.UTXO) error { return nil }

// WalletStore store wallet using leveldb
type mockWalletStore struct {
	walletDB dbm.DB
	batch    dbm.Batch
}

// NewWalletStore create new WalletStore struct
func newMockWalletStore(db dbm.DB) *mockWalletStore {
	return &mockWalletStore{
		walletDB: db,
		batch:    nil,
	}
}

func (store *mockWalletStore) InitBatch() error                                    { return nil }
func (store *mockWalletStore) CommitBatch() error                                  { return nil }
func (store *mockWalletStore) DeleteContractUTXO(bc.Hash)                          { return }
func (store *mockWalletStore) DeleteRecoveryStatus()                               { return }
func (store *mockWalletStore) DeleteTransactions(uint64)                           { return }
func (store *mockWalletStore) DeleteUnconfirmedTransaction(string)                 { return }
func (store *mockWalletStore) DeleteWalletTransactions()                           { return }
func (store *mockWalletStore) DeleteWalletUTXOs()                                  { return }
func (store *mockWalletStore) GetAsset(*bc.AssetID) (*asset.Asset, error)          { return nil, nil }
func (store *mockWalletStore) GetControlProgram(bc.Hash) (*acc.CtrlProgram, error) { return nil, nil }
func (store *mockWalletStore) GetGlobalTransactionIndex(string) []byte             { return nil }
func (store *mockWalletStore) GetStandardUTXO(bc.Hash) (*acc.UTXO, error)          { return nil, nil }

// func (store *mockWalletStore) GetTransaction(string) (*query.AnnotatedTx, error)   { return nil, nil }
func (store *mockWalletStore) GetUnconfirmedTransaction(string) (*query.AnnotatedTx, error) {
	return nil, nil
}

// func (store *mockWalletStore) GetRecoveryStatus([]byte) []byte              { return nil }
func (store *mockWalletStore) ListAccountUTXOs(string) ([]*acc.UTXO, error) { return nil, nil }
func (store *mockWalletStore) ListTransactions(string, string, uint, bool) ([]*query.AnnotatedTx, error) {
	return nil, nil
}
func (store *mockWalletStore) ListUnconfirmedTransactions() ([]*query.AnnotatedTx, error) {
	return nil, nil
}
func (store *mockWalletStore) SetAssetDefinition(*bc.AssetID, []byte)             { return }
func (store *mockWalletStore) SetContractUTXO(bc.Hash, *acc.UTXO) error           { return nil }
func (store *mockWalletStore) SetGlobalTransactionIndex(string, *bc.Hash, uint64) { return }

// func (store *mockWalletStore) SetRecoveryStatus([]byte, []byte)                   { return }
func (store *mockWalletStore) SetTransaction(uint64, *query.AnnotatedTx) error { return nil }
func (store *mockWalletStore) SetUnconfirmedTransaction(string, *query.AnnotatedTx) error {
	return nil
}

// GetRecoveryStatus delete recovery status
func (store *mockWalletStore) GetRecoveryStatus(recoveryKey []byte) []byte {
	return store.walletDB.Get(recoveryKey)
}

// GetTransaction get tx by txid
func (store *mockWalletStore) GetTransaction(txID string) (*query.AnnotatedTx, error) {
	formatKey := store.walletDB.Get(calcTxIndexKey(txID))
	if formatKey == nil {
		return nil, errors.New("account TXID not found")
	}
	rawTx := store.walletDB.Get(calcAnnotatedKey(string(formatKey)))
	tx := new(query.AnnotatedTx)
	if err := json.Unmarshal(rawTx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// GetWalletInfo get wallet information
func (store *mockWalletStore) GetWalletInfo() []byte {
	return store.walletDB.Get([]byte(WalletKey))
}

// SetRecoveryStatus set recovery status
func (store *mockWalletStore) SetRecoveryStatus(recoveryKey, rawStatus []byte) {
	if store.batch == nil {
		store.walletDB.Set(recoveryKey, rawStatus)
	} else {
		store.batch.Set(recoveryKey, rawStatus)
	}
}

// SetWalletInfo get wallet information
func (store *mockWalletStore) SetWalletInfo(rawWallet []byte) {
	if store.batch == nil {
		store.walletDB.Set([]byte(WalletKey), rawWallet)
	} else {
		store.batch.Set([]byte(WalletKey), rawWallet)
	}
}

type mockStore struct {
	db dbm.DB
}

// newStore creates and returns a new Store object.
func newStore(db dbm.DB) *mockStore {
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
