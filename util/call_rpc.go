package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/config"
	jww "github.com/spf13/jwalterweatherman"
)

var MainchainConfig *config.MainChainRpcConfig
var ValidatePegin bool

// Response describes the response standard.
type Response struct {
	Status      string      `json:"status,omitempty"`
	Code        string      `json:"code,omitempty"`
	Msg         string      `json:"msg,omitempty"`
	ErrorDetail string      `json:"error_detail,omitempty"`
	Data        interface{} `json:"data,omitempty"`
}

// CallRPC call api.
func CallRPC(path string, req ...interface{}) (interface{}, error) {

	host := MainchainConfig.MainchainRpcHost
	port, _ := strconv.ParseUint(MainchainConfig.MainchainRpcPort, 10, 16)
	token := MainchainConfig.MainchainToken

	var resp = &Response{}
	var request interface{}

	if req != nil {
		request = req[0]
	}

	// TODO主链的ip port token
	rpcURL := fmt.Sprintf("http://%s:%d", host, port)
	client := &rpc.Client{BaseURL: rpcURL}
	client.AccessToken = token
	client.Call(context.Background(), path, request, resp)
	switch resp.Status {
	case "fail":
		jww.ERROR.Println(resp.Msg)
		return nil, errors.New("fail")
	case "":
		jww.ERROR.Println("Unable to connect to the bytomd")
		return nil, errors.New("")
	}
	return resp.Data, nil
}

func IsConfirmedBytomBlock(txHeight uint64, nMinConfirmationDepth uint64) error {
	data, exitCode := CallRPC("get-block-count")
	if exitCode != nil {
		return exitCode
	}
	type blockHeight struct {
		BlockCount uint64 `json:"block_count"`
	}
	var mainchainHeight blockHeight
	tmp, _ := json.Marshal(data)
	json.Unmarshal(tmp, &mainchainHeight)
	if mainchainHeight.BlockCount < txHeight || (mainchainHeight.BlockCount-txHeight) < nMinConfirmationDepth {
		return errors.New("Peg-in bytom transaction needs more confirmations to be sent")
	}
	return nil
}
