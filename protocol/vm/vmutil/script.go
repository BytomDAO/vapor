package vmutil

import (
	"strings"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/vm"
)

// pre-define errors
var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

// DexContractArgs is a struct for dex contract arguments
type DexContractArgs struct {
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

// P2WDCProgram return the segwit pay to dex contract
func P2WDCProgram(dexContractArgs DexContractArgs) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(dexContractArgs.RequestedAsset.Bytes())
	builder.AddInt64(dexContractArgs.RatioMolecule)
	builder.AddInt64(dexContractArgs.RatioDenominator)
	builder.AddData(dexContractArgs.SellerProgram)
	builder.AddData(dexContractArgs.SellerKey)
	return builder.Build()
}

// P2DCProgram generates the script for control with dex contract
func P2DCProgram(dexContractArgs DexContractArgs, lockedAssetID bc.AssetID) ([]byte, error) {
	standardProgram, err := P2WDCProgram(dexContractArgs)
	if err != nil {
		return nil, err
	}

	dexProgram, err := DexProgram(strings.Compare(dexContractArgs.RequestedAsset.String(), lockedAssetID.String()))
	if err != nil {
		return nil, err
	}

	builder := NewBuilder()
	builder.AddData(dexContractArgs.SellerKey)
	builder.AddData(standardProgram)
	builder.AddData(dexContractArgs.SellerProgram)
	builder.AddInt64(dexContractArgs.RatioDenominator)
	builder.AddInt64(dexContractArgs.RatioMolecule)
	builder.AddData(dexContractArgs.RequestedAsset.Bytes())
	builder.AddOp(vm.OP_DEPTH)
	builder.AddData(dexProgram)
	builder.AddOp(vm.OP_FALSE)
	builder.AddOp(vm.OP_CHECKPREDICATE)
	return builder.Build()
}

// DexProgram is the actual execute dex program which not contain arguments
func DexProgram(assetComparedResult int) ([]byte, error) {
	var firstPositionOP, secondPostionOP vm.Op
	if assetComparedResult == 0 {
		return nil, errors.WithDetail(ErrBadValue, "the requestAssetID is same as lockedAssetID")
	} else if assetComparedResult > 0 {
		firstPositionOP = vm.OP_0
		secondPostionOP = vm.OP_1
	} else {
		firstPositionOP = vm.OP_2
		secondPostionOP = vm.OP_3
	}

	builder := NewBuilder()
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddJumpIf(0)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_MUL)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_DIV)
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHAN)
	builder.AddOp(vm.OP_8)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_LESSTHANOREQUAL)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_MUL)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_DIV)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_OVER)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHAN)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_LESSTHANOREQUAL)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_LESSTHAN)
	builder.AddOp(vm.OP_NOT)
	builder.AddOp(vm.OP_NOP)
	builder.AddJumpIf(1)
	builder.AddOp(firstPositionOP)
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(secondPostionOP)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_SUB)
	builder.AddOp(vm.OP_ASSET)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_8)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddJump(2)
	builder.SetJumpTarget(1)
	builder.AddOp(firstPositionOP)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	builder.AddOp(vm.OP_VERIFY)
	builder.SetJumpTarget(2)
	builder.AddJump(3)
	builder.SetJumpTarget(0)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	builder.SetJumpTarget(3)
	return builder.Build()
}
