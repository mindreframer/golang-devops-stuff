package yagnats

import (
	"encoding/json"
	. "launchpad.net/gocheck"
)

func (s *YSuite) TestInfoEncode(c *C) {
	packet := &InfoPacket{Payload: `{"a":1}`}
	c.Assert(string(packet.Encode()), Equals, "INFO {\"a\":1}\r\n")
}

func (s *YSuite) TestPingEncode(c *C) {
	packet := &PingPacket{}
	c.Assert(string(packet.Encode()), Equals, "PING\r\n")
}

func (s *YSuite) TestPongEncode(c *C) {
	packet := &PongPacket{}
	c.Assert(string(packet.Encode()), Equals, "PONG\r\n")
}

func (s *YSuite) TestConnectEncode(c *C) {
	packet := &ConnectPacket{
		User: "foo",
		Pass: "bar",
	}

	encoded := packet.Encode()
	c.Assert(string(encoded[0:8]), Equals, "CONNECT ")

	payload := encoded[8:]

	parsed := &connectionPayload{}
	json.Unmarshal(payload, &parsed)

	c.Check(parsed.Verbose, Equals, true)
	c.Check(parsed.Pedantic, Equals, true)
	c.Check(parsed.User, Equals, "foo")
	c.Check(parsed.Pass, Equals, "bar")
}

func (s *YSuite) TestOKEncode(c *C) {
	packet := &OKPacket{}
	c.Assert(string(packet.Encode()), Equals, "+OK\r\n")
}

func (s *YSuite) TestERREncode(c *C) {
	packet := &ERRPacket{Message: "Sup"}
	c.Assert(string(packet.Encode()), Equals, "-ERR 'Sup'\r\n")
}

func (s *YSuite) TestSubEncode(c *C) {
	packet := &SubPacket{Subject: "some.subject", Queue: "some.queue", ID: 42}
	c.Assert(string(packet.Encode()), Equals, "SUB some.subject some.queue 42\r\n")
}

func (s *YSuite) TestUnsubEncode(c *C) {
	packet := &UnsubPacket{ID: 42}
	c.Assert(string(packet.Encode()), Equals, "UNSUB 42\r\n")
}

func (s *YSuite) TestSubEncodeWithNoQueue(c *C) {
	packet := &SubPacket{Subject: "some.subject", ID: 42}
	c.Assert(string(packet.Encode()), Equals, "SUB some.subject 42\r\n")
}

func (s *YSuite) TestPubEncode(c *C) {
	packet := &PubPacket{Subject: "some.subject", Payload: []byte("sup?")}
	c.Assert(string(packet.Encode()), Equals, "PUB some.subject 4\r\nsup?\r\n")
}

func (s *YSuite) TestPubEncodeWithReplyTo(c *C) {
	packet := &PubPacket{Subject: "some.subject", ReplyTo: "some.reply", Payload: []byte("sup?")}
	c.Assert(string(packet.Encode()), Equals, "PUB some.subject some.reply 4\r\nsup?\r\n")
}

func (s *YSuite) TestMsgEncode(c *C) {
	packet := &MsgPacket{Subject: "some.subject", SubID: 42, Payload: []byte("sup?")}
	c.Assert(string(packet.Encode()), Equals, "MSG some.subject 42 4\r\nsup?\r\n")
}

func (s *YSuite) TestMsgEncodeWithReplyTo(c *C) {
	packet := &MsgPacket{Subject: "some.subject", SubID: 42, ReplyTo: "some.reply", Payload: []byte("sup?")}
	c.Assert(string(packet.Encode()), Equals, "MSG some.subject 42 some.reply 4\r\nsup?\r\n")
}
