package config

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/crypto"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/vm/vmutil"
)

// FedAddressPath is used to derive federation root xpubs for signing cross-chain txs
var FedAddressPath = [][]byte{
	[]byte{0x2C, 0x00, 0x00, 0x00},
	[]byte{0x99, 0x00, 0x00, 0x00},
	[]byte{0x01, 0x00, 0x00, 0x00},
	[]byte{0x00, 0x00, 0x00, 0x00},
	[]byte{0x01, 0x00, 0x00, 0x00},
}

func FederationPMultiSigScript(c *Config) []byte {
	xpubs := c.Federation.Xpubs
	derivedXPubs := chainkd.DeriveXPubs(xpubs, FedAddressPath)
	program, err := vmutil.P2SPMultiSigProgram(chainkd.XPubKeys(derivedXPubs), c.Federation.Quorum)
	if err != nil {
		log.Panicf("fail to generate federation scirpt for federation: %v", err)
	}

	return program
}

func FederationWScript(c *Config) []byte {
	script := FederationPMultiSigScript(c)
	scriptHash := crypto.Sha256(script)
	wscript, err := vmutil.P2WSHProgram(scriptHash)
	if err != nil {
		log.Panicf("Fail converts scriptHash to witness: %v", err)
	}

	return wscript
}

func GenesisTx() *types.Tx {
	contract, err := hex.DecodeString("00148c9d063ff74ee6d9ffa88d83aeb038068366c4c4")
	if err != nil {
		log.Panicf("fail on decode genesis tx output control program")
	}

	coinbaseInput := FederationWScript(CommonConfig)

	txData := types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput(coinbaseInput[:]),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(*consensus.BTMAssetID, consensus.BlockSubsidy(0), contract),
		},
	}
	return types.NewTx(txData)
}

func mainNetGenesisBlock() *types.Block {
	tx := GenesisTx()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		log.Panicf(err.Error())
	}
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1563344560002,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}

func testNetGenesisBlock() *types.Block {
	tx := GenesisTx()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		log.Panicf(err.Error())
	}
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1563344560001,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}

func soloNetGenesisBlock() *types.Block {
	tx := GenesisTx()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		log.Panicf(err.Error())
	}
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1563344560000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}

// GenesisBlock will return genesis block
func GenesisBlock() *types.Block {
	return map[string]func() *types.Block{
		"main": mainNetGenesisBlock,
		"test": testNetGenesisBlock,
		"solo": soloNetGenesisBlock,
	}[consensus.ActiveNetParams.Name]()
}
