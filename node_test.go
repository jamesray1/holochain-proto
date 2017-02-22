package holochain

import (
	"bytes"
	"context"
	"fmt"
	ic "github.com/libp2p/go-libp2p-crypto"
	net "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"strings"
	"testing"
)

func TestNewNode(t *testing.T) {

	node, err := makeNode(1234, "")
	defer node.Close()
	Convey("It should create a node", t, func() {
		So(err, ShouldBeNil)
		So(node.NetAddr.String(), ShouldEqual, "/ip4/127.0.0.1/tcp/1234")
		So(node.HashAddr.Pretty(), ShouldEqual, "QmNN6oDiV4GsfKDXPVmGLdBLLXCM28Jnm7pz7WD63aiwSG")
	})

	Convey("It should send between nodes", t, func() {
		node2, err := makeNode(4321, "node2")
		So(err, ShouldBeNil)
		defer node2.Close()

		node.Host.Peerstore().AddAddr(node2.HashAddr, node2.NetAddr, pstore.PermanentAddrTTL)
		var payload string
		node2.Host.SetStreamHandler("/testprotocol/1.0.0", func(s net.Stream) {
			defer s.Close()

			buf := make([]byte, 1024)
			n, err := s.Read(buf)
			if err != nil {
				payload = err.Error()
			} else {
				payload = string(buf[:n])
			}

			_, err = s.Write([]byte("I got: " + payload))

			if err != nil {
				panic(err)
			}
		})

		s, err := node.Host.NewStream(context.Background(), node2.HashAddr, "/testprotocol/1.0.0")
		So(err, ShouldBeNil)
		_, err = s.Write([]byte("greetings"))
		So(err, ShouldBeNil)

		out, err := ioutil.ReadAll(s)
		So(err, ShouldBeNil)
		So(payload, ShouldEqual, "greetings")
		So(string(out), ShouldEqual, "I got: greetings")
	})
}

func TestNodeSend(t *testing.T) {

	node1, err := makeNode(1234, "node1")
	if err != nil {
		panic(err)
	}
	defer node1.Close()

	node2, err := makeNode(1235, "node2")
	if err != nil {
		panic(err)
	}
	defer node2.Close()
	Convey("It should start the DHT protocol", t, func() {
		err := node1.StartDHT()
		So(err, ShouldBeNil)
	})
	Convey("It should start the Src protocol", t, func() {
		err := node2.StartSrc()
		So(err, ShouldBeNil)
	})

	m := Message{Type: PUT_REQUEST, Body: "fish"}
	Convey("It should send", t, func() {
		node1.Host.Peerstore().AddAddr(node2.HashAddr, node2.NetAddr, pstore.PermanentAddrTTL)
		node2.Host.Peerstore().AddAddr(node1.HashAddr, node1.NetAddr, pstore.PermanentAddrTTL)
		r1, err := node2.Send(DHTProtocol, node1.HashAddr, &m)
		So(err, ShouldBeNil)
		So(r1.Type, ShouldEqual, OK_RESPONSE)
		So(r1.Body, ShouldEqual, nil)
		r2, err := node1.Send(SourceProtocol, node2.HashAddr, &m)
		So(err, ShouldBeNil)
		So(r2.Type, ShouldEqual, ERROR_RESPONSE)
		So(r2.Body, ShouldEqual, "message type 2 not in holochain-src protocol")

	})
}

func TestMessageCoding(t *testing.T) {
	m := Message{Type: PUT_REQUEST, Body: "fish"}
	var d []byte
	var err error
	Convey("It should encode messages", t, func() {
		d, err = m.Encode()
		So(err, ShouldBeNil)
		So(fmt.Sprintf("%v", d), ShouldEqual, "[39 255 139 3 1 1 7 77 101 115 115 97 103 101 1 255 140 0 1 2 1 4 84 121 112 101 1 4 0 1 4 66 111 100 121 1 16 0 0 0 21 255 140 1 4 1 6 115 116 114 105 110 103 12 6 0 4 102 105 115 104 0]")

	})
	Convey("It should decode messages", t, func() {
		var m2 Message
		r := bytes.NewReader(d)
		err = m2.Decode(r)
		So(err, ShouldBeNil)
		So(fmt.Sprintf("%v", m), ShouldEqual, "{2 fish}")

	})
}

func makeNode(port int, id string) (*Node, error) {
	listenaddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	// use a constant reader so the key will be the same each time for the test...
	r := strings.NewReader(id + "1234567890123456789012345678901234567890")
	key, _, err := ic.GenerateEd25519Key(r)
	if err != nil {
		panic(err)
	}
	return NewNode(listenaddr, key)
}
