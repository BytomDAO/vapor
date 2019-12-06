package segwit

import (
	"errors"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/protocol/vm/vmutil"
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

// IsP2WMCScript is used to determine whether it is a P2WMC script or not
func IsP2WMCScript(prog []byte) bool {
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

	if insts[1].Op != vm.OP_DATA_32 || len(insts[1].Data) != 32 {
		return false
	}

	if !(insts[2].IsPushdata() && insts[3].IsPushdata() && insts[4].IsPushdata()) {
		return false
	}

	if _, err = vm.AsInt64(insts[2].Data); err != nil {
		return false
	}

	if _, err = vm.AsInt64(insts[3].Data); err != nil {
		return false
	}

	if !IsP2WScript(insts[4].Data) {
		return false
	}

	return insts[5].Op == vm.OP_DATA_32 && len(insts[5].Data) == 32
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

// ConvertP2MCProgram convert standard P2WMC program into P2MC program
func ConvertP2MCProgram(prog []byte) ([]byte, error) {
	magneticContractArgs, err := DecodeP2WMCProgram(prog)
	if err != nil {
		return nil, err
	}
	return vmutil.P2MCProgram(*magneticContractArgs)
}

// DecodeP2WMCProgram parse standard P2WMC arguments to magneticContractArgs
func DecodeP2WMCProgram(prog []byte) (*vmutil.MagneticContractArgs, error) {
	if !IsP2WMCScript(prog) {
		return nil, errors.New("invalid P2MC program")
	}

	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	magneticContractArgs := &vmutil.MagneticContractArgs{
		SellerProgram: insts[4].Data,
		SellerKey:     insts[5].Data,
	}
	requestedAsset := [32]byte{}
	copy(requestedAsset[:], insts[1].Data)
	magneticContractArgs.RequestedAsset = bc.NewAssetID(requestedAsset)

	if magneticContractArgs.RatioNumerator, err = vm.AsInt64(insts[2].Data); err != nil {
		return nil, err
	}

	if magneticContractArgs.RatioDenominator, err = vm.AsInt64(insts[3].Data); err != nil {
		return nil, err
	}

	return magneticContractArgs, nil
}

// GetHashFromStandardProg get hash from standard program
func GetHashFromStandardProg(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[1].Data, nil
}
