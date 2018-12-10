package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type RespMsg struct {
	MainchainAddress string `json:"mainchain_address,omitempty"`
	ControlProgram   string `json:"control_program,omitempty"`
	ClaimScript      string `json:"claim_script,omitempty"`
}

type Response struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data string `json:"data,omitempty"`
}

func getPeginInfo() (map[string]string, error) {
	resp, err := http.Get("http://127.0.0.1:8000/api/get_pegin_address")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("connect fail")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rep := Response{}
	err = json.Unmarshal(body, &rep)
	if err != nil {
		return nil, err
	}
	var msg []RespMsg
	err = json.Unmarshal([]byte(rep.Data), &msg)
	if err != nil {
		return nil, err
	}
	mapMsg := make(map[string]string)
	for _, m := range msg {
		mapMsg[m.ClaimScript] = m.ControlProgram
	}
	return mapMsg, nil
}
