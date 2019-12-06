package common

import (
	"errors"

	"github.com/bytom/vapor/common"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/protocol/vm/vmutil"
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

func GetControlProgramFromAddress(address string) ([]byte, error) {
	decodeaddress, err := common.DecodeAddress(address, &consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}

	redeemContract := decodeaddress.ScriptAddress()
	program := []byte{}
	switch decodeaddress.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return nil, errors.New("Invalid address")
	}
	if err != nil {
		return nil, err
	}
	return program, nil
}
