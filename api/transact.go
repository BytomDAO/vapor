package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/blockchain/signers"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/blockchain/txbuilder/mainchain"
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/crypto/sha3pool"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/net/http/reqid"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/bc/types/bytom"
	bytomtypes "github.com/vapor/protocol/bc/types/bytom/types"
	"github.com/vapor/protocol/vm/vmutil"
	"github.com/vapor/util"
)

var (
	defaultTxTTL    = 5 * time.Minute
	defaultBaseRate = float64(100000)
)

func (a *API) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	decoders := map[string]func([]byte) (txbuilder.Action, error){
		"control_address":              txbuilder.DecodeControlAddressAction,
		"control_program":              txbuilder.DecodeControlProgramAction,
		"issue":                        a.wallet.AssetReg.DecodeIssueAction,
		"retire":                       txbuilder.DecodeRetireAction,
		"spend_account":                a.wallet.AccountMgr.DecodeSpendAction,
		"spend_account_unspent_output": a.wallet.AccountMgr.DecodeSpendUTXOAction,
	}
	decoder, ok := decoders[action]
	return decoder, ok
}

func onlyHaveInputActions(req *BuildRequest) (bool, error) {
	count := 0
	for i, act := range req.Actions {
		actionType, ok := act["type"].(string)
		if !ok {
			return false, errors.WithDetailf(ErrBadActionType, "no action type provided on action %d", i)
		}

		if strings.HasPrefix(actionType, "spend") || actionType == "issue" {
			count++
		}
	}

	return count == len(req.Actions), nil
}

func (a *API) buildSingle(ctx context.Context, req *BuildRequest) (*txbuilder.Template, error) {
	if err := a.completeMissingIDs(ctx, req); err != nil {
		return nil, err
	}

	if ok, err := onlyHaveInputActions(req); err != nil {
		return nil, err
	} else if ok {
		return nil, errors.WithDetail(ErrBadActionConstruction, "transaction contains only input actions and no output actions")
	}

	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(ErrBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := a.actionDecoder(typ)
		if !ok {
			return nil, errors.WithDetailf(ErrBadActionType, "unknown action type %q on action %d", typ, i)
		}

		// Remarshal to JSON, the action may have been modified when we
		// filtered aliases.
		b, err := json.Marshal(act)
		if err != nil {
			return nil, err
		}
		action, err := decoder(b)
		if err != nil {
			return nil, errors.WithDetailf(ErrBadAction, "%s on action %d", err.Error(), i)
		}
		actions = append(actions, action)
	}
	actions = account.MergeSpendAction(actions)

	ttl := req.TTL.Duration
	if ttl == 0 {
		ttl = defaultTxTTL
	}
	maxTime := time.Now().Add(ttl)

	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime, req.TimeRange)
	if errors.Root(err) == txbuilder.ErrAction {
		// append each of the inner errors contained in the data.
		var Errs string
		var rootErr error
		for i, innerErr := range errors.Data(err)["actions"].([]error) {
			if i == 0 {
				rootErr = errors.Root(innerErr)
			}
			Errs = Errs + innerErr.Error()
		}
		err = errors.WithDetail(rootErr, Errs)
	}
	if err != nil {
		return nil, err
	}

	// ensure null is never returned for signing instructions
	if tpl.SigningInstructions == nil {
		tpl.SigningInstructions = []*txbuilder.SigningInstruction{}
	}
	return tpl, nil
}

// POST /build-transaction
func (a *API) build(ctx context.Context, buildReqs *BuildRequest) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	tmpl, err := a.buildSingle(subctx, buildReqs)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tmpl)
}

type submitTxResp struct {
	TxID *bc.Hash `json:"tx_id"`
}

// POST /submit-transaction
func (a *API) submit(ctx context.Context, ins struct {
	Tx types.Tx `json:"raw_transaction"`
}) Response {
	if err := txbuilder.FinalizeTx(ctx, a.chain, &ins.Tx); err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("tx_id", ins.Tx.ID.String()).Info("submit single tx")
	return NewSuccessResponse(&submitTxResp{TxID: &ins.Tx.ID})
}

// EstimateTxGasResp estimate transaction consumed gas
type EstimateTxGasResp struct {
	TotalNeu   int64 `json:"total_neu"`
	StorageNeu int64 `json:"storage_neu"`
	VMNeu      int64 `json:"vm_neu"`
}

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
	totalGas := totalTxSizeGas + totalP2WPKHGas + totalP2WSHGas

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

// POST /estimate-transaction-gas
func (a *API) estimateTxGas(ctx context.Context, in struct {
	TxTemplate txbuilder.Template `json:"transaction_template"`
}) Response {
	txGasResp, err := EstimateTxGas(in.TxTemplate)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(txGasResp)
}

func getPeginTxnOutputIndex(rawTx bytomtypes.Tx, controlProg []byte) int {
	for index, output := range rawTx.Outputs {
		if bytes.Equal(output.ControlProgram, controlProg) {
			return index
		}
	}
	return 0
}

func toHash(hexBytes []chainjson.HexBytes) (hashs []*bytom.Hash) {
	for _, data := range hexBytes {
		b32 := [32]byte{}
		copy(b32[:], data)
		res := bytom.NewHash(b32)
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
	tmpl, err := a.createrawpegin(ctx, ins)
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

// GetMerkleBlockResp is resp struct for GetTxOutProof API
type GetMerkleBlock struct {
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
}

func (a *API) createrawpegin(ctx context.Context, ins struct {
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
	if !bytomtypes.ValidateTxMerkleTreeProof(txHashes, flags, matchedTxIDs, ins.BlockHeader.BlockCommitment.TransactionsMerkleRoot) {
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
	if nOut == len(ins.RawTx.Outputs) {
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
	//txInput := types.NewClaimInputInput(nil, *ins.RawTx.Outputs[nOut].AssetId, ins.RawTx.Outputs[nOut].Amount, cp.ControlProgram)
	assetId := bc.AssetID{}
	assetId.V0 = ins.RawTx.Outputs[nOut].AssetId.GetV0()
	assetId.V1 = ins.RawTx.Outputs[nOut].AssetId.GetV1()
	assetId.V2 = ins.RawTx.Outputs[nOut].AssetId.GetV2()
	assetId.V3 = ins.RawTx.Outputs[nOut].AssetId.GetV3()

	sourceID := bc.Hash{}
	sourceID.V0 = ins.RawTx.OutputID(nOut).GetV0()
	sourceID.V1 = ins.RawTx.OutputID(nOut).GetV1()
	sourceID.V2 = ins.RawTx.OutputID(nOut).GetV2()
	sourceID.V3 = ins.RawTx.OutputID(nOut).GetV3()
	outputAccount := ins.RawTx.Outputs[nOut].Amount

	txInput := types.NewClaimInputInput(nil, sourceID, assetId, outputAccount, uint64(nOut), cp.ControlProgram)
	if err := builder.AddInput(txInput, &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	program, err := a.wallet.AccountMgr.CreateAddress(cp.AccountID, false)
	if err != nil {
		return nil, err
	}

	if err = builder.AddOutput(types.NewTxOutput(assetId, outputAccount, program.ControlProgram)); err != nil {
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
	MerkleBLock := GetMerkleBlock{
		BlockHeader:  ins.BlockHeader,
		TxHashes:     ins.TxHashes,
		StatusHashes: ins.StatusHashes,
		Flags:        ins.Flags,
		MatchedTxIDs: ins.MatchedTxIDs,
	}
	txOutProof, _ := json.Marshal(MerkleBLock)
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

func (a *API) buildMainChainTx(ins struct {
	Utxo           account.UTXO       `json:"utxo"`
	Tx             types.Tx           `json:"raw_transaction"`
	RootXPubs      []chainkd.XPub     `json:"root_xpubs"`
	Alias          string             `json:"alias"`
	ControlProgram string             `json:"control_program"`
	ClaimScript    chainjson.HexBytes `json:"claim_script"`
}) Response {

	var xpubs []chainkd.XPub
	for _, xpub := range ins.RootXPubs {
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(ins.ClaimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		chaildXPub := xpub.Child(tweak)
		xpubs = append(xpubs, chaildXPub)
	}
	acc := &account.Account{}
	var err error
	if acc, err = a.wallet.AccountMgr.FindByAlias(ins.Alias); err != nil {
		acc, err = a.wallet.AccountMgr.Create(xpubs, len(xpubs), ins.Alias)
		if err != nil {
			return NewErrorResponse(err)
		}
	}
	ins.Utxo.ControlProgramIndex = acc.Signer.KeyIndex

	txInput, sigInst, err := utxoToInputs(acc.Signer, &ins.Utxo)
	if err != nil {
		return NewErrorResponse(err)
	}

	builder := mainchain.NewBuilder(time.Now())
	builder.AddInput(txInput, sigInst)
	changeAmount := uint64(0)
	retire := false
	for _, key := range ins.Tx.GetResultIds() {
		output, err := ins.Tx.Retire(*key)
		if err != nil {
			log.WithFields(log.Fields{"moudle": "transact", "err": err}).Warn("buildMainChainTx error")
			continue
		}
		retire = true
		var controlProgram []byte
		retBool := true
		if controlProgram, retBool = getInput(ins.Tx.Entries, *key, ins.ControlProgram); !retBool {
			return NewErrorResponse(errors.New("The corresponding input cannot be found"))
		}

		assetID := bytom.AssetID{
			V0: output.Source.Value.AssetId.GetV0(),
			V1: output.Source.Value.AssetId.GetV1(),
			V2: output.Source.Value.AssetId.GetV2(),
			V3: output.Source.Value.AssetId.GetV3(),
		}
		out := bytomtypes.NewTxOutput(assetID, output.Source.Value.Amount, controlProgram)
		builder.AddOutput(out)
		changeAmount = ins.Utxo.Amount - output.Source.Value.Amount

	}

	if !retire {
		return NewErrorResponse(errors.New("It's not a transaction to retire assets"))
	}

	if changeAmount > 0 {
		u := ins.Utxo
		assetID := bytom.AssetID{
			V0: u.AssetID.GetV0(),
			V1: u.AssetID.GetV1(),
			V2: u.AssetID.GetV2(),
			V3: u.AssetID.GetV3(),
		}
		out := bytomtypes.NewTxOutput(assetID, changeAmount, ins.Utxo.ControlProgram)
		builder.AddOutput(out)
	}

	tmpl, tx, err := builder.Build()
	if err != nil {
		return NewErrorResponse(err)
	}
	//交易费估算
	txGasResp, err := EstimateTxGasForMainchain(*tmpl)
	if err != nil {
		return NewErrorResponse(err)
	}
	for i, out := range tmpl.Transaction.Outputs {
		if bytes.Equal(out.ControlProgram, ins.Utxo.ControlProgram) {
			tx.Outputs[i].Amount = changeAmount - uint64(txGasResp.TotalNeu)
		}
	}
	tmpl.Transaction = bytomtypes.NewTx(*tx)
	return NewSuccessResponse(tmpl)
}

//
func getInput(entry map[bc.Hash]bc.Entry, outputID bc.Hash, controlProgram string) ([]byte, bool) {
	output := entry[outputID].(*bc.Retirement)
	mux := entry[*output.Source.Ref].(*bc.Mux)

	for _, valueSource := range mux.GetSources() {
		spend := entry[*valueSource.Ref].(*bc.Spend)
		prevout := entry[*spend.SpentOutputId].(*bc.Output)

		var ctrlProgram chainjson.HexBytes
		ctrlProgram = prevout.ControlProgram.Code
		tmp, _ := ctrlProgram.MarshalText()
		if string(tmp) == controlProgram {
			return ctrlProgram, true
		}
	}
	return nil, false
}

// UtxoToInputs convert an utxo to the txinput
func utxoToInputs(signer *signers.Signer, u *account.UTXO) (*bytomtypes.TxInput, *mainchain.SigningInstruction, error) {
	sourceID := bytom.Hash{
		V0: u.SourceID.GetV0(),
		V1: u.SourceID.GetV1(),
		V2: u.SourceID.GetV2(),
		V3: u.SourceID.GetV3(),
	}

	assetID := bytom.AssetID{
		V0: u.AssetID.GetV0(),
		V1: u.AssetID.GetV1(),
		V2: u.AssetID.GetV2(),
		V3: u.AssetID.GetV3(),
	}

	txInput := bytomtypes.NewSpendInput(nil, sourceID, assetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst := &mainchain.SigningInstruction{}
	if signer == nil {
		return txInput, sigInst, nil
	}

	path := signers.Path(signer, signers.AccountKeySpace, u.ControlProgramIndex)
	if u.Address == "" {
		sigInst.AddWitnessKeys(signer.XPubs, path, signer.Quorum)
		return txInput, sigInst, nil
	}

	address, err := common.DecodeBytomAddress(u.Address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, nil, err
	}

	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
		derivedPK := derivedXPubs[0].PublicKey()
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, mainchain.DataWitness([]byte(derivedPK)))

	case *common.AddressWitnessScriptHash:
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		//path := signers.Path(signer, signers.AccountKeySpace, u.ControlProgramIndex)
		//derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
		derivedXPubs := signer.XPubs
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, signer.Quorum)
		if err != nil {
			return nil, nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, mainchain.DataWitness(script))

	default:
		return nil, nil, errors.New("unsupport address type")
	}

	return txInput, sigInst, nil
}

type signRespForMainchain struct {
	Tx           *mainchain.Template `json:"transaction"`
	SignComplete bool                `json:"sign_complete"`
}

func (a *API) signWithKey(ins struct {
	Xprv        string             `json:"xprv"`
	XPub        chainkd.XPub       `json:"xpub"`
	Txs         mainchain.Template `json:"transaction"`
	ClaimScript chainjson.HexBytes `json:"claim_script"`
}) Response {
	xprv := &chainkd.XPrv{}
	if err := xprv.UnmarshalText([]byte(ins.Xprv)); err != nil {
		return NewErrorResponse(err)
	}
	// pub + scriptPubKey 生成一个随机数A
	var tmp [32]byte
	h := hmac.New(sha256.New, ins.XPub[:])
	h.Write(ins.ClaimScript)
	tweak := h.Sum(tmp[:])
	// pub +  A 生成一个新的公钥pub_new
	privateKey := xprv.Child(tweak, false)

	if err := sign(&ins.Txs, privateKey); err != nil {
		return NewErrorResponse(err)
	}
	log.Info("Sign Transaction complete.")
	log.Info(mainchain.SignProgress(&ins.Txs))
	return NewSuccessResponse(&signRespForMainchain{Tx: &ins.Txs, SignComplete: mainchain.SignProgress(&ins.Txs)})
}

func sign(tmpl *mainchain.Template, xprv chainkd.XPrv) error {
	for i, sigInst := range tmpl.SigningInstructions {
		for j, wc := range sigInst.WitnessComponents {
			switch sw := wc.(type) {
			case *mainchain.SignatureWitness:
				err := sw.Sign(tmpl, uint32(i), xprv)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to signature witness component %d of input %d", j, i)
				}
			case *mainchain.RawTxSigWitness:
				err := sw.Sign(tmpl, uint32(i), xprv)
				if err != nil {
					return errors.WithDetailf(err, "adding signature(s) to raw-signature witness component %d of input %d", j, i)
				}
			}
		}
	}
	return materializeWitnesses(tmpl)
}

func materializeWitnesses(txTemplate *mainchain.Template) error {
	msg := txTemplate.Transaction

	if msg == nil {
		return errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	if len(txTemplate.SigningInstructions) > len(msg.Inputs) {
		return errors.Wrap(txbuilder.ErrBadInstructionCount)
	}

	for i, sigInst := range txTemplate.SigningInstructions {
		if msg.Inputs[sigInst.Position] == nil {
			return errors.WithDetailf(txbuilder.ErrBadTxInputIdx, "signing instruction %d references missing tx input %d", i, sigInst.Position)
		}

		var witness [][]byte
		for j, wc := range sigInst.WitnessComponents {
			err := wc.Materialize(&witness)
			if err != nil {
				return errors.WithDetailf(err, "error in witness component %d of input %d", j, i)
			}
		}
		msg.SetInputArguments(sigInst.Position, witness)
	}

	return nil
}
