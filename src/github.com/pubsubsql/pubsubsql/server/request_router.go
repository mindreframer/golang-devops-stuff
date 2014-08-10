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

package server

// requestRouter routs request to appropriate service for processing
type requestRouter struct {
	dataSrv            *dataService
	controllerRequests chan *requestItem
}

// requestRouter factory
func newRequestRouter(dataSrv *dataService) *requestRouter {
	return &requestRouter{dataSrv: dataSrv}
}

func (this *requestRouter) onError(item *requestItem) {
	ereq := item.req.(*errorRequest)
	res := newErrorResponse(ereq.err)
	res.requestId = item.getRequestId()
	item.sender.send(res)
}

func (this *requestRouter) route(item *requestItem) {
	switch item.req.getRequestType() {
	case requestTypeSql:
		this.dataSrv.acceptRequest(item)
	case requestTypeCmd:
		this.onCmd(item)
	case requestTypeError:
		this.onError(item)
	default:
		panic("unsuported request type")
	}
}

func (this *requestRouter) onCmd(item *requestItem) {
	switch item.req.(type) {
	case *cmdCloseRequest:
		loginfo("client connection:", item.sender.connectionId, "requested to disconnect ")
		item.sender.disconnecting = true
		item.sender.quit.Quit(0)
	default:
		this.onControllerCmd(item)
	}
}

func (this *requestRouter) onControllerCmd(item *requestItem) {
	if this.controllerRequests != nil {
		this.controllerRequests <- item
	}
}
