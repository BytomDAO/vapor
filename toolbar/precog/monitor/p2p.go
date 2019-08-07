package monitor

import (
	"sync"
	// "io/ioutil"
	// "os"
	// "time"

	cmn "github.com/tendermint/tmlibs/common"
	// "github.com/jinzhu/gorm"
	// log "github.com/sirupsen/logrus"
	// dbm "github.com/vapor/database/leveldb"

	// cfg "github.com/vapor/config"
	"github.com/vapor/p2p"
	conn "github.com/vapor/p2p/connection"
	// "github.com/vapor/p2p/signlib"
	// "github.com/vapor/toolbar/precog/config"
	// "github.com/vapor/toolbar/precog/database/orm"
)

// TODO: why foo&bar
// TODO: what is testReactor
// TODO: listen on right port
// TODO: why not discover
func initSwitchFunc(sw *p2p.Switch) *p2p.Switch {
	// Make two reactors of two channels each
	sw.AddReactor("foo", NewTestReactor([]*conn.ChannelDescriptor{
		{ID: byte(0x00), Priority: 10},
		{ID: byte(0x01), Priority: 10},
	}, true))
	sw.AddReactor("bar", NewTestReactor([]*conn.ChannelDescriptor{
		{ID: byte(0x02), Priority: 10},
		{ID: byte(0x03), Priority: 10},
	}, true))

	return sw
}

type PeerMessage struct {
	PeerID  string
	Bytes   []byte
	Counter int
}

//Reactor is responsible for handling incoming messages of one or more `Channels`
type Reactor interface {
	cmn.Service // Start, Stop

	// SetSwitch allows setting a switch.
	SetSwitch(*p2p.Switch)

	// GetChannels returns the list of channel descriptors.
	GetChannels() []*conn.ChannelDescriptor

	// AddPeer is called by the switch when a new peer is added.
	AddPeer(peer *p2p.Peer) error

	// RemovePeer is called by the switch when the peer is stopped (due to error
	// or other reason).
	RemovePeer(peer *p2p.Peer, reason interface{})

	// Receive is called when msgBytes is received from peer.
	//
	// NOTE reactor can not keep msgBytes around after Receive completes without
	// copying.
	//
	// CONTRACT: msgBytes are not nil.
	Receive(chID byte, peer *p2p.Peer, msgBytes []byte)
}

//BaseReactor base service of a reactor
type BaseReactor struct {
	cmn.BaseService // Provides Start, Stop, .Quit
	Switch          *p2p.Switch
}

//NewBaseReactor create new base Reactor
func NewBaseReactor(name string, impl Reactor) *BaseReactor {
	return &BaseReactor{
		BaseService: *cmn.NewBaseService(nil, name, impl),
		Switch:      nil,
	}
}

//SetSwitch setting a switch for reactor
func (br *BaseReactor) SetSwitch(sw *p2p.Switch) {
	br.Switch = sw
}

//GetChannels returns the list of channel descriptors
func (*BaseReactor) GetChannels() []*conn.ChannelDescriptor { return nil }

//AddPeer is called by the switch when a new peer is added
func (*BaseReactor) AddPeer(peer *p2p.Peer) {}

//RemovePeer is called by the switch when the peer is stopped (due to error or other reason)
func (*BaseReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {}

//Receive is called when msgBytes is received from peer
func (*BaseReactor) Receive(chID byte, peer *p2p.Peer, msgBytes []byte) {}

type TestReactor struct {
	BaseReactor

	mtx          sync.Mutex
	channels     []*conn.ChannelDescriptor
	logMessages  bool
	msgsCounter  int
	msgsReceived map[byte][]PeerMessage
}

func NewTestReactor(channels []*conn.ChannelDescriptor, logMessages bool) *TestReactor {
	tr := &TestReactor{
		channels:     channels,
		logMessages:  logMessages,
		msgsReceived: make(map[byte][]PeerMessage),
	}
	tr.BaseReactor = *NewBaseReactor("TestReactor", tr)

	return tr
}

// GetChannels implements Reactor
func (tr *TestReactor) GetChannels() []*conn.ChannelDescriptor {
	return tr.channels
}

// OnStart implements BaseService
func (tr *TestReactor) OnStart() error {
	tr.BaseReactor.OnStart()
	return nil
}

// OnStop implements BaseService
func (tr *TestReactor) OnStop() {
	tr.BaseReactor.OnStop()
}

// AddPeer implements Reactor by sending our state to peer.
func (tr *TestReactor) AddPeer(peer *p2p.Peer) error {
	return nil
}

// RemovePeer implements Reactor by removing peer from the pool.
func (tr *TestReactor) RemovePeer(peer *p2p.Peer, reason interface{}) {
}

// Receive implements Reactor by handling 4 types of messages (look below).
func (tr *TestReactor) Receive(chID byte, peer *p2p.Peer, msgBytes []byte) {
	if tr.logMessages {
		tr.mtx.Lock()
		defer tr.mtx.Unlock()
		tr.msgsReceived[chID] = append(tr.msgsReceived[chID], PeerMessage{peer.ID(), msgBytes, tr.msgsCounter})
		tr.msgsCounter++
	}
}
