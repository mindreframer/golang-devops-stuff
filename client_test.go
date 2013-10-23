package yagnats

import (
	"fmt"
	. "launchpad.net/gocheck"
	"net"
	"os/exec"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type YSuite struct {
	Client  *Client
	NatsCmd *exec.Cmd
}

var _ = Suite(&YSuite{})

func (s *YSuite) SetUpSuite(c *C) {
	s.NatsCmd = startNats(4223)
	waitUntilNatsUp(4223)
}

func (s *YSuite) TearDownSuite(c *C) {
	stopCmd(s.NatsCmd)
}

func (s *YSuite) SetUpTest(c *C) {
	client := NewClient()

	client.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4223",
		Username: "nats",
		Password: "nats",
	})

	s.Client = client
}

func (s *YSuite) TearDownTest(c *C) {
	s.Client.Disconnect()
	s.Client = nil
}

func (s *YSuite) TestConnectWithInvalidAddress(c *C) {
	badClient := NewClient()

	err := badClient.Connect(&ConnectionInfo{Addr: ""})

	c.Assert(err, Not(Equals), nil)
	c.Assert(err.Error(), Equals, "dial tcp: missing address")
}

func (s *YSuite) TestClientConnectWithInvalidAuth(c *C) {
	badClient := NewClient()

	err := badClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4223",
		Username: "cats",
		Password: "bats",
	})

	c.Assert(err, Not(Equals), nil)
}

func (s *YSuite) TestClientPing(c *C) {
	c.Assert(s.Client.Ping(), Equals, true)
}

func (s *YSuite) TestClientPingWhenNotConnected(c *C) {
	disconnectedClient := NewClient()
	c.Assert(disconnectedClient.Ping(), Equals, false)
}

func (s *YSuite) TestClientPingWhenConnectionClosed(c *C) {
	conn := <-s.Client.connection
	conn.Disconnect()
	c.Assert(s.Client.Ping(), Equals, false)
}

func (s *YSuite) TestClientPingWhenResponseIsTooSlow(c *C) {
	fakeConn := NewConnection("127.0.0.1:4223", "nats", "nats")

	conn, err := net.Dial("tcp", "127.0.0.1:4223")
	if err != nil {
		c.Error("Could not dial")
	}

	fakeConn.conn = conn

	disconnectedClient := NewClient()

	go func() {
		for {
			disconnectedClient.connection <- fakeConn
		}
	}()

	go func() {
		time.Sleep(1 * time.Second)
		fakeConn.pongs <- &PongPacket{}
	}()

	c.Assert(disconnectedClient.Ping(), Equals, false)
}

func (s *YSuite) TestClientSubscribe(c *C) {
	sub, _ := s.Client.Subscribe("some.subject", func(msg *Message) {})
	c.Assert(sub, Equals, 1)

	sub2, _ := s.Client.Subscribe("some.subject", func(msg *Message) {})
	c.Assert(sub2, Equals, 2)
}

func (s *YSuite) TestClientUnsubscribe(c *C) {
	payload1 := make(chan string)
	payload2 := make(chan string)

	sid1, _ := s.Client.Subscribe("some.subject", func(msg *Message) {
		payload1 <- msg.Payload
	})

	s.Client.Subscribe("some.subject", func(msg *Message) {
		payload2 <- msg.Payload
	})

	s.Client.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload1, 500)
	waitReceive(c, "hello!", payload2, 500)

	s.Client.Unsubscribe(sid1)

	s.Client.Publish("some.subject", "hello!")

	select {
	case <-payload1:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}

	waitReceive(c, "hello!", payload2, 500)
}

func (s *YSuite) TestClientSubscribeAndUnsubscribe(c *C) {
	payload := make(chan string)

	sid1, _ := s.Client.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)

	s.Client.Unsubscribe(sid1)

	s.Client.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *YSuite) TestClientAutoResubscribe(c *C) {
	doomedNats := startNats(4213)
	defer stopCmd(doomedNats)

	durableClient := NewClient()
	durableClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4213",
		Username: "nats",
		Password: "nats",
	})

	payload := make(chan string)

	durableClient.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	stopCmd(doomedNats)
	waitUntilNatsDown(4213)
	doomedNats = startNats(4213)
	defer stopCmd(doomedNats)

	waitUntilNatsUp(4213)

	durableClient.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)
}

func (s *YSuite) TestClientConnectCallback(c *C) {
	doomedNats := startNats(4213)
	defer stopCmd(doomedNats)

	connectionChannel := make(chan string)

	newClient := NewClient()
	newClient.ConnectedCallback = func() {
		connectionChannel <- "yo"
	}

	newClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4213",
		Username: "nats",
		Password: "nats",
	})

	waitReceive(c, "yo", connectionChannel, 500)
}

func (s *YSuite) TestClientReconnectCallback(c *C) {
	doomedNats := startNats(4213)
	defer stopCmd(doomedNats)

	connectionChannel := make(chan string)

	durableClient := NewClient()
	durableClient.ConnectedCallback = func() {
		connectionChannel <- "yo"
	}

	durableClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4213",
		Username: "nats",
		Password: "nats",
	})

	waitReceive(c, "yo", connectionChannel, 500)

	stopCmd(doomedNats)
	err := waitUntilNatsDown(4213)
	c.Assert(err, IsNil)

	doomedNats = startNats(4213)
	defer stopCmd(doomedNats)

	waitUntilNatsUp(4213)

	waitReceive(c, "yo", connectionChannel, 500)
}

func (s *YSuite) TestClientReconnectCallbackSelfPublish(c *C) {
	doomedNats := startNats(4213)
	defer stopCmd(doomedNats)

	connectionChannel := make(chan string)

	durableClient := NewClient()
	durableClient.ConnectedCallback = func() {
		durableClient.Publish("started", "hi")
	}

	durableClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4213",
		Username: "nats",
		Password: "nats",
	})

	// set up a bunch of subscriptions so resubscribing takes a while
	for subid := 0; subid < 1000; subid += 1 {
		durableClient.Subscribe(fmt.Sprintf("subscription.%d", subid), func(*Message) {
			// nothing
		})
	}

	durableClient.Subscribe("started", func(*Message) {
		connectionChannel <- "yo"
	})

	stopCmd(doomedNats)
	err := waitUntilNatsDown(4213)
	c.Assert(err, IsNil)

	doomedNats = startNats(4213)
	defer stopCmd(doomedNats)

	waitUntilNatsUp(4213)

	waitReceive(c, "yo", connectionChannel, 500)
}

func (s *YSuite) TestClientSubscribeInvalidSubject(c *C) {
	sid, err := s.Client.Subscribe(">.a", func(msg *Message) {})

	c.Assert(err, Not(Equals), nil)
	c.Assert(err.Error(), Equals, "Invalid Subject")
	c.Assert(sid, Equals, -1)
}

func (s *YSuite) TestClientUnsubscribeAll(c *C) {
	payload := make(chan string)

	s.Client.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)

	s.Client.UnsubscribeAll("some.subject")

	s.Client.Publish("some.subject", "hello!")

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *YSuite) TestClientPubSub(c *C) {
	payload := make(chan string)

	s.Client.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)
}

func (s *YSuite) TestClientPubSubWithQueue(c *C) {
	payload := make(chan string)

	s.Client.SubscribeWithQueue("some.subject", "some-queue", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.SubscribeWithQueue("some.subject", "some-queue", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *YSuite) TestClientPublishWithReply(c *C) {
	payload := make(chan string)

	s.Client.Subscribe("some.request", func(msg *Message) {
		s.Client.Publish(msg.ReplyTo, "response!")
	})

	s.Client.Subscribe("some.reply", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.PublishWithReplyTo("some.request", "hello!", "some.reply")

	waitReceive(c, "response!", payload, 500)
}

func (s *YSuite) TestClientDisconnect(c *C) {
	payload := make(chan string)

	s.Client.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Disconnect()

	otherClient := NewClient()
	otherClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4223",
		Username: "nats",
		Password: "nats",
	})

	otherClient.Publish("some.subject", "hello!")

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *YSuite) TestClientMessageWithoutSubscription(c *C) {
	payload := make(chan string)

	sid, err := s.Client.Subscribe("some.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	s.Client.Subscribe("some.other.subject", func(msg *Message) {
		payload <- msg.Payload
	})

	c.Assert(err, Equals, nil)

	delete(s.Client.subscriptions, sid)

	s.Client.Publish("some.subject", "hello!")
	s.Client.Publish("some.other.subject", "hello to other!")

	waitReceive(c, "hello to other!", payload, 500)
}

func (s *YSuite) TestClientLogging(c *C) {
	logger := &DefaultLogger{}
	s.Client.Logger = logger
	c.Assert(s.Client.Logger, Equals, logger)
}

func (s *YSuite) TestClientPassesLoggerToConnection(c *C) {
	logger := &DefaultLogger{}

	client := NewClient()
	client.Logger = logger

	conn, err := client.connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4223",
		Username: "nats",
		Password: "nats",
	})

	c.Assert(err, IsNil)

	c.Assert(conn.Logger, Equals, logger)
}

func (s *YSuite) TestClientMessageWhileResubscribing(c *C) {
	client := NewClient()

	client.Connect(&DisconnectingConnectionProvider{
		ReadBuffers: []string{
			// OK for foo sub, OK for bar sub
			"+OK\r\n+OK\r\n",

			// OK for foo resub, MSG to foo, OK for bar resub
			"+OK\r\nMSG foo 1 5\r\nhello\r\n+OK\r\n",
		},
	})

	payload := make(chan string)

	client.Subscribe("foo", func(msg *Message) {
		payload <- "resubscribed!"
	})

	client.Subscribe("bar", func(msg *Message) {
	})

	waitReceive(c, "resubscribed!", payload, 500)
}

func (s *YSuite) TestClientPubSubWithQueueReconnectsWithQueue(c *C) {
	doomedNats := startNats(4213)
	defer stopCmd(doomedNats)

	durableClient := NewClient()
	durableClient.Connect(&ConnectionInfo{
		Addr:     "127.0.0.1:4213",
		Username: "nats",
		Password: "nats",
	})

	payload := make(chan string)

	durableClient.SubscribeWithQueue("some.subject", "some-queue", func(msg *Message) {
		payload <- msg.Payload
	})

	durableClient.SubscribeWithQueue("some.subject", "some-queue", func(msg *Message) {
		payload <- msg.Payload
	})

	durableClient.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}

	stopCmd(doomedNats)
	waitUntilNatsDown(4213)

	doomedNats = startNats(4213)
	defer stopCmd(doomedNats)

	waitUntilNatsUp(4213)

	durableClient.Publish("some.subject", "hello!")

	waitReceive(c, "hello!", payload, 500)

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func waitReceive(c *C, expected string, from chan string, ms time.Duration) {
	select {
	case msg := <-from:
		c.Assert(msg, Equals, expected)
	case <-time.After(ms * time.Millisecond):
		c.Error("Timed out waiting for message.")
	}
}
