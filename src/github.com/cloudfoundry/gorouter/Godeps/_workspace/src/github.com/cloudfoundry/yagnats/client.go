package yagnats

import (
	"sync"
	"time"
)

type NATSClient interface {
	Ping() bool
	Connect(connectionProvider ConnectionProvider) error
	Disconnect()
	Publish(subject string, payload []byte) error
	PublishWithReplyTo(subject, reply string, payload []byte) error
	Subscribe(subject string, callback Callback) (int64, error)
	SubscribeWithQueue(subject, queue string, callback Callback) (int64, error)
	Unsubscribe(subscription int64) error
	UnsubscribeAll(subject string)
}

type Callback func(*Message)

type Client struct {
	connection          chan *Connection
	subscriptions       map[int64]*Subscription
	subscriptionCounter int64
	connected           bool
	disconnecting       bool
	lock                *sync.Mutex

	ConnectedCallback func()

	logger      Logger
	loggerMutex *sync.RWMutex
}

type Message struct {
	Subject string
	ReplyTo string
	Payload []byte
}

type Subscription struct {
	Subject  string
	Queue    string
	Callback Callback
	ID       int64
}

func NewClient() *Client {
	return &Client{
		connection:    make(chan *Connection),
		subscriptions: make(map[int64]*Subscription),
		lock:          &sync.Mutex{},

		logger:      &DefaultLogger{},
		loggerMutex: &sync.RWMutex{},
		connected:   false,
	}
}

func (c *Client) Ping() bool {
	select {
	case conn := <-c.connection:
		return conn.Ping()
	case <-time.After(500 * time.Millisecond):
		return false
	}
}

func (c *Client) Connect(cp ConnectionProvider) error {
	conn, err := c.connect(cp)
	if err != nil {
		return err
	}

	go c.serveConnections(conn, cp)

	if c.ConnectedCallback != nil {
		go c.ConnectedCallback()
	}

	return nil
}

func (c *Client) Disconnect() {
	if !c.connected || c.disconnecting {
		return
	}

	conn := <-c.connection
	c.disconnecting = true
	conn.Disconnect()
	c.connected = false
}

func (c *Client) Publish(subject string, payload []byte) error {
	conn := <-c.connection

	conn.Send(
		&PubPacket{
			Subject: subject,
			Payload: payload,
		},
	)

	return conn.ErrOrOK()
}

func (c *Client) PublishWithReplyTo(subject, reply string, payload []byte) error {
	conn := <-c.connection

	conn.Send(
		&PubPacket{
			Subject: subject,
			ReplyTo: reply,
			Payload: payload,
		},
	)

	return conn.ErrOrOK()
}

func (c *Client) Subscribe(subject string, callback Callback) (int64, error) {
	return c.subscribe(subject, "", callback)
}

func (c *Client) SubscribeWithQueue(subject, queue string, callback Callback) (int64, error) {
	return c.subscribe(subject, queue, callback)
}

func (c *Client) Unsubscribe(sid int64) error {
	conn := <-c.connection

	conn.Send(&UnsubPacket{ID: sid})

	c.lock.Lock()
	delete(c.subscriptions, sid)
	c.lock.Unlock()

	return conn.ErrOrOK()
}

func (c *Client) UnsubscribeAll(subject string) {
	idsToUnsubscribe := []int64{}
	c.lock.Lock()
	for id, sub := range c.subscriptions {
		if sub.Subject == subject {
			idsToUnsubscribe = append(idsToUnsubscribe, id)
		}
	}
	c.lock.Unlock()

	for _, id := range idsToUnsubscribe {
		c.Unsubscribe(id)
	}
}

func (c *Client) SetLogger(logger Logger) {
	c.loggerMutex.Lock()
	c.logger = logger
	c.loggerMutex.Unlock()
}

func (c *Client) Logger() Logger {
	c.loggerMutex.RLock()
	defer c.loggerMutex.RUnlock()
	return c.logger
}

func (c *Client) subscribe(subject, queue string, callback Callback) (int64, error) {
	conn := <-c.connection

	c.lock.Lock()
	c.subscriptionCounter++
	id := c.subscriptionCounter

	c.subscriptions[id] = &Subscription{
		Subject:  subject,
		Queue:    queue,
		Callback: callback,
		ID:       id,
	}
	c.lock.Unlock()

	conn.Send(
		&SubPacket{
			Subject: subject,
			Queue:   queue,
			ID:      id,
		},
	)

	err := conn.ErrOrOK()
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (c *Client) serveConnections(conn *Connection, cp ConnectionProvider) {
	c.connected = true

	// serve connection until disconnected
	for stop := false; !stop; {
		select {
		case <-conn.Disconnected:
			c.Logger().Warn("client.connection.disconnected")
			stop = true

		case c.connection <- conn:
			c.Logger().Debug("client.connection.served")
		}
	}

	// stop if client was told to disconnect
	if c.disconnecting {
		c.Logger().Info("client.disconnecting")
		return
	}

	c.reconnect(cp)
}

func (c *Client) connect(cp ConnectionProvider) (conn *Connection, err error) {
	conn, err = cp.ProvideConnection()
	if err != nil {
		return
	}

	conn.OnMessage(c.dispatchMessage)

	conn.SetLogger(c.Logger())

	return
}

func (c *Client) reconnect(cp ConnectionProvider) {
	// acquire new connection
	for {
		c.Logger().Debug("client.reconnect.starting")

		conn, err := c.connect(cp)
		if err == nil {
			go c.serveConnections(conn, cp)
			c.Logger().Debug("client.connection.resubscribing")
			c.resubscribe(conn)
			c.Logger().Debug("client.connection.resubscribed")

			if c.ConnectedCallback != nil {
				go c.ConnectedCallback()
			}
			break
		}

		c.Logger().Warnd(map[string]interface{}{"error": err.Error()}, "client.reconnect.failed")

		time.Sleep(500 * time.Millisecond)
	}
}

func (c *Client) resubscribe(conn *Connection) error {
	packetsToSend := []*SubPacket{}

	c.lock.Lock()
	for id, sub := range c.subscriptions {
		packetsToSend = append(packetsToSend, &SubPacket{
			Subject: sub.Subject,
			Queue:   sub.Queue,
			ID:      id,
		},
		)
	}
	c.lock.Unlock()

	for _, packet := range packetsToSend {
		conn.Send(packet)

		err := conn.ErrOrOK()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) dispatchMessage(msg *MsgPacket) {
	c.lock.Lock()
	sub := c.subscriptions[msg.SubID]
	if sub == nil {
		c.lock.Unlock()
		return
	}
	c.lock.Unlock()

	go sub.Callback(
		&Message{
			Subject: msg.Subject,
			Payload: msg.Payload,
			ReplyTo: msg.ReplyTo,
		},
	)
}
