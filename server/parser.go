/* Copyright (C) 2013 CompleteD LLC.
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

import "fmt"

// tokenProducer produces tokens for the parser.
type tokenProducer interface {
	Produce() *token
}

// parser
type parser struct {
	tokens    tokenProducer
	streaming bool
}

// Indicates that error happened during parse phase and returns errorRequest
func (this *parser) parseError(s string) *errorRequest {
	e := errorRequest{
		err: s,
	}
	return &e
}

// Helper functions

func (this *parser) parseSqlEqualVal(colval *columnValue, tok *token) request {
	//col
	if tok == nil {
		tok = this.tokens.Produce()
	}
	if tok.typ != tokenTypeSqlColumn {
		return this.parseError("expected.col name")
	}
	colval.col = tok.val
	// =
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlEqual {
		return this.parseError("expected = sign")
	}
	// value
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlValue {
		return this.parseError("expected valid value")
	}
	colval.val = tok.val
	return nil
}

func (this *parser) parseTableName(table *string) request {
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlTable {
		return this.parseError("expected table name")
	}
	*table = tok.val
	return nil
}

func (this *parser) parseColumnName(column *string) request {
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlColumn {
		return this.parseError("expected column name")
	}
	*column = tok.val
	return nil
}

func (this *parser) parseEOF(req request) request {
	tok := this.tokens.Produce()
	if tok.typ == tokenTypeEOF {
		return req
	}
	return this.parseError("expected EOF")
}

func (this *parser) parseSqlWhere(filter *sqlFilter, tok *token) request {
	//must be where
	if tok != nil && tok.typ != tokenTypeSqlWhere {
		return this.parseError("expected where clause")
	}
	return this.parseSqlEqualVal(&(filter.columnValue), nil)
}

// STATUS cmd
func (this *parser) parseCmdStatus() request {
	// into
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeEOF {
		return this.parseError("unexpected extra token")
	}
	return new(cmdStatusRequest)
}

// STOP cmd
func (this *parser) parseCmdStop() request {
	// into
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeEOF {
		return this.parseError("unexpected extra token")
	}
	return new(cmdStopRequest)
}

// CLOSE cmd
func (this *parser) parseCmdClose() request {
	// into
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeEOF {
		return this.parseError("unexpected extra token")
	}
	return new(cmdCloseRequest)
}

// INSERT sql statement

// Parses sql insert statement and returns sqlInsertRequest on success.
func (this *parser) parseSqlInsert() request {
	// into
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlInto {
		return this.parseError("expected into")
	}
	req := &sqlInsertRequest{
		colVals: make([]*columnValue, 0, config.PARSER_SQL_INSERT_REQUEST_COLUMN_CAPACITY),
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// (
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlLeftParenthesis {
		return this.parseError("expected ( ")
	}
	// columns
	columns := 0
	expectedType := tokenTypeSqlColumn
	var errreq request
	var str string
	for expectedType == tokenTypeSqlColumn {
		errreq, expectedType, str = this.parseSqlInsertColumn()
		if errreq != nil {
			return errreq
		}
		req.addColumn(str)
		columns++
	}
	// values
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlValues {
		return this.parseError("expected values keyword")
	}
	// (
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlLeftParenthesis {
		return this.parseError("expected values ( ")
	}
	//
	expectedType = tokenTypeSqlValue
	values := 0
	for expectedType == tokenTypeSqlValue {
		errreq, expectedType, str = this.parseSqlInsertValue()
		if errreq != nil {
			return errreq
		}
		if values < columns {
			req.setValueAt(values, str)
		}
		values++
	}
	if columns != values {
		s := fmt.Sprintf("number of columns:%d and values:%d do not match", columns, values)
		return this.parseError(s)
	}
	return this.returningColumnsHelper(nil, req, &req.returningColumns)
}

func (this *parser) returningColumnsHelper(tok *token, req request, r *returningColumns) request {
	if tok == nil {
		tok = this.tokens.Produce()
	}
	switch tok.typ {
	case tokenTypeEOF:
		return req
	case tokenTypeSqlReturning:
		tok = this.tokens.Produce()
		if tok.typ != tokenTypeSqlStar {
			if errreq := this.parseReturningColumns(&tok, r); errreq != nil {
				return errreq
			}
		} else {
			r.use = true
		}
	default:
		s := fmt.Sprintf("invalid token %v: expected returning", tok.val)
		return this.parseError(s)
	}
	return req
}

// Parses sql push statement and returns sqlInsertRequest on success.
func (this *parser) parseSqlPush() request {
	req := newSqlPushRequest()
	// into
	tok := this.tokens.Produce()
	switch tok.typ {
	case tokenTypeSqlBack:
		req.front = false
		tok = this.tokens.Produce()
	case tokenTypeSqlFront:
		req.front = true
		tok = this.tokens.Produce()
	}
	if tok.typ != tokenTypeSqlInto {
		return this.parseError("expected into")
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// (
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlLeftParenthesis {
		return this.parseError("expected ( ")
	}
	// columns
	columns := 0
	expectedType := tokenTypeSqlColumn
	var errreq request
	var str string
	for expectedType == tokenTypeSqlColumn {
		errreq, expectedType, str = this.parseSqlInsertColumn()
		if errreq != nil {
			return errreq
		}
		req.sqlInsertRequest.addColumn(str)
		columns++
	}
	// values
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlValues {
		return this.parseError("expected values keyword")
	}
	// (
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlLeftParenthesis {
		return this.parseError("expected values ( ")
	}
	//
	expectedType = tokenTypeSqlValue
	values := 0
	for expectedType == tokenTypeSqlValue {
		errreq, expectedType, str = this.parseSqlInsertValue()
		if errreq != nil {
			return errreq
		}
		if values < columns {
			req.sqlInsertRequest.setValueAt(values, str)
		}
		values++
	}
	if columns != values {
		s := fmt.Sprintf("number of columns:%d and values:%d do not match", columns, values)
		return this.parseError(s)
	}
	return this.returningColumnsHelper(nil, req, &req.returningColumns)
}

func (this *parser) parseSqlInsertColumn() (request, tokenType, string) {
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlColumn {
		return this.parseError("expected column name"), tokenTypeError, ""
	}
	str := tok.val
	tok = this.tokens.Produce()
	if tok.typ == tokenTypeSqlComma {
		return nil, tokenTypeSqlColumn, str
	}
	if tok.typ == tokenTypeSqlRightParenthesis {
		return nil, tokenTypeSqlValues, str
	}
	return this.parseError("expected , or ) "), tokenTypeError, ""
}

func (this *parser) parseSqlInsertValue() (request, tokenType, string) {
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlValue {
		return this.parseError("expected value"), tokenTypeError, ""
	}
	str := tok.val
	tok = this.tokens.Produce()
	if tok.typ == tokenTypeSqlComma {
		return nil, tokenTypeSqlValue, str
	}
	if tok.typ == tokenTypeSqlRightParenthesis {
		return nil, tokenTypeSqlRightParenthesis, str
	}
	return this.parseError("expected , or ) "), tokenTypeError, ""
}

// SELECT sql statement

func (this *parser) parseReturningColumns(tok **token, retColumns *returningColumns) request {
	nextIsColumn := true
	for {
		if nextIsColumn {
			if (*tok).typ != tokenTypeSqlColumn {
				return this.parseError("expected column name")
			}
			nextIsColumn = false
			retColumns.addColumn((*tok).val)
		} else {
			if (*tok).typ != tokenTypeSqlComma {
				break
			}
			nextIsColumn = true
		}
		*tok = this.tokens.Produce()
	}
	return nil
}

// Parses sql select statement and returns sqlSelectRequest on success.
func (this *parser) parseSqlSelect() request {
	// *
	req := newSqlSelectRequest()
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlStar {
		if errreq := this.parseReturningColumns(&tok, &req.returningColumns); errreq != nil {
			return errreq
		}
	} else {
		tok = this.tokens.Produce()
	}
	// from
	if tok.typ != tokenTypeSqlFrom {
		return this.parseError("expected from")
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// possible eof
	tok = this.tokens.Produce()
	if tok.typ == tokenTypeEOF {
		return req
	}
	// where
	if errreq := this.parseSqlWhere(&(req.filter), tok); errreq != nil {
		return errreq
	}
	// we are good
	return req
}

// Parses sql peek statement and returns sqlPeekRequest on success.
func (this *parser) parseSqlPeek() request {
	req := newSqlPeekRequest()
	tok := this.tokens.Produce()
	switch tok.typ {
	case tokenTypeSqlFront:
		req.front = true
		tok = this.tokens.Produce()
	case tokenTypeSqlBack:
		req.front = false
		tok = this.tokens.Produce()
	}
	if tok.typ != tokenTypeSqlStar {
		if errreq := this.parseReturningColumns(&tok, &req.returningColumns); errreq != nil {
			return errreq
		}
	} else {
		tok = this.tokens.Produce()
	}
	// from
	if tok.typ != tokenTypeSqlFrom {
		return this.parseError("expected from")
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeEOF {
		return this.parseError("expected eof token")
	}
	return req
}

// Parses sql pop statement and returns sqlPopRequest on success.
func (this *parser) parseSqlPop() request {
	req := newSqlPopRequest()
	tok := this.tokens.Produce()
	switch tok.typ {
	case tokenTypeSqlFront:
		req.front = true
		tok = this.tokens.Produce()
	case tokenTypeSqlBack:
		req.front = false
		tok = this.tokens.Produce()
	}

	switch tok.typ {
	case tokenTypeSqlStar:
		req.use = true
		tok = this.tokens.Produce()
	case tokenTypeSqlFrom:
		req.use = false
	default:
		if errreq := this.parseReturningColumns(&tok, &req.returningColumns); errreq != nil {
			return errreq
		}
	}
	// from
	if tok.typ != tokenTypeSqlFrom {
		return this.parseError("expected from")
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeEOF {
		return this.parseError("expected eof token")
	}
	return req
}

// UPDATE sql statement

// Parses sql update statement and returns sqlUpdateRequest on success.
func (this *parser) parseSqlUpdate() request {
	req := &sqlUpdateRequest{
		colVals: make([]*columnValue, 0, config.PARSER_SQL_UPDATE_REQUEST_COLUMN_CAPACITY),
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// set
	tok := this.tokens.Produce()
	if tok.typ == tokenTypeSqlSet {
		return this.parseSqlUpdateColVals(req)
	}
	return this.parseError("expected set keyword")
}

func (this *parser) parseSqlUpdateColVals(req *sqlUpdateRequest) request {
	count := 0
	tok := this.tokens.Produce()
loop:
	for ; ; tok = this.tokens.Produce() {
		switch tok.typ {
		case tokenTypeSqlColumn:
			colval := new(columnValue)
			req.colVals = append(req.colVals, colval)
			if errreq := this.parseSqlEqualVal(colval, tok); errreq != nil {
				return errreq
			}
			count++

		case tokenTypeSqlWhere:
			if errreq := this.parseSqlWhere(&(req.filter), tok); errreq != nil {
				return errreq
			}
			tok = nil
			break loop
		case tokenTypeSqlReturning:
			break loop
		case tokenTypeEOF:
			break loop

		case tokenTypeSqlComma:
			continue

		default:
			return this.parseError("expected.col or where keyword")
		}
	}
	if count == 0 {
		return this.parseError("expected at least on.col value pair")
	}
	return this.returningColumnsHelper(tok, req, &req.returningColumns)
}

// DELETE sql statement

// Parses sql delete statement and returns sqlDeleteRequest on success.
func (this *parser) parseSqlDelete() request {
	// from
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlFrom {
		return this.parseError("expected from")
	}
	req := new(sqlDeleteRequest)
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// possible eof
	tok = this.tokens.Produce()
	switch tok.typ {
	case tokenTypeEOF:
		return req
	case tokenTypeSqlWhere:
		if errreq := this.parseSqlWhere(&(req.filter), tok); errreq != nil {
			return errreq
		}
		tok = this.tokens.Produce()
	}
	return this.returningColumnsHelper(tok, req, &req.returningColumns)
}

// KEY sql statement

// Parses sql key statement and returns sqlKeyRequest on success.
func (this *parser) parseSqlKey() request {
	req := new(sqlKeyRequest)
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// column name
	if errreq := this.parseColumnName(&req.column); errreq != nil {
		return errreq
	}
	return this.parseEOF(req)
}

// TAG sql statement

// Parses sql tag statement and returns sqlRequest on success.
func (this *parser) parseSqlTag() request {
	req := new(sqlTagRequest)
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// column name
	if errreq := this.parseColumnName(&req.column); errreq != nil {
		return errreq
	}
	return this.parseEOF(req)
}

// SUBSCRIBE sql statement

// Parses sql subscribe statement and returns sqlSubscribeRequest on success.
func (this *parser) parseSqlSubscribe() request {
	tok := this.tokens.Produce()
	if tok.typ == tokenTypeSqlTopic {
		return &sqlSubscribeTopicRequest { topic: tok.val }
	}
	req := new(sqlSubscribeRequest)
	// skip
	if tok.typ == tokenTypeSqlSkip {
		req.skip = true
		tok = this.tokens.Produce()
	}

	if tok.typ != tokenTypeSqlStar {
		return this.parseError("expected * symbol")
	}
	// from
	tok = this.tokens.Produce()
	if tok.typ != tokenTypeSqlFrom {
		return this.parseError("expected from")
	}
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// possible eof
	tok = this.tokens.Produce()
	if tok.typ == tokenTypeEOF {
		return req
	}
	// where
	if errreq := this.parseSqlWhere(&(req.filter), tok); errreq != nil {
		return errreq
	}
	// we are good
	return req
}

// UNSUBSCRIBE sql statement

// Parses sql unsubscribe statement and returns sqlUnsubscribeRequest on success.
func (this *parser) parseSqlUnsubscribe() request {
	// from
	tok := this.tokens.Produce()
	if tok.typ != tokenTypeSqlFrom {
		return this.parseError("expected from")
	}
	req := new(sqlUnsubscribeRequest)
	// table name
	if errreq := this.parseTableName(&req.table); errreq != nil {
		return errreq
	}
	// possible eof
	tok = this.tokens.Produce()
	if tok.typ == tokenTypeEOF {
		return req
	}
	// than it must be where
	if errreq := this.parseSqlWhere(&(req.filter), tok); errreq != nil {
		return errreq
	}
	// we are good
	return req
}

// Runs the parser.
func (this *parser) run() request {
	tok := this.tokens.Produce()
	switch tok.typ {
	case tokenTypeSqlStream:
		this.streaming = true
		return this.run()
	case tokenTypeSqlInsert:
		return this.parseSqlInsert()
	case tokenTypeSqlSelect:
		return this.parseSqlSelect()
	case tokenTypeSqlUpdate:
		return this.parseSqlUpdate()
	case tokenTypeSqlDelete:
		return this.parseSqlDelete()
	case tokenTypeSqlPush:
		return this.parseSqlPush()
	case tokenTypeSqlPop:
		return this.parseSqlPop()
	case tokenTypeSqlPeek:
		return this.parseSqlPeek()
	case tokenTypeSqlSubscribe:
		return this.parseSqlSubscribe()
	case tokenTypeSqlUnsubscribe:
		return this.parseSqlUnsubscribe()
	case tokenTypeSqlKey:
		return this.parseSqlKey()
	case tokenTypeSqlTag:
		return this.parseSqlTag()
	case tokenTypeCmdStatus:
		return this.parseCmdStatus()
	case tokenTypeCmdStop:
		return this.parseCmdStop()
	case tokenTypeCmdClose:
		return this.parseCmdClose()
	}
	return this.parseError("invalid request")
}

// Parses tokens and returns an request.
func parse(tokens tokenProducer) request {
	parser := &parser{
		tokens:    tokens,
		streaming: false,
	}
	req := parser.run()
	if parser.streaming {
		req.setStreaming()
	}
	return req
}
