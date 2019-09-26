package segwit

import (
	"errors"

	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
)

// IsP2WScript is used to determine whether it is a P2WScript or not
func IsP2WScript(prog []byte) bool {
	return IsP2WPKHScript(prog) || IsP2WSHScript(prog) || IsStraightforward(prog)
}

// IsStraightforward is used to determine whether it is a Straightforward script or not
func IsStraightforward(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 1 {
		return false
	}
	return insts[0].Op == vm.OP_TRUE || insts[0].Op == vm.OP_FAIL
}

// IsP2WPKHScript is used to determine whether it is a P2WPKH script or not
func IsP2WPKHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_20 && len(insts[1].Data) == consensus.PayToWitnessPubKeyHashDataSize
}

// IsP2WSHScript is used to determine whether it is a P2WSH script or not
func IsP2WSHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_32 && len(insts[1].Data) == consensus.PayToWitnessScriptHashDataSize
}

// IsP2WDCScript is used to determine whether it is a P2WDC script or not
func IsP2WDCScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 6 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}

	if _, err = vm.AsInt64(insts[2].Data); err != nil {
		return false
	}

	if _, err = vm.AsInt64(insts[3].Data); err != nil {
		return false
	}

	return insts[1].Op == vm.OP_DATA_20 && len(insts[1].Data) == 32 && insts[5].Op == vm.OP_DATA_20 && len(insts[5].Data) == 32
}

// ConvertP2PKHSigProgram convert standard P2WPKH program into P2PKH program
func ConvertP2PKHSigProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2PKHSigProgram(insts[1].Data)
	}
	return nil, errors.New("unknow P2PKH version number")
}

// ConvertP2SHProgram convert standard P2WSH program into P2SH program
func ConvertP2SHProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2SHProgram(insts[1].Data)
	}
	return nil, errors.New("unknow P2SHP version number")
}

// ConvertP2DCProgram convert standard P2WDC program into P2DC program
func ConvertP2DCProgram(prog []byte, lockedAssetID bc.AssetID, args [][]byte) ([]byte, error) {
	dexContractArgs, err := DecodeP2DCProgram(prog)
	if err != nil {
		return nil, err
	}

	if len(args) != 2 {
		errors.New("Invalid P2DC arguments")
	}

	clauseSelector, err := vm.AsInt64(args[1])
	if err != nil {
		return nil, err
	}

	return vmutil.P2DCProgram(*dexContractArgs, lockedAssetID, clauseSelector)
}

// DecodeP2DCProgram parse standard P2WDC arguments to DexContractArgs
func DecodeP2DCProgram(prog []byte) (*vmutil.DexContractArgs, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	if !(len(insts) == 6 && insts[0].Op == vm.OP_0) {
		return nil, errors.New("Invalid P2DC program")
	}

	dexContractArgs := &vmutil.DexContractArgs{}
	requestedAsset := [32]byte{}
	copy(requestedAsset[:], insts[1].Data)
	dexContractArgs.RequestedAsset = bc.NewAssetID(requestedAsset)

	if dexContractArgs.RatioMolecule, err = vm.AsInt64(insts[2].Data); err != nil {
		return nil, err
	}

	if dexContractArgs.RatioDenominator, err = vm.AsInt64(insts[3].Data); err != nil {
		return nil, err
	}

	dexContractArgs.SellerProgram = insts[4].Data
	dexContractArgs.SellerKey = insts[5].Data
	return dexContractArgs, nil
}

// GetHashFromStandardProg get hash from standard program
func GetHashFromStandardProg(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[1].Data, nil
}
