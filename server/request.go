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

type requestType uint8

const (
	requestTypeError requestType = iota // error request indicates scan or parse error.
	requestTypeCmd                      // cmd requests help, status etc.
	requestTypeSql                      // sql actins insert, update etc.
)

// request
type request interface {
	getRequestType() requestType
	getTableName() string
}

// errorRequest is an error request.
type errorRequest struct {
	request
	err string
}

// Returns type of a request.
func (act errorRequest) getRequestType() requestType {
	return requestTypeError
}

// sqlRequest is a generic sql request.
type sqlRequest struct {
	request
	table string
}

func (this *sqlRequest) getRequestType() requestType {
	return requestTypeSql
}

func (this *sqlRequest) getTableName() string {
	return this.table
}

// cmdRequest is a generic command request.
type cmdRequest struct {
	request
	requestId uint32
}

func (this *cmdRequest) getRequestType() requestType {
	return requestTypeCmd
}

// 
type cmdStatusRequest struct {
	cmdRequest
}

type cmdStopRequest struct {
	cmdRequest
}

type cmdCloseRequest struct {
	cmdRequest
}

// columnValue is a pair of column and value
type columnValue struct {
	col string
	val string
}

// Temporarely stub for sqlFilter type that will be more capble in future versions.
type sqlFilter struct {
	columnValue
}

// Adds col = val to sqlFilter.
func (this *sqlFilter) addFilter(col string, val string) {
	this.col = col
	this.val = val
}

// sqlInsertRequest is a request for sql insert statement.
type sqlInsertRequest struct {
	sqlRequest
	colVals []*columnValue
}

// Adds column to columnValue slice.
func (this *sqlInsertRequest) addColumn(col string) {
	this.colVals = append(this.colVals, &columnValue{col: col})
}

// Adds column and value to columnValue slice for insert request.
func (this *sqlInsertRequest) addColVal(col string, val string) {
	this.colVals = append(this.colVals, &columnValue{col: col, val: val})
}

// Set value at a particular index of columnValue slice.
func (this *sqlInsertRequest) setValueAt(idx int, val string) {
	this.colVals[idx].val = val
}

// sqlSelectRequest is a request for sql select statement.
type sqlSelectRequest struct {
	sqlRequest
	cols   []string
	filter sqlFilter
}

func (req *sqlSelectRequest) addColumn(col string) {
	req.cols = append(req.cols, col)
}

// sqlUpdateRequest is a request for sql update statement.
type sqlUpdateRequest struct {
	sqlRequest
	colVals []*columnValue
	filter  sqlFilter
}

// Adds column and value to columnValue slice for udpate request.
func (this *sqlUpdateRequest) addColVal(col string, val string) {
	this.colVals = append(this.colVals, &columnValue{col: col, val: val})
}

// sqlDeleteRequest is a request for sql delete statement.
type sqlDeleteRequest struct {
	sqlRequest
	filter sqlFilter
}

// sqlKeyRequest is a request for sql key statement.
// Key defines unique index.
type sqlKeyRequest struct {
	sqlRequest
	column string
}

// sqlTagRequest is a request for sql tag statement.
// Tag defines non-unique index.
type sqlTagRequest struct {
	sqlRequest
	column string
}

// sqlSubscribeRequest is a request for sql subscribe statement.
type sqlSubscribeRequest struct {
	sqlRequest
	skip   bool
	filter sqlFilter
	sender *responseSender
}

// sqlUnsubscribeRequest is a request for sql unsubscribe statement.
type sqlUnsubscribeRequest struct {
	sqlRequest
	connectionId uint64
	filter       sqlFilter
}
