package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/vapor/crypto/ed25519"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/equity/compiler"
	equityutil "github.com/vapor/equity/equity/util"
	"github.com/vapor/errors"
)

func ExportFederationFile(fedFile string, config *Config) error {
	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config.Federation); err != nil {
		return err
	}

	return ioutil.WriteFile(fedFile, buf.Bytes(), 0644)
}

func LoadFederationFile(fedFile string, config *Config) error {
	file, err := os.Open(fedFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config.Federation)
}

var lockWithKeysFmt = `
contract LockWithKeys(%s: PublicKey) locks amount of asset {
	clause unlockWith2Sigs(%s: Signature) {
		verify checkTxMultiSig([%s],[%s])
		unlock amount of asset
	}
}
`

func generateFederationContract(pubkeys []ed25519.PublicKey, quorum int) (string, error) {
	num := len(pubkeys)
	if num == 0 {
		return "", errors.New("federation's XPubs is empty")
	}

	if quorum < len(pubkeys)/2+1 || quorum > num {
		return "", errors.New("The quorum with multiple contracts in the federation is incorrect")
	}

	params := ""
	unlockParams := ""
	checkPubkeysParams := ""
	checkSigsParams := ""

	for index := 0; index < num; index++ {
		param := fmt.Sprintf("pubkey%d", index+1)
		params += param
		checkPubkeysParams += param
		if (index + 1) < num {
			params += ","
			checkPubkeysParams += ","
		}
	}

	for index := 0; index < quorum; index++ {
		param := fmt.Sprintf("sig%d", index+1)
		unlockParams += param
		checkSigsParams += param
		if index+1 < quorum {
			unlockParams += ","
			checkSigsParams += ","
		}
	}

	lockWithKeys := fmt.Sprintf(lockWithKeysFmt, params, unlockParams, checkPubkeysParams, checkSigsParams)
	return lockWithKeys, nil
}

func GetFederationContractPrograms(pubkeys []ed25519.PublicKey, quorum int) ([]byte, error) {
	lockWithKeys, err := generateFederationContract(pubkeys, quorum)
	if err != nil {
		return nil, errors.Wrap(err, "Failed generate Federation Contract")
	}

	r := strings.NewReader(lockWithKeys)
	compiled, err := compiler.Compile(r)
	if err != nil {
		return nil, errors.Wrap(err, "Compile contract failed")
	}

	contract := compiled[len(compiled)-1]

	if len(pubkeys) < len(contract.Params) {
		return nil, errors.New("Compile contract failed")
	}

	contractArgs, err := convertArguments(contract, pubkeys)
	if err != nil {
		return nil, errors.Wrap(err, "Convert arguments into contract parameters error")
	}

	instantProg, err := equityutil.InstantiateContract(contract, contractArgs)
	if err != nil {
		return nil, errors.Wrap(err, "Instantiate contract error")
	}

	return instantProg, nil
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
