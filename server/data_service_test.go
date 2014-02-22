/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option any later version.
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

import "testing"
import "time"

func TestDataServiceRunAndStop(t *testing.T) {
	quit := NewQuitter()
	dataSrv := newDataService(quit)
	go dataSrv.run()
	if !quit.Quit(3 * time.Second) {
		t.Errorf("stoper.Stop() expected true but got false")
	}
}

func sqlHelper(sql string, sender *responseSender) *requestItem {
	pc := newTokens()
	lex(sql, pc)
	req := parse(pc).(request)
	return &requestItem{
		req:    req,
		sender: sender,
	}
}

func TestDataService(t *testing.T) {
	quit := NewQuitter()
	dataSrv := newDataService(quit)
	go dataSrv.run()
	sender := newResponseSenderStub(1)
	// insert
	dataSrv.acceptRequest(sqlHelper("insert into stocks (ticker, bid, ask, sector) values (IBM, 123, 124, TECH) ", sender))
	res := sender.testRecv()
	validateSqlInsertResponse(t, res)
	// select
	dataSrv.acceptRequest(sqlHelper(" select * from stocks ", sender))
	res = sender.testRecv()
	validateSqlSelect(t, res, 1, 5)
	// key
	dataSrv.acceptRequest(sqlHelper(" key stocks ticker ", sender))
	res = sender.testRecv()
	validateOkResponse(t, res)
	// tag
	dataSrv.acceptRequest(sqlHelper(" tag stocks sector ", sender))
	res = sender.testRecv()
	validateOkResponse(t, res)
	// subscribe
	dataSrv.acceptRequest(sqlHelper(" subscribe * from stocks sector = TECH ", sender))
	res = sender.testRecv()
	validateSqlSubscribeResponse(t, res)
	res = sender.testRecv() // action add
	// update
	dataSrv.acceptRequest(sqlHelper(" update stocks set bid = 140 where ticker = IBM ", sender))
	res = sender.testRecv() // first is action update
	res = sender.testRecv()
	validateSqlUpdate(t, res, 1)
	// delete
	dataSrv.acceptRequest(sqlHelper(" delete from stocks where ticker = IBM ", sender))
	res = sender.testRecv() // first is action delete
	res = sender.testRecv()
	validateSqlDelete(t, res, 1)
	// unsubscribe
	dataSrv.acceptRequest(sqlHelper(" unsubscribe from stocks where pubsubid = 1 ", sender))
	res = sender.testRecv() // first is action delete
	validateSqlUnsubscribe(t, res, 1)

	quit.Quit(time.Millisecond * 1000)
}
