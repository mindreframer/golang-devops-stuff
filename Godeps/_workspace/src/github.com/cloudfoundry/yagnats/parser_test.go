package yagnats

import (
	"bufio"
	"bytes"
	. "launchpad.net/gocheck"
)

func (s *YSuite) TestParsePing(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("PING \r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "PING\r\n")
}

func (s *YSuite) TestParseMultiPing(c *C) {
	io := bufio.NewReader(bytes.NewBuffer([]byte("PING\r\nPING\r\n")))

	parse1, err1 := Parse(io)
	parse2, err2 := Parse(io)

	c.Assert(err1, Equals, nil)
	c.Assert(parse1, Not(Equals), nil)
	c.Assert(string(parse1.Encode()), Equals, "PING\r\n")

	c.Assert(err2, Equals, nil)
	c.Assert(parse2, Not(Equals), nil)
	c.Assert(string(parse2.Encode()), Equals, "PING\r\n")
}

func (s *YSuite) TestParsePong(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("PONG \r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "PONG\r\n")
}

func (s *YSuite) TestParseInfo(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("INFO {\"a\":1} \r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "INFO {\"a\":1}\r\n")
}

func (s *YSuite) TestParseOK(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("+OK\r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "+OK\r\n")
}

func (s *YSuite) TestParseERR(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("-ERR 'foo'  \r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "-ERR 'foo'\r\n")
}

func (s *YSuite) TestParseMsg(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("MSG some.subject 42 4\r\nsup?  \r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "MSG some.subject 42 4\r\nsup?\r\n")
}

func (s *YSuite) TestParseMsgWithReplyTo(c *C) {
	packet, err := Parse(bufio.NewReader(bytes.NewBuffer([]byte("MSG some.subject 42 some.reply 4\r\nsup?\r\n"))))

	c.Assert(err, Equals, nil)
	c.Assert(packet, Not(Equals), nil)

	c.Assert(string(packet.Encode()), Equals, "MSG some.subject 42 some.reply 4\r\nsup?\r\n")
}

func (s *YSuite) TestParseMultiMsg(c *C) {
	io := bufio.NewReader(bytes.NewBuffer([]byte("MSG some.subject 42 4\r\nsup?\r\nMSG some.other.subject 43 6\r\nsup 2?\r\n")))

	parse1, err1 := Parse(io)
	parse2, err2 := Parse(io)

	c.Assert(err1, Equals, nil)
	c.Assert(parse1, Not(Equals), nil)
	c.Assert(string(parse1.Encode()), Equals, "MSG some.subject 42 4\r\nsup?\r\n")

	c.Assert(err2, Equals, nil)
	c.Assert(parse2, Not(Equals), nil)
	c.Assert(string(parse2.Encode()), Equals, "MSG some.other.subject 43 6\r\nsup 2?\r\n")
}
