package api

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/account"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/sha3pool"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	bytomtypes "github.com/vapor/protocol/bc/types/bytom/types"
	"github.com/vapor/protocol/validation"
	"github.com/vapor/util"
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

func (a *API) claimPeginTx(ctx context.Context, ins struct {
	Password     string                 `json:"password"`
	RawTx        bytomtypes.Tx          `json:"raw_transaction"`
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
	ClaimScript  chainjson.HexBytes     `json:"claim_script"`
}) Response {
	tmpl, err := a.createRawPegin(ctx, ins)
	if err != nil {
		log.WithField("build err", err).Error("fail on createrawpegin.")
		return NewErrorResponse(err)
	}
	// 交易签名
	if err := txbuilder.Sign(ctx, tmpl, ins.Password, a.PseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return NewErrorResponse(err)
	}

	// submit
	if err := txbuilder.FinalizeTx(ctx, a.chain, tmpl.Transaction); err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("tx_id", tmpl.Transaction.ID.String()).Info("claim script tx")
	return NewSuccessResponse(&submitTxResp{TxID: &tmpl.Transaction.ID})
}

func (a *API) createRawPegin(ctx context.Context, ins struct {
	Password     string                 `json:"password"`
	RawTx        bytomtypes.Tx          `json:"raw_transaction"`
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
	ClaimScript  chainjson.HexBytes     `json:"claim_script"`
}) (*txbuilder.Template, error) {
	// proof验证
	var flags []uint8
	for flag := range ins.Flags {
		flags = append(flags, uint8(flag))
	}
	txHashes := toHash(ins.TxHashes)
	matchedTxIDs := toHash(ins.MatchedTxIDs)
	statusHashes := toHash(ins.StatusHashes)
	if !types.ValidateTxMerkleTreeProof(txHashes, flags, matchedTxIDs, ins.BlockHeader.BlockCommitment.TransactionsMerkleRoot) {
		return nil, errors.New("Merkleblock validation failed")
	}
	// CheckBytomProof
	//difficulty.CheckBytomProofOfWork(ins.BlockHeader.Hash(), ins.BlockHeader)
	// 增加spv验证以及连接主链api查询交易的确认数
	if util.ValidatePegin {
		if err := util.IsConfirmedBytomBlock(ins.BlockHeader.Height, consensus.ActiveNetParams.PeginMinDepth); err != nil {
			return nil, err
		}
	}
	// 找出与claim script有关联的交易的输出
	var claimScript []byte
	nOut := len(ins.RawTx.Outputs)
	if ins.ClaimScript == nil {
		// 遍历寻找与交易输出有关的claim script
		cps, err := a.wallet.AccountMgr.ListControlProgram()
		if err != nil {
			return nil, err
		}

		for _, cp := range cps {
			_, controlProg := a.wallet.AccountMgr.GetPeginControlPrograms(cp.ControlProgram)
			if controlProg == nil {
				continue
			}
			// 获取交易的输出
			nOut = getPeginTxnOutputIndex(ins.RawTx, controlProg)
			if nOut != len(ins.RawTx.Outputs) {
				claimScript = cp.ControlProgram
			}
		}
	} else {
		claimScript = ins.ClaimScript
		_, controlProg := a.wallet.AccountMgr.GetPeginControlPrograms(claimScript)
		// 获取交易的输出
		nOut = getPeginTxnOutputIndex(ins.RawTx, controlProg)
	}
	if nOut == len(ins.RawTx.Outputs) || nOut == -1 {
		return nil, errors.New("Failed to find output in bytom to the mainchain_address from getpeginaddress")
	}

	// 根据ClaimScript 获取account id
	var hash [32]byte
	sha3pool.Sum256(hash[:], claimScript)
	data := a.wallet.DB.Get(account.ContractKey(hash))
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
	sourceID := *ins.RawTx.OutputID(nOut)
	outputAccount := ins.RawTx.Outputs[nOut].Amount
	assetID := *ins.RawTx.Outputs[nOut].AssetId

	txInput := types.NewClaimInput(nil, sourceID, assetID, outputAccount, uint64(nOut), cp.ControlProgram)
	if err := builder.AddInput(txInput, &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	program, err := a.wallet.AccountMgr.CreateAddress(cp.AccountID, false)
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
	amount := strconv.FormatUint(ins.RawTx.Outputs[nOut].Amount, 10)
	stack = append(stack, []byte(amount))
	// 主链的gennesisBlockHash
	stack = append(stack, []byte(consensus.ActiveNetParams.ParentGenesisBlockHash))
	// claim script
	stack = append(stack, claimScript)
	// raw tx
	tx, _ := json.Marshal(ins.RawTx)
	stack = append(stack, tx)
	// proof
	blockHeader, err := ins.BlockHeader.MarshalText()
	if err != nil {
		return nil, err
	}
	merkleBlock := validation.MerkleBlock{
		BlockHeader:  blockHeader,
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

func (a *API) claimContractPeginTx(ctx context.Context, ins struct {
	Password     string                 `json:"password"`
	RawTx        bytomtypes.Tx          `json:"raw_transaction"`
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
	ClaimScript  chainjson.HexBytes     `json:"claim_script"`
}) Response {
	tmpl, err := a.createContractRawPegin(ctx, ins)
	if err != nil {
		log.WithField("build err", err).Error("fail on claimContractPeginTx.")
		return NewErrorResponse(err)
	}
	// 交易签名
	if err := txbuilder.Sign(ctx, tmpl, ins.Password, a.PseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return NewErrorResponse(err)
	}

	// submit
	if err := txbuilder.FinalizeTx(ctx, a.chain, tmpl.Transaction); err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("tx_id", tmpl.Transaction.ID.String()).Info("claim script tx")
	return NewSuccessResponse(&submitTxResp{TxID: &tmpl.Transaction.ID})
}

func (a *API) createContractRawPegin(ctx context.Context, ins struct {
	Password     string                 `json:"password"`
	RawTx        bytomtypes.Tx          `json:"raw_transaction"`
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
	ClaimScript  chainjson.HexBytes     `json:"claim_script"`
}) (*txbuilder.Template, error) {
	// proof验证
	var flags []uint8
	for flag := range ins.Flags {
		flags = append(flags, uint8(flag))
	}
	txHashes := toHash(ins.TxHashes)
	matchedTxIDs := toHash(ins.MatchedTxIDs)
	statusHashes := toHash(ins.StatusHashes)
	if !types.ValidateTxMerkleTreeProof(txHashes, flags, matchedTxIDs, ins.BlockHeader.BlockCommitment.TransactionsMerkleRoot) {
		return nil, errors.New("Merkleblock validation failed")
	}
	// CheckBytomProof
	//difficulty.CheckBytomProofOfWork(ins.BlockHeader.Hash(), ins.BlockHeader)
	// 增加spv验证以及连接主链api查询交易的确认数
	if util.ValidatePegin {
		if err := util.IsConfirmedBytomBlock(ins.BlockHeader.Height, consensus.ActiveNetParams.PeginMinDepth); err != nil {
			return nil, err
		}
	}
	// 找出与claim script有关联的交易的输出
	var claimScript []byte
	nOut := len(ins.RawTx.Outputs)
	if ins.ClaimScript == nil {
		// 遍历寻找与交易输出有关的claim script
		cps, err := a.wallet.AccountMgr.ListControlProgram()
		if err != nil {
			return nil, err
		}

		for _, cp := range cps {
			_, controlProg := a.wallet.AccountMgr.GetPeginContractControlPrograms(claimScript)
			// 获取交易的输出
			nOut = getPeginTxnOutputIndex(ins.RawTx, controlProg)
			if nOut != len(ins.RawTx.Outputs) {
				claimScript = cp.ControlProgram
			}
		}
	} else {
		claimScript = ins.ClaimScript
		_, controlProg := a.wallet.AccountMgr.GetPeginContractControlPrograms(claimScript)
		// 获取交易的输出
		nOut = getPeginTxnOutputIndex(ins.RawTx, controlProg)
	}
	if nOut == len(ins.RawTx.Outputs) || nOut == -1 {
		return nil, errors.New("Failed to find output in bytom to the mainchain_address from createContractRawPegin")
	}

	// 根据ClaimScript 获取account id
	var hash [32]byte
	sha3pool.Sum256(hash[:], claimScript)
	data := a.wallet.DB.Get(account.ContractKey(hash))
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

	sourceID := *ins.RawTx.OutputID(nOut)
	outputAccount := ins.RawTx.Outputs[nOut].Amount
	assetID := *ins.RawTx.Outputs[nOut].AssetId

	txInput := types.NewClaimInput(nil, sourceID, assetID, outputAccount, uint64(nOut), cp.ControlProgram)
	if err := builder.AddInput(txInput, &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	program, err := a.wallet.AccountMgr.CreateAddress(cp.AccountID, false)
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
	amount := strconv.FormatUint(ins.RawTx.Outputs[nOut].Amount, 10)
	stack = append(stack, []byte(amount))
	// 主链的gennesisBlockHash
	stack = append(stack, []byte(consensus.ActiveNetParams.ParentGenesisBlockHash))
	// claim script
	stack = append(stack, claimScript)
	// raw tx
	tx, _ := ins.RawTx.MarshalText()
	//tx, _ := json.Marshal(ins.RawTx)
	stack = append(stack, tx)
	// proof
	blockHeader, err := ins.BlockHeader.MarshalText()
	if err != nil {
		return nil, err
	}
	merkleBlock := validation.MerkleBlock{
		BlockHeader:  blockHeader,
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
