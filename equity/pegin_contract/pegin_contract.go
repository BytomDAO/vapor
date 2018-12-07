package pegin_contract

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/equity/compiler"
)

const module = "pegin_contract"

var lockWith2of3KeysFmt = `
contract LockWith2of3Keys(%s) locks amount of asset {
  clause unlockWith2Sigs(%s) {
    verify checkTxMultiSig(%s)
    unlock amount of asset
  }
}
`

func GetPeginContractPrograms(claimScript []byte) ([]byte, error) {

	pubkeys := getNewXpub(consensus.ActiveNetParams.FedpegXPubs, claimScript)
	num := len(pubkeys)
	if num == 0 {
		return nil, errors.New("Fedpeg's XPubs is empty")
	}
	params := ""
	unlockParams := ""
	checkParams := "["

	for index := 0; index < num; index++ {
		param := fmt.Sprintf("pubkey%d", index+1)
		params += param
		checkParams += param
		if (index + 1) < num {
			params += ","
			checkParams += ","
		}
	}
	params += ": PublicKey"
	checkParams += "],["

	signNum := getNumberOfSignaturesRequired(pubkeys)
	for index := 0; index < signNum; index++ {
		param := fmt.Sprintf("sig%d", index+1)
		unlockParams += param
		checkParams += param
		if index+1 < signNum {
			unlockParams += ","
			checkParams += ","
		}
	}

	unlockParams += ": Signature"
	checkParams += "]"

	lockWith2of3Keys := fmt.Sprintf(lockWith2of3KeysFmt, params, unlockParams, checkParams)
	r := strings.NewReader(lockWith2of3Keys)
	compiled, err := compiler.Compile(r)
	if err != nil {
		return nil, errors.New("Compile contract failed")
	}

	contract := compiled[len(compiled)-1]

	if num < len(contract.Params) {
		return nil, errors.New("Compile contract failed")
	}

	contractArgs, err := convertArguments(contract, pubkeys)
	if err != nil {
		log.WithFields(log.Fields{"module": module, "error": err}).Error("Convert arguments into contract parameters error")
		return nil, errors.New("Convert arguments into contract parameters error")
	}

	instantProg, err := instantiateContract(contract, contractArgs)
	if err != nil {
		log.WithFields(log.Fields{"module": module, "error": err}).Error("Instantiate contract error")
		return nil, errors.New("Instantiate contract error")
	}

	return instantProg, nil
}

func getNewXpub(federationRedeemXPub []chainkd.XPub, claimScript []byte) []ed25519.PublicKey {

	var pubkeys []ed25519.PublicKey
	for _, xpub := range federationRedeemXPub {
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(claimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		chaildXPub := xpub.Child(tweak)
		pubkeys = append(pubkeys, chaildXPub.PublicKey())
	}
	return pubkeys
}

func getNumberOfSignaturesRequired(pubkeys []ed25519.PublicKey) int {
	return len(pubkeys)/2 + 1
}

// InstantiateContract instantiate contract parameters
func instantiateContract(contract *compiler.Contract, args []compiler.ContractArg) ([]byte, error) {
	program, err := compiler.Instantiate(contract.Body, contract.Params, contract.Recursive, args)
	if err != nil {
		return nil, err
	}

	return program, nil
}

func convertArguments(contract *compiler.Contract, args []ed25519.PublicKey) ([]compiler.ContractArg, error) {
	var contractArgs []compiler.ContractArg
	for i, p := range contract.Params {
		var argument compiler.ContractArg
		switch p.Type {
		case "PublicKey":
			argument.S = (*chainjson.HexBytes)(&args[i])
		default:
			return nil, errors.New("Contract parameter type error")
		}
		contractArgs = append(contractArgs, argument)
	}

	return contractArgs, nil
}
