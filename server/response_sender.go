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

// responseSender is a wrapper around client channel for forwarding reponses back to a client connection.
// It correctly reacts to the client connection close notification.
type responseSender struct {
	sender        chan response // channel to publish responses to
	connectionId  uint64
	quit          *Quitter
	disconnecting bool
}

// Returns new responseSender.
func newResponseSenderStub(connectionId uint64) *responseSender {
	return &responseSender{
		sender:        make(chan response, config.CHAN_RESPONSE_SENDER_BUFFER_SIZE),
		connectionId:  connectionId,
		quit:          NewQuitter(),
		disconnecting: false,
	}
}

// send sends the response to the client
func (this *responseSender) send(res response) bool {
	select {
	case this.sender <- res:
		debug("response was sent")
		return !this.quit.Done()
	case <-this.quit.GetChan():
		debug("connection is closed")
	default:
		logwarn("sender queue is full for connection: ", this.connectionId)
		// notify client connection that it needs to close due to inability to
		// send responses in a timely manner
		this.quit.Quit(0)
	}
	return false
}

// tryRecv attemps to receive a response from the client.
func (this *responseSender) tryRecv() response {
	select {
	case res := <-this.sender:
		return res
	default:
		return nil
	}
	return nil
}

// recv receives a response from the client.
// For testing only.	
func (this *responseSender) testRecv() response {
	return <-this.sender
}
