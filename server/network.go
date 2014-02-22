/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package pubsubsql

import (
	"net"
	"strconv"
	"sync"
)

// networkContext
type networkContext struct {
	quit   *Quitter
	router *requestRouter
}

func newNetworkContextStub() *networkContext {
	quit := NewQuitter()
	//
	datasrv := newDataService(quit)
	go datasrv.run()
	//
	router := newRequestRouter(datasrv)
	//
	context := new(networkContext)
	context.quit = quit
	context.router = router
	//
	return context
}

// network
type networkConnectionContainer interface {
	removeConnection(*networkConnection)
}

type network struct {
	networkConnectionContainer
	mutex       sync.Mutex
	connections map[uint64]*networkConnection
	listener    net.Listener
	context     *networkContext
}

func (this *network) addConnection(netconn *networkConnection) {
	if this.context.quit.Done() {
		return
	}
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.connections == nil {
		this.connections = make(map[uint64]*networkConnection)
	}
	this.connections[netconn.getConnectionId()] = netconn
	loginfo("new client connection id:", strconv.FormatUint(netconn.getConnectionId(), 10))
}

func (this *network) removeConnection(netconn *networkConnection) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.connections != nil {
		delete(this.connections, netconn.getConnectionId())
	}
}

func (this *network) connectionCount() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	count := len(this.connections)
	return count
}

func (this *network) closeConnections() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	for _, c := range this.connections {
		c.close()
	}
	this.connections = nil
}

func newNetwork(context *networkContext) *network {
	return &network{
		listener: nil,
		context:  context,
	}
}

func (this *network) start(address string) bool {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logerror("Failed to listen for incoming connections ", err.Error())
		return false
	}
	//host, port := net.SplitHostPort(address)
	loginfo("listening for incoming connections on ", address)	
	this.listener = listener
	var connectionId uint64 = 0
	// accept connections
	acceptor := func() {
		quit := this.context.quit
		quit.Join()
		defer quit.Leave()
		for {
			conn, err := this.listener.Accept()
			// stop was called
			if quit.Done() {
				return
			}
			if err == nil {
				connectionId++
				netconn := newNetworkConnection(conn, this.context, connectionId, this)
				this.addConnection(netconn)
				go netconn.run()
			} else {
				logerror("failed to accept client connection", err.Error())
			}
		}
	}
	go acceptor()
	return true
}

func (this *network) stop() {
	if this.listener != nil {
		this.listener.Close()
	}
	this.closeConnections()
}

//

type networkConnection struct {
	parent networkConnectionContainer
	conn   net.Conn
	quit   *Quitter
	router *requestRouter
	sender *responseSender
}

func newNetworkConnection(conn net.Conn, context *networkContext, connectionId uint64, parent networkConnectionContainer) *networkConnection {
	return &networkConnection{
		parent: parent,
		conn:   conn,
		quit:   context.quit,
		router: context.router,
		sender: newResponseSenderStub(connectionId),
	}
}

func (this *networkConnection) remove() {
	this.parent.removeConnection(this)
}

func (this *networkConnection) getConnectionId() uint64 {
	return this.sender.connectionId
}

func (this *networkConnection) watchForQuit() {
	select {
	case <-this.sender.quit.GetChan():
	case <-this.quit.GetChan():
	}
	this.conn.Close()
	this.parent.removeConnection(this)
}

func (this *networkConnection) close() {
	this.sender.quit.Quit(0)
}

func (this *networkConnection) run() {
	go this.watchForQuit()
	go this.read()
	this.write()
}

func (this *networkConnection) Done() bool {
	// connection can be stopped becuase of global shutdown sequence
	// or response sender is full
	// or socket error
	return this.sender.quit.Done() || this.quit.Done()
}

func (c *networkConnection) route(header *netHeader, req request) {
	item := &requestItem{
		header: header,
		req:    req,
		sender: c.sender,
	}
	c.router.route(item)
}

func (this *networkConnection) read() {
	this.quit.Join()
	defer this.quit.Leave()
	reader := newnetHelper(this.conn, config.NET_READWRITE_BUFFER_SIZE)
	//
	var err error
	var message []byte
	var header *netHeader
	tokens := newTokens()
	for {
		err = nil
		if this.Done() {
			break
		}
		header, message, err = reader.readMessage()
		if err != nil {
			break
		}
		tokens.reuse()
		// parse and route the message
		lex(string(message), tokens)
		req := parse(tokens)
		this.route(header, req)
	}
	if err != nil && !this.Done() {
		logwarn("failed to read from client connection:", this.sender.connectionId, err.Error())
		// notify writer and sender that we are done
		this.sender.quit.Quit(0)
	}
}

func (this *networkConnection) write() {
	this.quit.Join()
	defer this.quit.Leave()
	writer := newnetHelper(this.conn, config.NET_READWRITE_BUFFER_SIZE)
	var err error
	for {
		select {
		case res := <-this.sender.sender:
			debug("response is ready to be send over tcp")
			// merge responses if applicable
			nextRes := this.sender.tryRecv();
			for nextRes != nil && res.merge(nextRes) {
				nextRes = this.sender.tryRecv();
			}
			// write messages in batches if applicable
			var msg []byte
			more := true
			for err == nil && more {
				if this.Done() {
					return
				}
				msg, more = res.toNetworkReadyJSON()
				err = writer.writeMessage(msg)
				if err != nil {
					break
				}
				if !more && nextRes != nil {
					res = nextRes
					nextRes = nil			
					more = true
				}
			}
			if err != nil && !this.Done() {
				logwarn("failed to write to client connection:", this.sender.connectionId, err.Error())
				// notify reader and sender that we are done
				this.sender.quit.Quit(0)
				return
			}
		case <-this.quit.GetChan():
			debug("on write stop")
			return
		case <-this.sender.quit.GetChan():
			debug("on write connection stop")
			return
		}
	}
}
