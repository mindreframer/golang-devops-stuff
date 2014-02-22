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

// requestItem is a container for client request and sender used to send back responses
type requestItem struct {
	header *netHeader
	req    request
	sender *responseSender
}

func (this *requestItem) getRequestId() uint32 {
	if this.header != nil {
		return this.header.RequestId
	}
	return uint32(0)
}

// dataService pre-processes sqlRequests and forwards them to approptiate tables for further proccessging.
// It servers as a collection container for tables.
type dataService struct {
	requests chan *requestItem
	quit     *Quitter
	tables   map[string]*table
}

// newDataService returns new dataService.
func newDataService(quit *Quitter) *dataService {
	return &dataService{
		requests: make(chan *requestItem, config.CHAN_DATASERVICE_REQUESTS_BUFFER_SIZE),
		quit:     quit,
		tables:   make(map[string]*table),
	}
}

// acceptRequest accepts the request from a client.
func (this *dataService) acceptRequest(item *requestItem) {
	select {
	case this.requests <- item:
	case <-this.quit.GetChan():
	}
}

// run is an event loop function that recieves sql requests from connected clients and forwards them for further processing.
func (this *dataService) run() {
	this.quit.Join()
	defer this.quit.Leave()
	for {
		select {
		case item := <-this.requests:
			if this.quit.Done() {
				debug("data service exited due to quit notification")
				return
			}
			this.onSqlRequest(item)
		case <-this.quit.GetChan():
			debug("data service exited due to quit notification")
			return
		}
	}
}

// onSqlRequest forwards sql request to the appropriate table.
func (this *dataService) onSqlRequest(item *requestItem) {
	tableName := item.req.getTableName()
	tbl := this.tables[tableName]
	if tbl == nil {
		// auto create table and go run table event loop
		tbl = newTable(tableName)
		this.tables[tableName] = tbl
		tbl.quit = this.quit
		tbl.requests = make(chan *requestItem, config.CHAN_TABLE_REQUESTS_BUFFER_SIZE)
		loginfo("table", tableName, " was created; connection:", item.sender.connectionId) 
		go tbl.run()
	}
	// forward sql request to the table
	tbl.requests <- item
}
