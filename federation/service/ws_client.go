package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	apiPath = "/websocket-subscribe"

	// TopicNotifyNewTransactions is a topic can be subscribed, when a new valid transaction is incoming, the client will be notified.
	TopicNotifyNewTransactions = "notify_new_transactions"

	// ResponseNewTransaction is a notification type indicate a new transaction is incomming.
	ResponseNewTransaction = "new_transaction"
)

// WSClient establish a websocket connection with websocket server,
// which can subscribe topics, and receive the corresponding message.
type WSClient struct {
	conn            *websocket.Conn
	processCh       chan *WSResponse
	remoteAddr      string
	closed          bool
	subscribeTopics []string
}

// NewWSClient create a new websocket client
func NewWSClient(addr string, processCh chan *WSResponse) *WSClient {
	return &WSClient{remoteAddr: addr, processCh: processCh}
}

// Connect establish a websocket connection with websocket server,
// and listen to the arrive message
func (w *WSClient) Connect() error {
	if err := w.tryConnect(); err != nil {
		return err
	}

	go w.listen()
	return nil
}

func (w *WSClient) tryConnect() error {
	u := url.URL{Scheme: "ws", Host: w.remoteAddr, Path: apiPath}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	w.conn = conn
	return err
}

func (w *WSClient) reconnect() bool {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if w.closed {
			return false
		}
		if err := w.tryConnect(); err != nil {
			log.WithField("err", err).Error("reconnect websocket server fail")
			continue
		}
		for _, topic := range w.subscribeTopics {
			if err := w.Subscribe(topic); err != nil {
				log.WithField("err", err).WithField("topic", topic).Error("subscribe topic fail")
				return false
			}
		}
		return true
	}
	return false
}

// Close remove the websocket connection
func (w *WSClient) Close() error {
	w.closed = true
	return w.conn.Close()
}

// wsRequest means the data structure of the request
type wsRequest struct {
	Topic string `json:"topic"`
}

// WSResponse means the returned data structure
type WSResponse struct {
	NotificationType string          `json:"notification_type"`
	Data             json.RawMessage `json:"data"`
	ErrorDetail      string          `json:"error_detail,omitempty"`
}

func (w *WSClient) Subscribe(topic string) error {
	if w.conn == nil {
		return errors.New("must connect the ws server before subscribe")
	}
	req := &wsRequest{Topic: topic}
	msg, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if err := w.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return err
	}

	w.subscribeTopics = append(w.subscribeTopics, topic)
	return nil
}

func (w *WSClient) listen() {
	for {
		_, msg, err := w.conn.ReadMessage()
		if err != nil {
			log.WithField("err", err).Error("read message error")
			if w.closed {
				break
			}
			switch err.(type) {
			case *websocket.CloseError:
				if w.reconnect() {
					continue
				}
				break
			default:
				continue
			}
		}

		resp := &WSResponse{}
		if err = json.Unmarshal(msg, resp); err != nil {
			log.WithField("err", err).Error("Unmarshal error")
			continue
		}

		w.processCh <- resp
	}
}
