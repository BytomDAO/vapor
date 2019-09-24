package segwit

import (
	"errors"

	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
)

func IsP2WScript(prog []byte) bool {
	return IsP2WPKHScript(prog) || IsP2WSHScript(prog) || IsStraightforward(prog)
}

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
	return insts[1].Op == vm.OP_DATA_20 && len(insts[1].Data) == 32 && insts[5].Op == vm.OP_DATA_20 && len(insts[5].Data) == 32
}

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

func ConvertP2DCProgram(prog []byte, lockedAssetID bc.AssetID) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	if insts[0].Op == vm.OP_0 {
		dexContractArgs := vmutil.DexContractArgs{}
		var requestedAsset [32]byte
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
		return vmutil.P2DCProgram(dexContractArgs, lockedAssetID)
	}
	return nil, errors.New("unknow P2DC version number")
}

func GetHashFromStandardProg(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[1].Data, nil
}
