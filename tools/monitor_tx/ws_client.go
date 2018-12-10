package main

import (
	"encoding/json"
	"net/url"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	Conn *websocket.Conn
}

func (WS *WSClient) New(host string) error {
	u := url.URL{Scheme: "ws", Host: host, Path: "/websocket-subscribe"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	WS.Conn = c
	return nil
}

func (WS *WSClient) SendData(req interface{}) {
	msg, _ := json.Marshal(req)
	WS.Conn.WriteMessage(websocket.TextMessage, msg)
}

func (WS *WSClient) RecvData() ([]byte, error) {
	_, msg, err := WS.Conn.ReadMessage()
	return msg, err
}
