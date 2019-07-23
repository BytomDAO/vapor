package common

import (
	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/protocol/vm/vmutil"
)

func GetAddressFromControlProgram(prog []byte) string {
	if segwit.IsP2WPKHScript(prog) {
		if pubHash, err := segwit.GetHashFromStandardProg(prog); err == nil {
			return buildP2PKHAddress(pubHash)
		}
	} else if segwit.IsP2WSHScript(prog) {
		if scriptHash, err := segwit.GetHashFromStandardProg(prog); err == nil {
			return buildP2SHAddress(scriptHash)
		}
	}

	return ""
}

func buildP2PKHAddress(pubHash []byte) string {
	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		return ""
	}

	return address.EncodeAddress()
}

func buildP2SHAddress(scriptHash []byte) string {
	address, err := common.NewAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
	if err != nil {
		return ""
	}

	return address.EncodeAddress()
}

func GetControlProgramFromAddress(address string) []byte {
	decodeaddress, err := common.DecodeAddress(address, &consensus.ActiveNetParams)
	if err != nil {
		return nil
	}

	redeemContract := decodeaddress.ScriptAddress()
	program := []byte{}
	switch decodeaddress.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return nil
	}
	if err != nil {
		return nil
	}
	return program
}
