package claim

import (
	"bytes"
	"encoding/json"
	"strconv"

	bytomtypes "github.com/vapor/claim/bytom/protocolbc/types"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto"
	"github.com/vapor/equity/pegin_contract"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
	"github.com/vapor/util"
)

type MerkleBlock struct {
	BlockHeader  []byte     `json:"block_header"`
	TxHashes     []*bc.Hash `json:"tx_hashes"`
	StatusHashes []*bc.Hash `json:"status_hashes"`
	Flags        []uint32   `json:"flags"`
	MatchedTxIDs []*bc.Hash `json:"matched_tx_ids"`
}

type BytomClaimValidation struct {
}

func (b *BytomClaimValidation) IsValidPeginWitness(peginWitness [][]byte, prevout bc.Output) (err error) {

	assetAmount := &bc.AssetAmount{
		AssetId: prevout.Source.Value.AssetId,
		Amount:  prevout.Source.Value.Amount,
	}

	src := &bc.ValueSource{
		Ref:      prevout.Source.Ref,
		Value:    assetAmount,
		Position: prevout.Source.Position,
	}
	prog := &bc.Program{prevout.ControlProgram.VmVersion, prevout.ControlProgram.Code}
	bytomPrevout := bc.NewOutput(src, prog, prevout.Source.Position)

	if len(peginWitness) != 5 {
		return errors.New("peginWitness is error")
	}
	amount, err := strconv.ParseUint(string(peginWitness[0]), 10, 64)
	if err != nil {
		return err
	}
	if !consensus.MoneyRange(amount) {
		return errors.New("Amount out of range")
	}
	/*
		if len(peginWitness[1]) != 32 {
			return errors.New("The length of gennesisBlockHash is not correct")
		}
	*/
	claimScript := peginWitness[2]

	rawTx := &bytomtypes.Tx{}
	err = rawTx.UnmarshalText(peginWitness[3])
	if err != nil {
		return err
	}

	merkleBlock := &MerkleBlock{}
	err = json.Unmarshal(peginWitness[4], merkleBlock)
	if err != nil {
		return err
	}
	// proof验证
	var flags []uint8
	for flag := range merkleBlock.Flags {
		flags = append(flags, uint8(flag))
	}
	blockHeader := &bytomtypes.BlockHeader{}
	if err = blockHeader.UnmarshalText(merkleBlock.BlockHeader); err != nil {
		return err
	}

	if !types.ValidateTxMerkleTreeProof(merkleBlock.TxHashes, flags, merkleBlock.MatchedTxIDs, blockHeader.BlockCommitment.TransactionsMerkleRoot) {
		return errors.New("Merkleblock validation failed")
	}

	// 交易进行验证
	if err = b.checkPeginTx(rawTx, bytomPrevout, amount, claimScript); err != nil {
		return err
	}
	var hash bc.Hash
	hash.UnmarshalText(peginWitness[1])
	// Check the genesis block corresponds to a valid peg (only one for now)
	if hash.String() != consensus.ActiveNetParams.ParentGenesisBlockHash {
		return errors.New("ParentGenesisBlockHash don't match")
	}
	// TODO Finally, validate peg-in via rpc call

	if util.ValidatePegin {
		if err := util.IsConfirmedBytomBlock(blockHeader.Height, consensus.ActiveNetParams.PeginMinDepth); err != nil {
			return err
		}
	}

	return nil
}

func (b *BytomClaimValidation) checkPeginTx(rawTx *bytomtypes.Tx, prevout *bc.Output, claimAmount uint64, claimScript []byte) error {
	// Check the transaction nout/value matches
	amount := rawTx.Outputs[prevout.Source.Position].Amount
	if claimAmount != amount {
		return errors.New("transaction nout/value do not matches")
	}
	// Check that the witness program matches the p2ch on the p2sh-p2wsh transaction output
	//federationRedeemScript := vmutil.CalculateContract(consensus.ActiveNetParams.FedpegXPubs, claimScript)
	//scriptHash := crypto.Sha256(federationRedeemScript)
	peginContractPrograms, err := pegin_contract.GetPeginContractPrograms(claimScript)
	if err != nil {
		return err
	}

	scriptHash := crypto.Sha256(peginContractPrograms)
	controlProg, err := vmutil.P2WSHProgram(scriptHash)
	if err != nil {
		return err
	}
	if !bytes.Equal(rawTx.Outputs[prevout.Source.Position].ControlProgram, controlProg) {
		return errors.New("The output control program of transaction does not match the control program of the system's alliance contract")
	}
	return nil
}
