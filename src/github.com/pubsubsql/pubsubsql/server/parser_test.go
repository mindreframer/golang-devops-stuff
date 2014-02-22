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

import "testing"

func expectedError(t *testing.T, a request) {
	switch a.(type) {
	case *errorRequest:

	default:
		t.Errorf("parse error: expected error")
	}

}

// STATUS
func validateStatus(t *testing.T, req request) {
	switch req.(type) {
	case *errorRequest:
		e := req.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *cmdStatusRequest:

	default:
		t.Errorf("parse error: invalid request type expected sqlStatusRequest")
	}
}

func TestParseCmdStatus(t *testing.T) {
	pc := newTokens()
	lex(" status ", pc)
	req := parse(pc)
	validateStatus(t, req)
}

// STOP
func validateStop(t *testing.T, req request) {
	switch req.(type) {
	case *errorRequest:
		e := req.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *cmdStopRequest:

	default:
		t.Errorf("parse error: invalid request type expected sqlStopRequest")
	}
}

func TestParseCmdStop(t *testing.T) {
	pc := newTokens()
	lex(" stop ", pc)
	req := parse(pc)
	validateStop(t, req)
}

// CLOSE
func validateClose(t *testing.T, req request) {
	switch req.(type) {
	case *errorRequest:
		e := req.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *cmdCloseRequest:

	default:
		t.Errorf("parse error: invalid request type expected sqlCloseRequest")
	}
}

func TestParseCmdClose(t *testing.T) {
	pc := newTokens()
	lex(" close ", pc)
	req := parse(pc)
	validateClose(t, req)
}

// INSERT
func validateInsert(t *testing.T, a request, y *sqlInsertRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlInsertRequest:
		x := a.(*sqlInsertRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match " + x.table)
		}
		// number of columns and values
		if len(x.colVals) != len(y.colVals) {
			t.Errorf("parse error: colVals lens do not match")
			break
		}
		// columns and values
		for i := 0; i < len(x.colVals); i++ {
			if *(y.colVals[i]) != *(x.colVals[i]) {
				t.Errorf("parse error: colVals do not match")
				t.Errorf("x.col:%s vs y.col:%s", x.colVals[i].col, y.colVals[i].col)
			}
		}
	default:
		t.Errorf("parse error: invalid request type expected sqlInsertRequest")
	}
}

func TestParseSqlInsertStatement1(t *testing.T) {
	pc := newTokens()
	lex(" insert into stocks (ticker, bid, ask) values (IBM, 12, 14.5645) ", pc)
	x := parse(pc)
	var y sqlInsertRequest
	y.table = "stocks"
	y.addColVal("ticker", "IBM")
	y.addColVal("bid", "12")
	y.addColVal("ask", "14.5645")
	validateInsert(t, x, &y)
}

func TestParseSqlInsertStatement2(t *testing.T) {
	pc := newTokens()
	lex(" insert ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into  ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert int ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks ( ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks () ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1,) ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1, col2 ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1, col2) value ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1, col2) values ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1, col2) values (val1)", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1, col2) values (val1, val2, ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" insert into stocks (col1, col2) values (val1, val2, val3) ", pc)
	x = parse(pc)
	expectedError(t, x)
}

// SELECT
func validateSelect(t *testing.T, a request, y *sqlSelectRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlSelectRequest:
		x := a.(*sqlSelectRequest)
		// columns
		if len(x.cols) != len(y.cols) {
			t.Errorf("parse error: columns do not match ")
		}
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match " + x.table)
		}
		// filter
		if x.filter != y.filter {
			t.Errorf("parse error: filters do not match")
		}
	default:
		t.Errorf("parse error: invalid request type expected sqlSelectRequest")
	}
}

func TestParseSqlSelectStatement1(t *testing.T) {
	pc := newTokens()
	lex(" select *  from stocks ", pc)
	x := parse(pc)
	var y sqlSelectRequest
	y.table = "stocks"
	validateSelect(t, x, &y)
}

func TestParseSqlSelectStatement2(t *testing.T) {
	pc := newTokens()
	lex(" select ticker, bid, ask  from stocks ", pc)
	x := parse(pc)
	var y sqlSelectRequest
	y.table = "stocks"
	y.addColumn("ticker")
	y.addColumn("bid")
	y.addColumn("ask")
	validateSelect(t, x, &y)
}

func TestParseSqlSelectStatement3(t *testing.T) {
	pc := newTokens()
	lex(" select *  from stocks where  ticker = 'IBM'", pc)
	x := parse(pc)
	var y sqlSelectRequest
	y.table = "stocks"
	y.filter.addFilter("ticker", "IBM")
	validateSelect(t, x, &y)
}

func TestParseSqlSelectStatement4(t *testing.T) {
	pc := newTokens()
	lex(" select ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" select *", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" select ticker , from stocks", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" select * from ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" select * from stocks where", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" select * from stocks where ticker ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" select * from stocks where ticker =", pc)
	x = parse(pc)
	expectedError(t, x)
}

// UPDATE
func validateUpdate(t *testing.T, a request, y *sqlUpdateRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlUpdateRequest:
		x := a.(*sqlUpdateRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match " + x.table)
		}
		// number of columns and values
		if len(x.colVals) != len(y.colVals) {
			t.Errorf("parse error: colVals lens do not match")
			break
		}
		// columns and values
		for i := 0; i < len(x.colVals); i++ {
			if *(y.colVals[i]) != *(x.colVals[i]) {
				t.Errorf("parse error: colVals do not match")
				t.Errorf("x.col:%s vs y.col:%s", x.colVals[i].col, y.colVals[i].col)
			}
		}
		// filter
		if x.filter != y.filter {
			t.Errorf("parse error: filters do not match")

		}

	default:
		t.Errorf("parse error: invalid request type expected sqlUpdateRequest")
	}
}

func TestParseSqlUpdateStatement1(t *testing.T) {
	pc := newTokens()
	lex(" update stocks set bid = 140.45, ask = 142.01, sector = 'TECH' where ticker = IBM", pc)
	x := parse(pc)
	var y sqlUpdateRequest
	y.table = "stocks"
	y.addColVal("bid", "140.45")
	y.addColVal("ask", "142.01")
	y.addColVal("sector", "TECH")
	y.filter.addFilter("ticker", "IBM")
	validateUpdate(t, x, &y)
}

func TestParseSqlUpdateStatement2(t *testing.T) {
	pc := newTokens()
	lex(" update stocks set bid = 140.45, ask = 142.01", pc)
	x := parse(pc)
	var y sqlUpdateRequest
	y.table = "stocks"
	y.addColVal("bid", "140.45")
	y.addColVal("ask", "142.01")
	validateUpdate(t, x, &y)

}

func TestParseSqlUpdateStatement3(t *testing.T) {
	pc := newTokens()
	lex(" update stocks set bid = ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" update stocks ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" update stocks set ", pc)
	x = parse(pc)
	expectedError(t, x)
}

// DELETE
func validateDelete(t *testing.T, a request, y *sqlDeleteRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlDeleteRequest:
		x := a.(*sqlDeleteRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match  " + x.table)
		}
		// filter
		if x.filter != y.filter {
			t.Errorf("parse error: filters do not match")
		}

	default:
		t.Errorf("parse error: invalid request type expected sqlDeleteRequest")
	}
}

func TestParseSqlDeleteStatement1(t *testing.T) {
	pc := newTokens()
	lex(" delete  from stocks ", pc)
	x := parse(pc)
	var y sqlDeleteRequest
	y.table = "stocks"
	validateDelete(t, x, &y)
}

func TestParseSqlDeleteStatement2(t *testing.T) {
	pc := newTokens()
	lex(" delete  from stocks where  ticker = 'IBM'", pc)
	x := parse(pc)
	var y sqlDeleteRequest
	y.table = "stocks"
	y.filter.addFilter("ticker", "IBM")
	validateDelete(t, x, &y)
}

func TestParseSqlDeleteStatement3(t *testing.T) {
	pc := newTokens()
	lex(" delete ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" delete from", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" delete from stocks where", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" delete from stocks where ticker ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" delete from stocks where ticker =", pc)
	x = parse(pc)
	expectedError(t, x)
}

// SUBSCRIBE
func validateSubscribe(t *testing.T, a request, y *sqlSubscribeRequest, skip bool) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlSubscribeRequest:
		x := a.(*sqlSubscribeRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match " + x.table)
		}
		// filter
		if x.filter != y.filter {
			t.Errorf("parse error: filters do not match")
		}
		if x.skip != skip {
			t.Errorf("parse error: skip do not match")
		}

	default:
		t.Errorf("parse error: invalid request type expected sqlSubscribeRequest")
	}

}

func TestParseSqlSubscribeStatement1(t *testing.T) {
	pc := newTokens()
	lex(" subscribe *  from stocks ", pc)
	x := parse(pc)
	var y sqlSubscribeRequest
	y.table = "stocks"
	validateSubscribe(t, x, &y, false)
}

func TestParseSqlSubscribeStatement2(t *testing.T) {
	pc := newTokens()
	lex(" subscribe *  from stocks where  ticker = 'IBM'", pc)
	x := parse(pc)
	var y sqlSubscribeRequest
	y.table = "stocks"
	y.filter.addFilter("ticker", "IBM")
	validateSubscribe(t, x, &y, false)
}

func TestParseSqlSubscribeStatement3(t *testing.T) {
	pc := newTokens()
	lex(" subscribe skip *  from stocks where  ticker = 'IBM'", pc)
	x := parse(pc)
	var y sqlSubscribeRequest
	y.table = "stocks"
	y.filter.addFilter("ticker", "IBM")
	validateSubscribe(t, x, &y, true)
}

func TestParseSqlSubscribeStatement4(t *testing.T) {
	pc := newTokens()
	lex(" subscribe ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" subscribe *", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" subscribe * from ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" subscribe * from stocks where", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" subscribe * from stocks where ticker ", pc)
	x = parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" subscribe * from stocks where ticker =", pc)
	x = parse(pc)
	expectedError(t, x)
}

// UNSUBSCRIBE
func validateUnsubscribe(t *testing.T, a request, y *sqlUnsubscribeRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlUnsubscribeRequest:
		x := a.(*sqlUnsubscribeRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match  " + x.table)
		}
		// filter
		if x.filter != y.filter {
			t.Errorf("parse error: filters do not match")
			t.Errorf(y.filter.col + " " + y.filter.val)
			t.Errorf(x.filter.col + " " + x.filter.val)
		}

	default:
		t.Errorf("parse error: invalid request type expected sqlUnsubscribeRequest")
	}
}

func TestParseSqlUnsubscribeStatement1(t *testing.T) {
	pc := newTokens()
	lex(" unsubscribe  from stocks ", pc)
	x := parse(pc)
	var y sqlUnsubscribeRequest
	y.table = "stocks"
	validateUnsubscribe(t, x, &y)
}

func TestParseSqlUnsubscribeStatement2(t *testing.T) {
	pc := newTokens()
	lex(" unsubscribe ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" unsubscribe from", pc)
	x = parse(pc)
	expectedError(t, x)
}

func TestParseSqlUnsubscribeStatement3(t *testing.T) {
	pc := newTokens()
	lex("unsubscribe  from stocks where  ticker = 'IBM'", pc)
	x := parse(pc)
	var y sqlUnsubscribeRequest
	y.table = "stocks"
	y.filter.addFilter("ticker", "IBM")
	validateUnsubscribe(t, x, &y)
}

// KEY
func validateKey(t *testing.T, a request, y *sqlKeyRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlKeyRequest:
		x := a.(*sqlKeyRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match  " + x.table)
		}
		// column name
		if x.column != y.column {
			t.Errorf("parse error: column names do not match  " + x.column)
		}

	default:
		t.Errorf("parse error: invalid request type expected sqlKeyRequest")
	}
}

func TestParseSqlKeyStatement1(t *testing.T) {
	pc := newTokens()
	lex(" key stocks ticker", pc)
	x := parse(pc)
	var y sqlKeyRequest
	y.table = "stocks"
	y.column = "ticker"
	validateKey(t, x, &y)
}

func TestParseSqlKeyStatement2(t *testing.T) {
	pc := newTokens()
	lex(" key ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" key stocks", pc)
	x = parse(pc)
	expectedError(t, x)
}

// TAG
func validateTag(t *testing.T, a request, y *sqlTagRequest) {
	switch a.(type) {
	case *errorRequest:
		e := a.(*errorRequest)
		t.Errorf("parse error: " + e.err)

	case *sqlTagRequest:
		x := a.(*sqlTagRequest)
		// table name
		if x.table != y.table {
			t.Errorf("parse error: table names do not match  " + x.table)
		}
		// column name
		if x.column != y.column {
			t.Errorf("parse error: column names do not match  " + x.column)
		}

	default:
		t.Errorf("parse error: invalid request type expected sqlTagRequest")
	}
}

func TestParseSqlTagStatement1(t *testing.T) {
	pc := newTokens()
	lex(" tag stocks sector", pc)
	x := parse(pc)
	var y sqlTagRequest
	y.table = "stocks"
	y.column = "sector"
	validateTag(t, x, &y)
}

func TestParseSqlTagStatement2(t *testing.T) {
	pc := newTokens()
	lex(" tag ", pc)
	x := parse(pc)
	expectedError(t, x)
	//
	pc = newTokens()
	lex(" tag stocks", pc)
	x = parse(pc)
	expectedError(t, x)
}
