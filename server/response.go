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

import "strconv"

type responseStatusType int8

const (
	responseStatusOk  responseStatusType = iota // ok.
	responseStatusErr                           // error.
)

// response
type response interface {
	getResponseStatus() responseStatusType
	toNetworkReadyJSON() ([]byte, bool)
	setRequestId(requestId uint32)
	merge(res response) bool
}

type requestIdResponse struct {
	response
	requestId uint32
}

func (this *requestIdResponse) setRequestId(requestId uint32) {
	this.requestId = requestId
}

func (this *requestIdResponse) merge(res response) bool {
	return false
}

// json helper functions
func ok(builder *JSONBuilder) {
	builder.nameValue("status", "ok")
}

func action(builder *JSONBuilder, action string) {
	builder.nameValue("action", action)
}

// errorResponse
type errorResponse struct {
	requestIdResponse
	msg string
}

func newErrorResponse(msg string) *errorResponse {
	return &errorResponse{
		msg: msg,
	}
}

func (this *errorResponse) getResponsStatus() responseStatusType {
	return responseStatusErr
}

func (this *errorResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.nameValue("status", "err")
	builder.valueSeparator()
	builder.nameValue("msg", this.msg)
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), false
}

// okResponse
type okResponse struct {
	requestIdResponse
	action string
}

func newOkResponse(action string) *okResponse {
	return &okResponse{action: action}
}

func (this *okResponse) getResponsStatus() responseStatusType {
	return responseStatusOk
}

func (this *okResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, this.action)
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), false
}

// cmdStatusResponse
type cmdStatusResponse struct {
	requestIdResponse
	connections int
}

func newCmdStatusResponse(connections int) *cmdStatusResponse {
	return &cmdStatusResponse{
		connections: connections,
	}
}

func (this *cmdStatusResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, "status")
	builder.valueSeparator()
	builder.nameIntValue("connections", this.connections)
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), false
}

// sqlSelectResponse is a response for sql select statement
type sqlSelectResponse struct {
	requestIdResponse
	columns []*column
	records []*record
	//
	init    bool
	rows    int
	fromrow int
	torow   int
}

func row(builder *JSONBuilder, columns []*column, rec *record) {
	builder.beginArray()
	// columns and values
	for colIndex, _ := range columns {
		if colIndex != 0 {
			builder.valueSeparator()
		}
		builder.string(rec.getValue(colIndex))
	}
	builder.endArray()
}

func (this *sqlSelectResponse) data(builder *JSONBuilder, pubsub bool) bool {
	// we are not returning data but only number of rows affected
	if len(this.columns) == 0 {
		builder.nameIntValue("rows", this.rows)
		return false
	}
	// write the columns first
	builder.string("columns")
	builder.nameSeparator()
	builder.beginArray()
	for colIndex, col := range this.columns {
		// another row
		if colIndex != 0 {
			builder.valueSeparator()
		}
		builder.string(col.name)
	}
	builder.endArray()
	builder.objectSeparator()
	// now write data (records)
	if !this.init {
		this.init = true
		this.rows = len(this.records)
		this.fromrow = 0
		this.torow = 0
	}
	more := len(this.records) > config.DATA_BATCH_SIZE
	records := this.records
	if more {
		records = this.records[0:config.DATA_BATCH_SIZE]
		this.records = this.records[config.DATA_BATCH_SIZE:]
		this.fromrow = this.torow + 1
		this.torow = this.fromrow + config.DATA_BATCH_SIZE - 1
	} else if this.rows > 0 {
		this.fromrow = this.torow + 1
		this.torow = this.rows
	}
	// rows, fromrow, torow
	rows := this.rows
	fromrow := this.fromrow
	torow := this.torow
	if pubsub && fromrow > 0 {
		rows = torow - fromrow + 1
		torow = rows
		fromrow = 1
	}
	builder.nameIntValue("rows", rows)
	builder.valueSeparator()
	builder.nameIntValue("fromrow", fromrow)
	builder.valueSeparator()
	builder.nameIntValue("torow", torow)
	builder.valueSeparator()
	//
	builder.string("data")
	builder.nameSeparator()
	builder.beginArray()
	for recIndex, rec := range records {
		// another row
		if recIndex != 0 {
			builder.valueSeparator()
		}
		builder.newLine()
		row(builder, this.columns, rec)
	}
	builder.newLine()
	builder.endArray()
	return more
}

func (this *sqlSelectResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, "select")
	builder.valueSeparator()
	more := this.data(builder, false)
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), more
}

func (this *sqlSelectResponse) copyRecordData(source *record) {
	l := len(this.columns)
	dest := &record{
		values: make([]string, l, l),
	}
	for idx, col := range this.columns {
		dest.setValue(idx, source.getValue(col.ordinal))
	}
	addRecordToSlice(&this.records, dest)
}

// sqlActionDataResponse
type sqlActionDataResponse struct {
	sqlSelectResponse
	action string
}

func newUpdateResponse() *sqlActionDataResponse {
	return &sqlActionDataResponse{
		action: "update",
	}
}

func newDeleteResponse() *sqlActionDataResponse {
	return &sqlActionDataResponse{
		action: "delete",
	}
}

func newInsertResponse() *sqlActionDataResponse {
	return &sqlActionDataResponse{
		action: "insert",
	}
}

func newPushResponse() *sqlActionDataResponse {
	return &sqlActionDataResponse{
		action: "push",
	}
}

func newPopResponse() *sqlActionDataResponse {
	return &sqlActionDataResponse{
		action: "pop",
	}
}

func newPeekResponse() *sqlActionDataResponse {
	return &sqlActionDataResponse{
		action: "peek",
	}
}

func (this *sqlActionDataResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, this.action)
	builder.valueSeparator()
	more := this.data(builder, false)
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), more
}

// sqlSubscribeResponse
type sqlSubscribeResponse struct {
	requestIdResponse
	pubsubid uint64
}

func (this *sqlSubscribeResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, "subscribe")
	builder.valueSeparator()
	builder.nameValue("pubsubid", strconv.FormatUint(this.pubsubid, 10))
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), false
}

func newSubscribeResponse(sub *subscription) response {
	return &sqlSubscribeResponse{
		pubsubid: sub.id,
	}
}

// sqlPubSubResponse
type sqlPubSubResponse struct {
	sqlSelectResponse
	pubsubid uint64
}

func (this *sqlPubSubResponse) toNetworkReadyJSONHelper(act string) ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, act)
	builder.valueSeparator()
	builder.nameValue("pubsubid", strconv.FormatUint(this.pubsubid, 10))
	builder.valueSeparator()
	more := this.data(builder, true)
	builder.endObject()
	return builder.getNetworkBytes(0), more
}

func mergeHelper(res1 *sqlPubSubResponse, res2 *sqlPubSubResponse) bool {
	if res1.pubsubid != res2.pubsubid {
		return false
	}
	if len(res1.columns) != len(res2.columns) {
		return false
	}
	res1.records = append(res1.records, res2.records...)
	return true
}

// sqlActionAddResponse
type sqlActionAddResponse struct {
	sqlPubSubResponse
}

func (this *sqlActionAddResponse) toNetworkReadyJSON() ([]byte, bool) {
	return this.toNetworkReadyJSONHelper("add")
}

func (this *sqlActionAddResponse) merge(res response) bool {
	switch res.(type) {
	case *sqlActionAddResponse:
		source := res.(*sqlActionAddResponse)
		return mergeHelper(&this.sqlPubSubResponse, &source.sqlPubSubResponse)
	}
	return false
}

// sqlActionInsertResponse
type sqlActionInsertResponse struct {
	sqlPubSubResponse
}

func (this *sqlActionInsertResponse) toNetworkReadyJSON() ([]byte, bool) {
	return this.toNetworkReadyJSONHelper("insert")
}

func (this *sqlActionInsertResponse) merge(res response) bool {
	switch res.(type) {
	case *sqlActionInsertResponse:
		source := res.(*sqlActionInsertResponse)
		return mergeHelper(&this.sqlPubSubResponse, &source.sqlPubSubResponse)
	}
	return false
}

// sqlActonDeleteResponse
type sqlActionDeleteResponse struct {
	sqlPubSubResponse
}

func (this *sqlActionDeleteResponse) toNetworkReadyJSON() ([]byte, bool) {
	return this.toNetworkReadyJSONHelper("delete")
}

func (this *sqlActionDeleteResponse) merge(res response) bool {
	switch res.(type) {
	case *sqlActionDeleteResponse:
		source := res.(*sqlActionDeleteResponse)
		return mergeHelper(&this.sqlPubSubResponse, &source.sqlPubSubResponse)
	}
	return false
}

// sqlActionRemoveResponse
type sqlActionRemoveResponse struct {
	sqlPubSubResponse
}

func (this *sqlActionRemoveResponse) toNetworkReadyJSON() ([]byte, bool) {
	return this.toNetworkReadyJSONHelper("remove")
}

func (this *sqlActionRemoveResponse) merge(res response) bool {
	switch res.(type) {
	case *sqlActionRemoveResponse:
		source := res.(*sqlActionRemoveResponse)
		return mergeHelper(&this.sqlPubSubResponse, &source.sqlPubSubResponse)
	}
	return false
}

// sqlActionUpdateResponse
type sqlActionUpdateResponse struct {
	sqlPubSubResponse
}

func (this *sqlActionUpdateResponse) toNetworkReadyJSON() ([]byte, bool) {
	return this.toNetworkReadyJSONHelper("update")
}

func (this *sqlActionUpdateResponse) merge(res response) bool {
	switch res.(type) {
	case *sqlActionUpdateResponse:
		source := res.(*sqlActionUpdateResponse)
		if this.pubsubid != source.pubsubid {
			return false
		}
		if len(this.columns) != len(source.columns) {
			return false
		}
		// now check if columns are the same
		for idx, col := range this.columns {
			if col.ordinal != source.columns[idx].ordinal {
				return false
			}
		}
		this.records = append(this.records, source.records...)
		return true
	}
	return false
}

func newSqlActionUpdateResponse(pubsubid uint64, cols []*column, rec *record) *sqlActionUpdateResponse {
	var res sqlActionUpdateResponse
	res.columns = cols
	res.pubsubid = pubsubid
	res.copyRecordData(rec)
	return &res
}

// sqlUnsubscribeResponse
type sqlUnsubscribeResponse struct {
	requestIdResponse
	unsubscribed int
}

func (this *sqlUnsubscribeResponse) toNetworkReadyJSON() ([]byte, bool) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	ok(builder)
	builder.valueSeparator()
	action(builder, "unsubscribe")
	builder.valueSeparator()
	builder.nameIntValue("subscriptions", this.unsubscribed)
	builder.endObject()
	return builder.getNetworkBytes(this.requestId), false
}
