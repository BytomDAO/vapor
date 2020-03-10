package vmutil

import (
	"github.com/bytom/vapor/crypto/ed25519"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/vm"
)

// pre-define errors
var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

// MagneticContractArgs is a struct for magnetic contract arguments
type MagneticContractArgs struct {
	RequestedAsset   bc.AssetID
	RatioNumerator   int64
	RatioDenominator int64
	SellerProgram    []byte
	SellerKey        []byte
}

// IsUnspendable checks if a contorl program is absolute failed
func IsUnspendable(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}

func (b *Builder) addP2SPMultiSig(pubkeys []ed25519.PublicKey, nrequired int) error {
	if err := checkMultiSigParams(int64(nrequired), int64(len(pubkeys))); err != nil {
		return err
	}

	b.AddOp(vm.OP_TXSIGHASH) // stack is now [... NARGS SIG SIG SIG PREDICATEHASH]
	for _, p := range pubkeys {
		b.AddData(p)
	}
	b.AddInt64(int64(nrequired))    // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M]
	b.AddInt64(int64(len(pubkeys))) // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M N]
	b.AddOp(vm.OP_CHECKMULTISIG)    // stack is now [... NARGS]
	return nil
}

// DefaultCoinbaseProgram generates the script for contorl coinbase output
func DefaultCoinbaseProgram() ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_TRUE)
	return builder.Build()
}

// P2WPKHProgram return the segwit pay to public key hash
func P2WPKHProgram(hash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(hash)
	return builder.Build()
}

// P2WSHProgram return the segwit pay to script hash
func P2WSHProgram(hash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(hash)
	return builder.Build()
}

// RetireProgram generates the script for retire output
func RetireProgram(comment []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_FAIL)
	if len(comment) != 0 {
		builder.AddData(comment)
	}
	return builder.Build()
}

// P2PKHSigProgram generates the script for control with pubkey hash
func P2PKHSigProgram(pubkeyHash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_HASH160)
	builder.AddData(pubkeyHash)
	builder.AddOp(vm.OP_EQUALVERIFY)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	return builder.Build()
}

// P2SHProgram generates the script for control with script hash
func P2SHProgram(scriptHash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_SHA3)
	builder.AddData(scriptHash)
	builder.AddOp(vm.OP_EQUALVERIFY)
	builder.AddInt64(-1)
	builder.AddOp(vm.OP_SWAP)
	builder.AddInt64(0)
	builder.AddOp(vm.OP_CHECKPREDICATE)
	return builder.Build()
}

// P2SPMultiSigProgram generates the script for control transaction output
func P2SPMultiSigProgram(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	builder := NewBuilder()
	if err := builder.addP2SPMultiSig(pubkeys, nrequired); err != nil {
		return nil, err
	}
	return builder.Build()
}

// P2SPMultiSigProgramWithHeight generates the script with block height for control transaction output
func P2SPMultiSigProgramWithHeight(pubkeys []ed25519.PublicKey, nrequired int, blockHeight int64) ([]byte, error) {
	builder := NewBuilder()
	if blockHeight > 0 {
		builder.AddInt64(blockHeight)
		builder.AddOp(vm.OP_BLOCKHEIGHT)
		builder.AddOp(vm.OP_GREATERTHAN)
		builder.AddOp(vm.OP_VERIFY)
	} else if blockHeight < 0 {
		return nil, errors.WithDetail(ErrBadValue, "negative blockHeight")
	}
	if err := builder.addP2SPMultiSig(pubkeys, nrequired); err != nil {
		return nil, err
	}
	return builder.Build()
}

func checkMultiSigParams(nrequired, npubkeys int64) error {
	if nrequired < 0 {
		return errors.WithDetail(ErrBadValue, "negative quorum")
	}
	if npubkeys < 0 {
		return errors.WithDetail(ErrBadValue, "negative pubkey count")
	}
	if nrequired > npubkeys {
		return errors.WithDetail(ErrBadValue, "quorum too big")
	}
	if nrequired == 0 && npubkeys > 0 {
		return errors.WithDetail(ErrBadValue, "quorum empty with non-empty pubkey list")
	}
	return nil
}

// P2WMCProgram return the segwit pay to magnetic contract
func P2WMCProgram(magneticContractArgs MagneticContractArgs) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(magneticContractArgs.RequestedAsset.Bytes())
	builder.AddInt64(magneticContractArgs.RatioNumerator)
	builder.AddInt64(magneticContractArgs.RatioDenominator)
	builder.AddData(magneticContractArgs.SellerProgram)
	builder.AddData(magneticContractArgs.SellerKey)
	return builder.Build()
}

// P2MCProgram generates the script for control with magnetic contract
//
// MagneticContract source code:
// contract MagneticContract(requestedAsset: Asset,
//                           ratioNumerator: Integer,
//                           ratioDenominator: Integer,
//                           sellerProgram: Program,
//                           standardProgram: Program,
//                           sellerKey: PublicKey) locks valueAmount of valueAsset {
//  clause partialTrade(exchangeAmount: Amount) {
//   define actualAmount: Integer = exchangeAmount * ratioDenominator / ratioNumerator
//   verify actualAmount > 0 && actualAmount < valueAmount
//   define receiveAmount: Integer = exchangeAmount * 999 / 1000
//   lock receiveAmount of requestedAsset with sellerProgram
//   lock valueAmount-actualAmount of valueAsset with standardProgram
//   unlock actualAmount of valueAsset
//  }
//  clause fullTrade() {
//   define requestedAmount: Integer = valueAmount * ratioNumerator / ratioDenominator
//   define requestedAmount: Integer = requestedAmount * 999 / 1000
//   verify requestedAmount > 0
//   lock requestedAmount of requestedAsset with sellerProgram
//   unlock valueAmount of valueAsset
//  }
//  clause cancel(sellerSig: Signature) {
//   verify checkTxSig(sellerKey, sellerSig)
//   unlock valueAmount of valueAsset
//  }
// }
//
// contract stack flow:
// 7                        [... <position> <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 7]
// ROLL                     [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <position>]
// TOALTSTACK               [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// 6                        [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 6]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector>]
// DUP                      [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector> <clause selector>]
// 2                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector> <clause selector> 2]
// NUMEQUAL                 [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector> (<clause selector> == 2)]
// JUMPIF:$cancel           [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector>]
// JUMPIF:$fullTrade        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// $partialTrade            [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// 6                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 6]
// PICK                     [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset exchangeAmount]
// 3                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset exchangeAmount 3]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram ratioNumerator requestedAsset exchangeAmount ratioDenominator]
// 3                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioNumerator requestedAsset exchangeAmount ratioDenominator 3]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset exchangeAmount ratioDenominator ratioNumerator]
// MULFRACTION              [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount]
// AMOUNT                   [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount]
// OVER                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount actualAmount]
// 0                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount actualAmount 0]
// GREATERTHAN              [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount > 0)]
// 2                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount > 0) 2]
// PICK                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount > 0) actualAmount]
// 2                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount > 0) actualAmount 2]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount (actualAmount > 0) actualAmount valueAmount]
// LESSTHAN                 [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount (actualAmount > 0) (actualAmount < valueAmount)]
// BOOLAND                  [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount ((actualAmount > 0) && (actualAmount < valueAmount))]
// VERIFY                   [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount]
// FROMALTSTACK             [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount <position>]
// DUP                      [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> <position>]
// TOALTSTACK               [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount <position>]
// 6                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> 6]
// ROLL                     [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> exchangeAmount]
// 999                      [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> exchangeAmount 999]
// 1000                     [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> exchangeAmount 1000]
// MULFRACTION              [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> receiveAmount]
// 3                        [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> receiveAmount 3]
// ROLL                     [... sellerKey standardProgram sellerProgram actualAmount <position> receiveAmount requestedAsset]
// 1                        [... sellerKey standardProgram sellerProgram actualAmount <position> receiveAmount requestedAsset 1]
// 5                        [... sellerKey standardProgram sellerProgram actualAmount <position> receiveAmount requestedAsset 1 5]
// ROLL                     [... sellerKey standardProgram actualAmount <position> receiveAmount requestedAsset 1 sellerProgram]
// CHECKOUTPUT              [... sellerKey standardProgram actualAmount checkOutput(receiveAmount, requestedAsset, sellerProgram)]
// VERIFY                   [... sellerKey standardProgram actualAmount]
// FROMALTSTACK             [... sellerKey standardProgram actualAmount <position>]
// 1                        [... sellerKey standardProgram actualAmount <position> 1]
// ADD                      [... sellerKey standardProgram actualAmount (<position> + 1)]
// AMOUNT                   [... sellerKey standardProgram actualAmount (<position> + 1) valueAmount]
// 2                        [... sellerKey standardProgram actualAmount (<position> + 1) valueAmount 2]
// ROLL                     [... sellerKey standardProgram (<position> + 1) valueAmount actualAmount]
// SUB                      [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount)]
// ASSET                    [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount) valueAsset]
// 1                        [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount) valueAsset 1]
// 4                        [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount) valueAsset 1 4]
// ROLL                     [... sellerKey (<position> + 1) (valueAmount - actualAmount) valueAsset 1 standardProgram]
// CHECKOUTPUT              [... sellerKey checkOutput((valueAmount - actualAmount), valueAsset, standardProgram)]
// JUMP:$_end               [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// $fullTrade               [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// AMOUNT                   [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset valueAmount]
// 2                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset valueAmount 2]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator requestedAsset valueAmount ratioNumerator]
// 3                        [... sellerKey standardProgram sellerProgram ratioDenominator requestedAsset valueAmount ratioNumerator 3]
// ROLL                     [... sellerKey standardProgram sellerProgram requestedAsset valueAmount ratioNumerator ratioDenominator]
// MULFRACTION              [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount]
// 999                      [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount 999]
// 1000                     [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount 999 1000]
// MULFRACTION              [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount]
// DUP                      [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount requestedAmount]
// 0                        [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount requestedAmount 0]
// GREATERTHAN              [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount (requestedAmount > 0)]
// VERIFY                   [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount]
// FROMALTSTACK             [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount <position>]
// SWAP                     [... sellerKey standardProgram sellerProgram requestedAsset <position> requestedAmount]
// 2                        [... sellerKey standardProgram sellerProgram requestedAsset <position> requestedAmount 2]
// ROLL                     [... sellerKey standardProgram sellerProgram <position> requestedAmount requestedAsset]
// 1                        [... sellerKey standardProgram sellerProgram <position> requestedAmount requestedAsset 1]
// 4                        [... sellerKey standardProgram sellerProgram <position> requestedAmount requestedAsset 1 4]
// ROLL                     [... sellerKey standardProgram <position> requestedAmount requestedAsset 1 sellerProgram]
// CHECKOUTPUT              [... sellerKey standardProgram checkOutput(requestedAmount, requestedAsset, sellerProgram)]
// JUMP:$_end               [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// $cancel                  [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector>]
// DROP                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// 6                        [... sellerSig sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 6]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset sellerSig]
// 6                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset sellerSig 6]
// ROLL                     [... standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset sellerSig sellerKey]
// TXSIGHASH SWAP CHECKSIG  [... standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset checkTxSig(sellerKey, sellerSig)]
// $_end                    [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
func P2MCProgram(magneticContractArgs MagneticContractArgs) ([]byte, error) {
	standardProgram, err := P2WMCProgram(magneticContractArgs)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder()
	// contract arguments
	builder.AddData(magneticContractArgs.SellerKey)
	builder.AddData(standardProgram)
	builder.AddData(magneticContractArgs.SellerProgram)
	builder.AddInt64(magneticContractArgs.RatioDenominator)
	builder.AddInt64(magneticContractArgs.RatioNumerator)
	builder.AddData(magneticContractArgs.RequestedAsset.Bytes())

	// contract instructions
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_NUMEQUAL)
	builder.AddJumpIf(0)
	builder.AddJumpIf(1)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_OVER)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHAN)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_LESSTHAN)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddInt64(999)
	builder.AddInt64(1000)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_5)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_ADD)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_SUB)
	builder.AddOp(vm.OP_ASSET)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddJump(2)
	builder.SetJumpTarget(1)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddInt64(999)
	builder.AddInt64(1000)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHAN)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddJump(3)
	builder.SetJumpTarget(0)
	builder.AddOp(vm.OP_DROP)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	builder.SetJumpTarget(2)
	builder.SetJumpTarget(3)
	return builder.Build()
}
