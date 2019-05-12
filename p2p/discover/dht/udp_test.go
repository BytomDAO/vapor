package dht

import (
	"net"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/vapor/crypto/ed25519"
)

func TestCodec(t *testing.T) {
	testCases := []struct {
		ptype byte
		req   interface{}
	}{
		{ptype: byte(pingPacket), req: &ping{Version: Version, Topics: []Topic{"abc"}, Rest: []byte{0x01, 0x02, 0x03, 0x04, 0x05}, From: rpcEndpoint{IP: net.IP{0x01}}, To: rpcEndpoint{IP: net.IP{0x02}}}},
		{ptype: byte(pongPacket), req: &pong{To: rpcEndpoint{IP: net.IP{0x02}}, WaitPeriods: []uint32{0x01}, Rest: []byte{0x01, 0x02, 0x03, 0x04, 0x05}, ReplyTok: []byte{0x01}}},
	}
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal("GenerateKey err:", err)
	}

	magicNumber := uint64(999999)
	for i, v := range testCases {
		packet, hash, err := encodePacket(privKey, v.ptype, v.req, magicNumber)
		if err != nil {
			t.Fatal("encodePacket err. index:", i, "err:", err)
		}

		pkt := ingressPacket{}
		if err := decodePacket(packet, &pkt, magicNumber); err != nil {
			t.Fatal("decodePacket err. index:", i, "err:", err)
		}

		if !reflect.DeepEqual(hash, pkt.hash) {
			t.Fatal("codec hash err, index:", i, "got:", pkt.hash, "want:", hash)
		}

		if !reflect.DeepEqual(pkt.data, v.req) {
			t.Fatal("codec index:", i, "got:", spew.Sdump(pkt.data), "want:", spew.Sdump(v.req))
		}
	}
}
