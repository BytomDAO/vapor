package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/vapor/config"
	chainjson "github.com/vapor/encoding/json"
	bytomtypes "github.com/vapor/protocol/bc/types/bytom/types"
	"github.com/vapor/util"
)

var lock sync.Mutex
var gClaimTxMap map[uint64]claimTx
var currentHeight uint64

type WSRequest struct {
	Topic string `json:"topic"`
}

type WSResponse struct {
	NotificationType string      `json:"notification_type"`
	Data             interface{} `json:"data"`
	ErrorDetail      string      `json:"error_detail,omitempty"`
}

type claimTx struct {
	Password     string                 `json:"password"`
	RawTx        bytomtypes.Tx          `json:"raw_transaction"`
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
	ClaimScript  chainjson.HexBytes     `json:"claim_script"`
}

var startHeight uint64 = 0

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "monitor claim tx",
	Run:   run,
}

func init() {
	runCmd.PersistentFlags().Uint64Var(&startHeight, "start_height", 0, "Start monitoring block height for transactions")
}

func run(cmd *cobra.Command, args []string) {

}

func main() {

	if _, err := runCmd.ExecuteC(); err != nil {
		os.Exit(util.ErrLocalExe)
	}

	gClaimTxMap = make(map[uint64]claimTx)
	currentHeight = 0
	client := &WSClient{}
	if err := client.New("127.0.0.1:9888"); err != nil {
		log.Println(err)
		return
	}
	go sendClaimTx()
	go getRawTransactionWithHeight()
	req := WSRequest{
		Topic: "notify_raw_blocks",
	}
	client.SendData(req)

	for {
		msg, err := client.RecvData()
		if err != nil {
			fmt.Println(err)
			break
		}
		var rep WSResponse
		err = json.Unmarshal(msg, &rep)
		if err != nil {
			log.Printf("Unmarshal error: %v", err)
		}

		block := &bytomtypes.Block{}
		switch rep.NotificationType {
		case "raw_blocks_connected":
			data := fmt.Sprint(rep.Data)
			err = block.UnmarshalText([]byte(data))
			if err != nil {
				block = nil
			}
		case "raw_blocks_disconnected":
			data := fmt.Sprint(rep.Data)
			err = block.UnmarshalText([]byte(data))
			if err != nil {
				block = nil
			}
		case "request_status":
			if rep.ErrorDetail != "" {
				log.Println(rep.ErrorDetail)
			}
			block = nil
		default:
			block = nil
		}
		if block != nil {
			currentHeight = block.Height
			err := getRawTransaction(block)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func sendClaimTx() {
	for {
		// 存储tx到数据库、列表中
		for k, v := range gClaimTxMap {
			if k <= currentHeight {
				resp, exitCode := util.ClientCall("/claim-pegin-transaction", &v)
				if exitCode != util.Success {
					lock.Lock()
					delete(gClaimTxMap, k)
					lock.Unlock()
					continue
				}
				type txID struct {
					Txid string `json:"tx_id"`
				}
				var out txID
				restoreStruct(resp, &out)
				lock.Lock()
				delete(gClaimTxMap, k)
				lock.Unlock()
				fmt.Println(out.Txid)
				time.Sleep(3 * time.Second)
			}
		}
	}

}

func restoreStruct(data interface{}, out interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if ok != true {
		fmt.Println("invalid type assertion")
		os.Exit(util.ErrLocalParse)
	}

	rawData, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(util.ErrLocalParse)
	}
	json.Unmarshal(rawData, out)
}

func getRawTransaction(block *bytomtypes.Block) error {
	peginInfo, err := getPeginInfo()
	if err != nil {
		return err
	}
	for _, tx := range block.Transactions {
		for _, output := range tx.Outputs {
			for k, v := range peginInfo {
				var claimScript chainjson.HexBytes
				if err := claimScript.UnmarshalText([]byte(k)); err != nil {
					return err
				}
				var controlProgram chainjson.HexBytes
				if err := controlProgram.UnmarshalText([]byte(v)); err != nil {
					return err
				}

				if !bytes.Equal(output.ControlProgram, controlProgram) {
					continue
				}
				blockHash := block.Hash()
				merkleProof, err := getMerkleProof(blockHash.String(), tx.ID.String())
				if err != nil {
					return err
				}
				tmp := claimTx{
					Password:     "123456",
					RawTx:        *tx,
					BlockHeader:  merkleProof.BlockHeader,
					TxHashes:     merkleProof.TxHashes,
					StatusHashes: merkleProof.StatusHashes,
					Flags:        merkleProof.Flags,
					MatchedTxIDs: merkleProof.MatchedTxIDs,
					ClaimScript:  claimScript,
				}
				// 存储tx到数据库、列表中
				height := block.Height + 6
				lock.Lock()
				gClaimTxMap[height] = tmp
				lock.Unlock()
			}
		}
	}

	return nil
}

func getRawTransactionWithHeight() {
	for {
		if currentHeight > 0 {
			num := currentHeight - startHeight
			for i := uint64(0); i < num; i++ {
				block, err := getBlockWithHeight(startHeight)
				if err != nil {
					log.Fatal(err)
				}
				err = getRawTransaction(&block)
				if err != nil {
					log.Fatal(err)
				}
				startHeight += 1
			}
			if startHeight >= currentHeight {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
}

type MerkleBlockResp struct {
	BlockHeader  bytomtypes.BlockHeader `json:"block_header"`
	TxHashes     []chainjson.HexBytes   `json:"tx_hashes"`
	StatusHashes []chainjson.HexBytes   `json:"status_hashes"`
	Flags        []uint32               `json:"flags"`
	MatchedTxIDs []chainjson.HexBytes   `json:"matched_tx_ids"`
}

func getMerkleProof(blockHash string, txId string) (MerkleBlockResp, error) {
	//body_json = {"tx_id": tx_id,"block_hash": block_hash}
	type Req struct {
		TxID      string `json:"tx_id"`
		BlockHash string `json:"block_hash"`
	}
	util.MainchainConfig = &config.MainChainRpcConfig{
		MainchainRpcHost: "127.0.0.1",
		MainchainRpcPort: "9888",
	}
	var blockHeader MerkleBlockResp
	resp, err := util.CallRPC("/get-merkle-proof", &Req{TxID: txId, BlockHash: blockHash})
	if err != nil {
		return blockHeader, err
	}
	tmp, _ := json.Marshal(resp)

	json.Unmarshal(tmp, &blockHeader)

	return blockHeader, nil
}

func getBlockWithHeight(blockHeight uint64) (bytomtypes.Block, error) {
	type Req struct {
		BlockHeight uint64 `json:"block_height"`
	}
	util.MainchainConfig = &config.MainChainRpcConfig{
		MainchainRpcHost: "127.0.0.1",
		MainchainRpcPort: "9888",
	}
	type RawBlockResp struct {
		RawBlock *bytomtypes.Block `json:"raw_block"`
	}
	var block RawBlockResp
	resp, err := util.CallRPC("/get-raw-block", &Req{BlockHeight: blockHeight})
	if err != nil {
		return *block.RawBlock, err
	}
	tmp, _ := json.Marshal(resp)

	json.Unmarshal(tmp, &block)
	return *block.RawBlock, nil
}
