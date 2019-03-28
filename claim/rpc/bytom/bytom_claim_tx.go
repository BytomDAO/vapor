package bytom

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/claim"
	bytomtypes "github.com/vapor/claim/bytom/protocolbc/types"
	"github.com/vapor/claim/rpc"
	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/util"
	"github.com/vapor/wallet"
)

func getPeginTxnOutputIndex(rawTx bytomtypes.Tx, controlProg []byte) int {
	for index, output := range rawTx.Outputs {
		if bytes.Equal(output.ControlProgram, controlProg) {
			return index
		}
	}
	return -1
}

func toHash(hexBytes []chainjson.HexBytes) (hashs []*bc.Hash) {
	for _, data := range hexBytes {
		b32 := [32]byte{}
		copy(b32[:], data)
		res := bc.NewHash(b32)
		hashs = append(hashs, &res)
	}
	return
}

type BytomClaimTx struct {
	rpc.ClaimTxParam
	Wallet *wallet.Wallet
	Chain  *protocol.Chain
}

func (b *BytomClaimTx) pseudohsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
	return b.Wallet.Hsm.XSign(xpub, path, data[:], password)
}

func (b *BytomClaimTx) ClaimPeginTx(ctx context.Context) (interface{}, error) {
	tmpl, err := b.createRawPegin(b.ClaimTxParam)
	if err != nil {
		log.WithField("build err", err).Error("fail on createrawpegin.")
		return nil, err
	}

	// 交易签名
	if err := txbuilder.Sign(ctx, tmpl, b.ClaimTxParam.Password, b.pseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return nil, err
	}

	// submit
	if err := txbuilder.FinalizeTx(ctx, b.Chain, tmpl.Transaction); err != nil {
		return nil, err
	}

	log.WithField("tx_id", tmpl.Transaction.ID.String()).Info("claim script tx")
	return &struct {
		TxID *bc.Hash `json:"tx_id"`
	}{TxID: &tmpl.Transaction.ID}, nil
}

func (b *BytomClaimTx) createRawPegin(ins rpc.ClaimTxParam) (*txbuilder.Template, error) {
	// proof验证
	var flags []uint8
	for flag := range ins.Flags {
		flags = append(flags, uint8(flag))
	}
	txHashes := toHash(ins.TxHashes)
	matchedTxIDs := toHash(ins.MatchedTxIDs)
	statusHashes := toHash(ins.StatusHashes)
	blockHeader := &bytomtypes.BlockHeader{}
	if err := blockHeader.UnmarshalText([]byte(ins.BlockHeader)); err != nil {
		return nil, err
	}
	if !types.ValidateTxMerkleTreeProof(txHashes, flags, matchedTxIDs, blockHeader.BlockCommitment.TransactionsMerkleRoot) {
		return nil, errors.New("Merkleblock validation failed")
	}
	// CheckBytomProof
	//difficulty.CheckBytomProofOfWork(ins.BlockHeader.Hash(), ins.BlockHeader)
	// 增加spv验证以及连接主链api查询交易的确认数
	if util.ValidatePegin {
		if err := util.IsConfirmedBytomBlock(blockHeader.Height, consensus.ActiveNetParams.PeginMinDepth); err != nil {
			return nil, err
		}
	}
	// 找出与claim script有关联的交易的输出
	var claimScript []byte
	rawTx := &bytomtypes.Tx{}
	if err := rawTx.UnmarshalText([]byte(ins.RawTx)); err != nil {
		return nil, err
	}
	nOut := len(rawTx.Outputs)
	if ins.ClaimScript == nil {
		// 遍历寻找与交易输出有关的claim script
		cps, err := b.Wallet.AccountMgr.ListControlProgram()
		if err != nil {
			return nil, err
		}

		for _, cp := range cps {
			_, controlProg := b.Wallet.AccountMgr.GetPeginControlPrograms(cp.ControlProgram)
			if controlProg == nil {
				continue
			}
			// 获取交易的输出
			nOut = getPeginTxnOutputIndex(*rawTx, controlProg)
			if nOut != len(rawTx.Outputs) {
				claimScript = cp.ControlProgram
			}
		}
	} else {
		claimScript = ins.ClaimScript
		_, controlProg := b.Wallet.AccountMgr.GetPeginControlPrograms(claimScript)
		// 获取交易的输出
		nOut = getPeginTxnOutputIndex(*rawTx, controlProg)
	}
	if nOut == len(rawTx.Outputs) || nOut == -1 {
		return nil, errors.New("Failed to find output in bytom to the mainchain_address from getpeginaddress")
	}

	// 根据ClaimScript 获取account id
	var hash [32]byte
	sha3pool.Sum256(hash[:], claimScript)
	data := b.Wallet.DB.Get(account.ContractKey(hash))
	if data == nil {
		return nil, errors.New("Failed to find control program through claim script")
	}

	cp := &account.CtrlProgram{}
	if err := json.Unmarshal(data, cp); err != nil {
		return nil, errors.New("Failed on unmarshal control program")
	}

	// 构造交易
	// 用输出作为交易输入 生成新的交易
	builder := txbuilder.NewBuilder(time.Now())
	// TODO 根据raw tx生成一个utxo
	sourceID := *rawTx.OutputID(nOut)
	outputAccount := rawTx.Outputs[nOut].Amount
	assetID := *rawTx.Outputs[nOut].AssetId

	txInput := types.NewClaimInput(nil, sourceID, assetID, outputAccount, uint64(nOut), cp.ControlProgram)
	if err := builder.AddInput(txInput, &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	program, err := b.Wallet.AccountMgr.CreateAddress(cp.AccountID, false)
	if err != nil {
		return nil, err
	}

	if err = builder.AddOutput(types.NewTxOutput(assetID, outputAccount, program.ControlProgram)); err != nil {
		return nil, err
	}

	tmpl, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// todo把一些主链的信息加到交易的stack中
	var stack [][]byte

	//amount
	amount := strconv.FormatUint(rawTx.Outputs[nOut].Amount, 10)
	stack = append(stack, []byte(amount))
	// 主链的gennesisBlockHash
	stack = append(stack, []byte(consensus.ActiveNetParams.ParentGenesisBlockHash))
	// claim script
	stack = append(stack, claimScript)
	// raw tx
	tx, _ := json.Marshal(rawTx)
	stack = append(stack, tx)
	// proof
	blockHeaderByte, err := blockHeader.MarshalText()
	if err != nil {
		return nil, err
	}
	merkleBlock := claim.MerkleBlock{
		BlockHeader:  blockHeaderByte,
		TxHashes:     txHashes,
		StatusHashes: statusHashes,
		Flags:        ins.Flags,
		MatchedTxIDs: matchedTxIDs,
	}

	txOutProof, _ := json.Marshal(merkleBlock)

	stack = append(stack, txOutProof)

	//	tmpl.Transaction.Inputs[0].Peginwitness = stack
	txData.Inputs[0].Peginwitness = stack

	//交易费估算
	txGasResp, err := EstimateTxGas(*tmpl)
	if err != nil {
		return nil, err
	}
	txData.Outputs[0].Amount = txData.Outputs[0].Amount - uint64(txGasResp.TotalNeu)
	//重设置Transaction
	tmpl.Transaction = types.NewTx(*txData)
	return tmpl, nil
}

func (b *BytomClaimTx) ClaimContractPeginTx(ctx context.Context) (interface{}, error) {
	tmpl, err := b.createContractRawPegin(b.ClaimTxParam)
	if err != nil {
		log.WithField("build err", err).Error("fail on claimContractPeginTx.")
		return nil, err
	}
	// 交易签名
	if err := txbuilder.Sign(ctx, tmpl, b.Password, b.pseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return nil, err
	}

	// submit
	if err := txbuilder.FinalizeTx(ctx, b.Chain, tmpl.Transaction); err != nil {
		return nil, err
	}

	log.WithField("tx_id", tmpl.Transaction.ID.String()).Info("claim script tx")
	return &struct {
		TxID *bc.Hash `json:"tx_id"`
	}{TxID: &tmpl.Transaction.ID}, nil
}

func (b *BytomClaimTx) createContractRawPegin(ins rpc.ClaimTxParam) (*txbuilder.Template, error) {
	// proof验证
	var flags []uint8
	for flag := range ins.Flags {
		flags = append(flags, uint8(flag))
	}
	txHashes := toHash(ins.TxHashes)
	matchedTxIDs := toHash(ins.MatchedTxIDs)
	statusHashes := toHash(ins.StatusHashes)
	blockHeader := &bytomtypes.BlockHeader{}
	if err := blockHeader.UnmarshalText([]byte(ins.BlockHeader)); err != nil {
		return nil, err
	}

	if !types.ValidateTxMerkleTreeProof(txHashes, flags, matchedTxIDs, blockHeader.BlockCommitment.TransactionsMerkleRoot) {
		return nil, errors.New("Merkleblock validation failed")
	}
	// CheckBytomProof
	//difficulty.CheckBytomProofOfWork(ins.BlockHeader.Hash(), ins.BlockHeader)
	// 增加spv验证以及连接主链api查询交易的确认数
	if util.ValidatePegin {
		if err := util.IsConfirmedBytomBlock(blockHeader.Height, consensus.ActiveNetParams.PeginMinDepth); err != nil {
			return nil, err
		}
	}
	// 找出与claim script有关联的交易的输出
	var claimScript []byte
	rawTx := &bytomtypes.Tx{}
	if err := rawTx.UnmarshalText([]byte(ins.RawTx)); err != nil {
		return nil, err
	}

	nOut := len(rawTx.Outputs)
	if ins.ClaimScript == nil {
		// 遍历寻找与交易输出有关的claim script
		cps, err := b.Wallet.AccountMgr.ListControlProgram()
		if err != nil {
			return nil, err
		}

		for _, cp := range cps {
			_, controlProg := b.Wallet.AccountMgr.GetPeginContractControlPrograms(claimScript)
			// 获取交易的输出
			nOut = getPeginTxnOutputIndex(*rawTx, controlProg)
			if nOut != len(rawTx.Outputs) {
				claimScript = cp.ControlProgram
			}
		}
	} else {
		claimScript = ins.ClaimScript
		_, controlProg := b.Wallet.AccountMgr.GetPeginContractControlPrograms(claimScript)
		// 获取交易的输出
		nOut = getPeginTxnOutputIndex(*rawTx, controlProg)
	}
	if nOut == len(rawTx.Outputs) || nOut == -1 {
		return nil, errors.New("Failed to find output in bytom to the mainchain_address from createContractRawPegin")
	}

	// 根据ClaimScript 获取account id
	var hash [32]byte
	sha3pool.Sum256(hash[:], claimScript)
	data := b.Wallet.DB.Get(account.ContractKey(hash))
	if data == nil {
		return nil, errors.New("Failed to find control program through claim script")
	}

	cp := &account.CtrlProgram{}
	if err := json.Unmarshal(data, cp); err != nil {
		return nil, errors.New("Failed on unmarshal control program")
	}

	// 构造交易
	// 用输出作为交易输入 生成新的交易
	builder := txbuilder.NewBuilder(time.Now())
	// TODO 根据raw tx生成一个utxo

	sourceID := *rawTx.OutputID(nOut)
	outputAccount := rawTx.Outputs[nOut].Amount
	assetID := *rawTx.Outputs[nOut].AssetId

	txInput := types.NewClaimInput(nil, sourceID, assetID, outputAccount, uint64(nOut), cp.ControlProgram)
	if err := builder.AddInput(txInput, &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	program, err := b.Wallet.AccountMgr.CreateAddress(cp.AccountID, false)
	if err != nil {
		return nil, err
	}

	if err = builder.AddOutput(types.NewTxOutput(assetID, outputAccount, program.ControlProgram)); err != nil {
		return nil, err
	}

	tmpl, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// todo把一些主链的信息加到交易的stack中
	var stack [][]byte

	//amount
	amount := strconv.FormatUint(rawTx.Outputs[nOut].Amount, 10)
	stack = append(stack, []byte(amount))
	// 主链的gennesisBlockHash
	stack = append(stack, []byte(consensus.ActiveNetParams.ParentGenesisBlockHash))
	// claim script
	stack = append(stack, claimScript)
	// raw tx
	tx, _ := rawTx.MarshalText()
	stack = append(stack, tx)
	// proof
	blockHeaderByte, err := blockHeader.MarshalText()
	if err != nil {
		return nil, err
	}
	merkleBlock := claim.MerkleBlock{
		BlockHeader:  blockHeaderByte,
		TxHashes:     txHashes,
		StatusHashes: statusHashes,
		Flags:        ins.Flags,
		MatchedTxIDs: matchedTxIDs,
	}
	txOutProof, _ := json.Marshal(merkleBlock)
	stack = append(stack, txOutProof)
	//	tmpl.Transaction.Inputs[0].Peginwitness = stack
	txData.Inputs[0].Peginwitness = stack

	//交易费估算
	txGasResp, err := EstimateTxGas(*tmpl)
	if err != nil {
		return nil, err
	}
	txData.Outputs[0].Amount = txData.Outputs[0].Amount - uint64(txGasResp.TotalNeu)
	//重设置Transaction
	tmpl.Transaction = types.NewTx(*txData)
	return tmpl, nil
}

// EstimateTxGasResp estimate transaction consumed gas
type EstimateTxGasResp struct {
	TotalNeu   int64 `json:"total_neu"`
	StorageNeu int64 `json:"storage_neu"`
	VMNeu      int64 `json:"vm_neu"`
}

var (
	defaultBaseRate = float64(100000)
	flexibleGas     = int64(1800)
)

// EstimateTxGas estimate consumed neu for transaction
func EstimateTxGas(template txbuilder.Template) (*EstimateTxGasResp, error) {
	// base tx size and not include sign
	data, err := template.Transaction.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	baseTxSize := int64(len(data))

	// extra tx size for sign witness parts
	signSize := estimateSignSize(template.SigningInstructions)

	// total gas for tx storage
	totalTxSizeGas, ok := checked.MulInt64(baseTxSize+signSize, consensus.StorageGasRate)
	if !ok {
		return nil, errors.New("calculate txsize gas got a math error")
	}

	// consume gas for run VM
	totalP2WPKHGas := int64(0)
	totalP2WSHGas := int64(0)
	baseP2WPKHGas := int64(1419)
	// flexible Gas is used for handle need extra utxo situation

	for pos, inpID := range template.Transaction.Tx.InputIDs {
		sp, err := template.Transaction.Spend(inpID)
		if err != nil {
			continue
		}

		resOut, err := template.Transaction.Output(*sp.SpentOutputId)
		if err != nil {
			continue
		}

		if segwit.IsP2WPKHScript(resOut.ControlProgram.Code) {
			totalP2WPKHGas += baseP2WPKHGas
		} else if segwit.IsP2WSHScript(resOut.ControlProgram.Code) {
			sigInst := template.SigningInstructions[pos]
			totalP2WSHGas += estimateP2WSHGas(sigInst)
		}
	}

	// total estimate gas
	totalGas := totalTxSizeGas + totalP2WPKHGas + totalP2WSHGas + flexibleGas

	// rounding totalNeu with base rate 100000
	totalNeu := float64(totalGas*consensus.VMGasRate) / defaultBaseRate
	roundingNeu := math.Ceil(totalNeu)
	estimateNeu := int64(roundingNeu) * int64(defaultBaseRate)

	// TODO add priority

	return &EstimateTxGasResp{
		TotalNeu:   estimateNeu,
		StorageNeu: totalTxSizeGas * consensus.VMGasRate,
		VMNeu:      (totalP2WPKHGas + totalP2WSHGas) * consensus.VMGasRate,
	}, nil
}

// estimate p2wsh gas.
// OP_CHECKMULTISIG consume (984 * a - 72 * b - 63) gas,
// where a represent the num of public keys, and b represent the num of quorum.
func estimateP2WSHGas(sigInst *txbuilder.SigningInstruction) int64 {
	P2WSHGas := int64(0)
	baseP2WSHGas := int64(738)

	for _, witness := range sigInst.WitnessComponents {
		switch t := witness.(type) {
		case *txbuilder.SignatureWitness:
			P2WSHGas += baseP2WSHGas + (984*int64(len(t.Keys)) - 72*int64(t.Quorum) - 63)
		case *txbuilder.RawTxSigWitness:
			P2WSHGas += baseP2WSHGas + (984*int64(len(t.Keys)) - 72*int64(t.Quorum) - 63)
		}
	}
	return P2WSHGas
}

// estimate signature part size.
// if need multi-sign, calculate the size according to the length of keys.
func estimateSignSize(signingInstructions []*txbuilder.SigningInstruction) int64 {
	signSize := int64(0)
	baseWitnessSize := int64(300)

	for _, sigInst := range signingInstructions {
		for _, witness := range sigInst.WitnessComponents {
			switch t := witness.(type) {
			case *txbuilder.SignatureWitness:
				signSize += int64(t.Quorum) * baseWitnessSize
			case *txbuilder.RawTxSigWitness:
				signSize += int64(t.Quorum) * baseWitnessSize
			}
		}
	}
	return signSize
}
