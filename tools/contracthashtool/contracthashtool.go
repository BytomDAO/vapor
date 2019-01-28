package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto"
	"github.com/vapor/crypto/ed25519/chainkd"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/equity/pegin_contract"
	"github.com/vapor/util"
)

var (
	fedpegXPubs    string
	fedpegXPrv     string
	claimScriptStr string
	mode           = uint16(0)
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "pegin contract tool",
	Run:   run,
}

func init() {
	//runCmd.PersistentFlags().Uint64Var(&startHeight, "start_height", 0, "Start monitoring block height for transactions")
	runCmd.PersistentFlags().StringVar(&fedpegXPubs, "fedpeg_xpubs", "", "Change federated peg to use a different xpub.Use Comma Delimiters.")
	runCmd.PersistentFlags().StringVar(&fedpegXPrv, "xprv", "", "Generates one of the private keys corresponding to the payment contract address.")
	runCmd.PersistentFlags().Uint16Var(&mode, "mode", 0, "0: generates the contract address for the payment  1: generate the private key corresponding to the payment contract address.")
	runCmd.PersistentFlags().StringVar(&claimScriptStr, "claim_script", "", "Redemption of the script.")
}

func run(cmd *cobra.Command, args []string) {
	if mode == 0 {
		if fedpegXPubs == "" {
			cmn.Exit(cmn.Fmt("OH GOD WHAT DID YOU DO? fedpeg_xpubs is empty."))
		}
		var federationRedeemXPubs []chainkd.XPub
		fedpegXPubs := strings.Split(fedpegXPubs, ",")
		for _, xpubStr := range fedpegXPubs {
			var xpub chainkd.XPub
			xpub.UnmarshalText([]byte(xpubStr))
			federationRedeemXPubs = append(federationRedeemXPubs, xpub)
		}
		consensus.ActiveNetParams.FedpegXPubs = federationRedeemXPubs
		if claimScriptStr == "" {
			cmn.Exit(cmn.Fmt("OH GOD WHAT DID YOU DO? claim_script is empty."))
		}
		var claimScript chainjson.HexBytes
		claimScript.UnmarshalText([]byte(claimScriptStr))
		peginContractPrograms, err := pegin_contract.GetPeginContractPrograms(claimScript)
		if err != nil {
			cmn.Exit(cmn.Fmt("GetPeginContractPrograms returns an error, %v", err))
		}
		scriptHash := crypto.Sha256(peginContractPrograms)
		address, err := common.NewPeginAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
		if err != nil {
			cmn.Exit(cmn.Fmt("NewPeginAddressWitnessScriptHash returns an error, %v", err))
		}
		fmt.Println("contract address:", address.EncodeAddress())
		fmt.Println("claim_script:", claimScriptStr)

	} else if mode == 1 {
		if fedpegXPrv == "" {
			cmn.Exit(cmn.Fmt("OH GOD WHAT DID YOU DO? xprv is empty."))
		}
		if claimScriptStr == "" {
			cmn.Exit(cmn.Fmt("OH GOD WHAT DID YOU DO? claim_script is empty."))
		}
		var claimScript chainjson.HexBytes
		claimScript.UnmarshalText([]byte(claimScriptStr))

		var xprv chainkd.XPrv
		xprv.UnmarshalText([]byte(fedpegXPrv))
		xpub := xprv.XPub()
		// pub + scriptPubKey 生成一个随机数A
		var tmp [32]byte
		h := hmac.New(sha256.New, xpub[:])
		h.Write(claimScript)
		tweak := h.Sum(tmp[:])
		// pub +  A 生成一个新的公钥pub_new
		privateKey := xprv.Child(tweak, false)
		fmt.Println("New secret key: ", privateKey.String())
		fmt.Println("claim_script:", claimScriptStr)
	} else {
		cmn.Exit(cmn.Fmt("OH GOD WHAT DID YOU DO?"))
	}
}

func main() {
	if _, err := runCmd.ExecuteC(); err != nil {
		os.Exit(util.ErrLocalExe)
	}
}
