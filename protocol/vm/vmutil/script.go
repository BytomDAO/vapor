package vmutil

import (
	"strings"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/vm"
)

// magneticClauseSelector is the global selector for the magnetic transaction
var magneticClauseSelector = 0

// pre-define errors
var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

// MagneticContractArgs is a struct for magnetic contract arguments
type MagneticContractArgs struct {
	RequestedAsset   bc.AssetID
	RatioMolecule    int64
	RatioDenominator int64
	SellerProgram    []byte
	SellerKey        ed25519.PublicKey
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
	builder.AddInt64(magneticContractArgs.RatioMolecule)
	builder.AddInt64(magneticContractArgs.RatioDenominator)
	builder.AddData(magneticContractArgs.SellerProgram)
	builder.AddData(magneticContractArgs.SellerKey)
	return builder.Build()
}

// P2MCProgram generates the script for control with magnetic contract
func P2MCProgram(magneticContractArgs MagneticContractArgs, lockedAssetID bc.AssetID, clauseSelector int64) ([]byte, error) {
	standardProgram, err := P2WMCProgram(magneticContractArgs)
	if err != nil {
		return nil, err
	}

	assetComparedResult := strings.Compare(magneticContractArgs.RequestedAsset.String(), lockedAssetID.String())
	magneticProgram, err := MagneticProgram(assetComparedResult, clauseSelector)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder()
	builder.AddData(magneticContractArgs.SellerKey)
	builder.AddData(standardProgram)
	builder.AddData(magneticContractArgs.SellerProgram)
	builder.AddInt64(magneticContractArgs.RatioDenominator)
	builder.AddInt64(magneticContractArgs.RatioMolecule)
	builder.AddData(magneticContractArgs.RequestedAsset.Bytes())
	builder.AddOp(vm.OP_DEPTH)
	builder.AddData(magneticProgram)
	builder.AddOp(vm.OP_FALSE)
	builder.AddOp(vm.OP_CHECKPREDICATE)
	return builder.Build()
}

// MagneticProgram is the actual execute magnetic program which not contain arguments
//
// MagneticContract source code:
// contract MagneticContract(requestedAsset: Asset,
//                           ratioMolecule: Integer,
//                           ratioDenominator: Integer,
//                           sellerProgram: Program,
//                           standardProgram: Program,
//                           sellerKey: PublicKey) locks valueAmount of valueAsset {
// clause partialTrade(exchangeAmount: Amount) {
// 	 define actualAmount: Integer = exchangeAmount * ratioDenominator / ratioMolecule
// 	 verify actualAmount > 0 && actualAmount < valueAmount
//   lock exchangeAmount of requestedAsset with sellerProgram
//   lock valueAmount-actualAmount of valueAsset with standardProgram
//   unlock actualAmount of valueAsset
// }
// clause fullTrade(exchangeAmount: Amount) {
//   define actualAmount: Integer = exchangeAmount * ratioDenominator / ratioMolecule
//   verify actualAmount > 0 && actualAmount == valueAmount
//   lock exchangeAmount of requestedAsset with sellerProgram
//   unlock valueAmount of valueAsset
// }
// clause cancel(sellerSig: Signature) {
//   verify checkTxSig(sellerKey, sellerSig)
//   unlock valueAmount of valueAsset
// }
//
// contract stack flow:
// 6                        [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset 6]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset <clause selector>]
// DUP                      [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset <clause selector> <clause selector>]
// 2                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset <clause selector> <clause selector> 2]
// NUMEQUAL                 [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset <clause selector> (<clause selector> == 2)]
// JUMPIF:$cancel           [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset <clause selector>]
// JUMPIF:$fullTrade        [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
// $partialTrade            [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
// 6                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset 6]
// PICK                     [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset exchangeAmount]
// 3                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset exchangeAmount 3]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram ratioMolecule requestedAsset exchangeAmount ratioDenominator]
// MUL                      [... exchangeAmount sellerKey standardProgram sellerProgram ratioMolecule requestedAsset (exchangeAmount * ratioDenominator)]
// 2                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioMolecule requestedAsset (exchangeAmount * ratioDenominator) 2]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset (exchangeAmount * ratioDenominator) ratioMolecule]
// DIV                      [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount]
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
// 0                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount 0]
// 6                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount 0 6]
// ROLL                     [... sellerKey standardProgram sellerProgram requestedAsset actualAmount 0 exchangeAmount]
// 3                        [... sellerKey standardProgram sellerProgram requestedAsset actualAmount 0 exchangeAmount 3]
// ROLL                     [... sellerKey standardProgram sellerProgram actualAmount 0 exchangeAmount requestedAsset]
// 1                        [... sellerKey standardProgram sellerProgram actualAmount 0 exchangeAmount requestedAsset 1]
// 5                        [... sellerKey standardProgram sellerProgram actualAmount 0 exchangeAmount requestedAsset 1 5]
// ROLL                     [... sellerKey standardProgram actualAmount 0 exchangeAmount requestedAsset 1 sellerProgram]
// CHECKOUTPUT              [... sellerKey standardProgram actualAmount checkOutput(exchangeAmount, requestedAsset, sellerProgram)]
// VERIFY                   [... sellerKey standardProgram actualAmount]
// 1                        [... sellerKey standardProgram actualAmount 1]
// AMOUNT                   [... sellerKey standardProgram actualAmount 1 valueAmount]
// 2                        [... sellerKey standardProgram actualAmount 1 valueAmount 2]
// ROLL                     [... sellerKey standardProgram 1 valueAmount actualAmount]
// SUB                      [... sellerKey standardProgram 1 (valueAmount - actualAmount)]
// ASSET                    [... sellerKey standardProgram 1 (valueAmount - actualAmount) valueAsset]
// 1                        [... sellerKey standardProgram 1 (valueAmount - actualAmount) valueAsset 1]
// 4                        [... sellerKey standardProgram 1 (valueAmount - actualAmount) valueAsset 1 4]
// ROLL                     [... sellerKey 1 (valueAmount - actualAmount) valueAsset 1 standardProgram]
// CHECKOUTPUT              [... sellerKey checkOutput((valueAmount - actualAmount), valueAsset, standardProgram)]
// JUMP:$_end               [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
// $fullTrade               [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
// 6                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset 6]
// PICK                     [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset exchangeAmount]
// 3                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset exchangeAmount 3]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram ratioMolecule requestedAsset exchangeAmount ratioDenominator]
// MUL                      [... exchangeAmount sellerKey standardProgram sellerProgram ratioMolecule requestedAsset (exchangeAmount * ratioDenominator)]
// 2                        [... exchangeAmount sellerKey standardProgram sellerProgram ratioMolecule requestedAsset (exchangeAmount * ratioDenominator) 2]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset (exchangeAmount * ratioDenominator) ratioMolecule]
// DIV                      [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount]
// AMOUNT                   [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount]
// OVER                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount actualAmount]
// 0                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount actualAmount 0]
// GREATERTHAN              [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount > 0)]
// 2                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount > 0) 2]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset valueAmount (actualAmount > 0) actualAmount]
// 2                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset valueAmount (actualAmount > 0) actualAmount 2]
// ROLL                     [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset (actualAmount > 0) actualAmount valueAmount]
// EQUAL                    [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset (actualAmount > 0) (actualAmount == valueAmount)]
// BOOLAND                  [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset ((actualAmount > 0) && (actualAmount == valueAmount))]
// VERIFY                   [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset]
// 0                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset 0]
// 5                        [... exchangeAmount sellerKey standardProgram sellerProgram requestedAsset 0 5]
// ROLL                     [... sellerKey standardProgram sellerProgram requestedAsset 0 exchangeAmount]
// 2                        [... sellerKey standardProgram sellerProgram requestedAsset 0 exchangeAmount 2]
// ROLL                     [... sellerKey standardProgram sellerProgram 0 exchangeAmount requestedAsset]
// 1                        [... sellerKey standardProgram sellerProgram 0 exchangeAmount requestedAsset 1]
// 4                        [... sellerKey standardProgram sellerProgram 0 exchangeAmount requestedAsset 1 4]
// ROLL                     [... sellerKey standardProgram 0 exchangeAmount requestedAsset 1 sellerProgram]
// CHECKOUTPUT              [... sellerKey standardProgram checkOutput(exchangeAmount, requestedAsset, sellerProgram)]
// JUMP:$_end               [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
// $cancel                  [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset <clause selector>]
// DROP                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
// 6                        [... sellerSig sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset 6]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset sellerSig]
// 6                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset sellerSig 6]
// ROLL                     [... standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset sellerSig sellerKey]
// TXSIGHASH SWAP CHECKSIG  [... standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset checkTxSig(sellerKey, sellerSig)]
// $_end                    [... sellerKey standardProgram sellerProgram ratioDenominator ratioMolecule requestedAsset]
func MagneticProgram(assetComparedResult int, clauseSelector int64) ([]byte, error) {
	var firstPositionOP, secondPostionOP vm.Op
	switch {
	case assetComparedResult > 0:
		firstPositionOP = vm.OP_0
		secondPostionOP = vm.OP_1

		// the composition of magnetic contract transaction must comply with the rules:
		// the first input requestAsset must greater than lockedAsset,
		// and the second input requestAsset must less than lockedAsset
		magneticClauseSelector = 0
		if clauseSelector == 1 {
			magneticClauseSelector = 1
		}

	case assetComparedResult < 0:
		if magneticClauseSelector == 1 {
			firstPositionOP = vm.OP_1
			secondPostionOP = vm.OP_2
		} else {
			firstPositionOP = vm.OP_2
			secondPostionOP = vm.OP_3
		}

	default:
		return nil, errors.WithDetail(ErrBadValue, "the requestAssetID is same as lockedAssetID")
	}

	builder := NewBuilder()
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
	builder.AddOp(vm.OP_MUL)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_DIV)
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
	builder.AddOp(firstPositionOP)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_5)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(secondPostionOP)
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
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_MUL)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_DIV)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_OVER)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHAN)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_EQUAL)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(firstPositionOP)
	builder.AddOp(vm.OP_5)
	builder.AddOp(vm.OP_ROLL)
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
