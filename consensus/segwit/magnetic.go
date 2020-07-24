package segwit

import (
	"errors"

	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/vm"
	"github.com/bytom/vapor/protocol/vm/vmutil"
)

const (
	magneticV1 = iota + 1
	magneticV2
)

// isMagneticScript is used to determine whether it is a Magnetic script with specific version or not
func isMagneticScript(prog []byte, version int) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	if len(insts) != 6 {
		return false
	}

	switch version {
	case magneticV1:
		if insts[0].Op != vm.OP_0 {
			return false
		}
	case magneticV2:
		if insts[0].Op != vm.OP_1 {
			return false
		}
	default:
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

// IsP2WMCScript is used to determine whether it is the v1 P2WMC script or not
func IsP2WMCScript(prog []byte) bool {
	return isMagneticScript(prog, magneticV1)
}

// IsP2WMCScriptV2 is used to determine whether it is the v2 P2WMC script or not
func IsP2WMCScriptV2(prog []byte) bool {
	return isMagneticScript(prog, magneticV2)
}

// DecodeP2WMCProgram parse standard P2WMC arguments to magneticContractArgs
func DecodeP2WMCProgram(prog []byte) (*vmutil.MagneticContractArgs, error) {
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

// ConvertP2MCProgram convert standard P2WMC program into the v1 P2MC program
func ConvertP2MCProgram(prog []byte) ([]byte, error) {
	if !IsP2WMCScript(prog) {
		return nil, errors.New("invalid the v1 of magnetic P2MC program")
	}

	magneticContractArgs, err := DecodeP2WMCProgram(prog)
	if err != nil {
		return nil, err
	}

	return vmutil.P2MCProgram(*magneticContractArgs)
}

// ConvertP2MCProgramV2 convert standard P2WMC program into the v2 P2MC program
func ConvertP2MCProgramV2(prog []byte) ([]byte, error) {
	if !IsP2WMCScriptV2(prog) {
		return nil, errors.New("invalid the v2 of magnetic P2MC program")
	}

	magneticContractArgs, err := DecodeP2WMCProgram(prog)
	if err != nil {
		return nil, err
	}

	return vmutil.P2MCProgramV2(*magneticContractArgs)
}
