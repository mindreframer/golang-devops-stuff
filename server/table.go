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
	"strconv"
	"sync/atomic"
)

// this function is purely for testing porposes
func (this *table) getTagedColumnValuesCount(name string, val string) int {
	col := this.getColumn(name)
	if col == nil || !col.isTag() {
		return 0
	}
	i := 0
	for tg := col.tagmap.getTag(val); tg != nil; tg = tg.next {
		i++
	}
	return i
}

var subid uint64 = 0

// table
type table struct {
	name         string
	colMap       map[string]*column
	colSlice     []*column
	records      []*record
	tagedColumns []*column
	pubsub       pubsub
	//
	subscriptions mapSubscriptionByConnection
	//
	requests chan *requestItem
	quit     *Quitter
	//
	requestId uint32
	//
	count uint32;
}

// table factory
func newTable(name string) *table {
	table := &table{
		name:          name,
		colMap:        make(map[string]*column),
		colSlice:      make([]*column, 0, config.TABLE_COLUMNS_CAPACITY),
		records:       make([]*record, 0, config.TABLE_RECORDS_CAPACITY),
		tagedColumns:  make([]*column, 0, config.TABLE_COLUMNS_CAPACITY),
		subscriptions: make(mapSubscriptionByConnection),
		requestId:     0,
	}
	table.addColumn("id")
	return table
}

// COLUMNS functions

// Returns total number of columns.
func (this *table) getColumnCount() int {
	l := len(this.colSlice)
	if l != len(this.colMap) {
		panic("Something bad happened column slice and map do not match")
	}
	return l
}

// Adds column and returns column added column.
func (this *table) addColumn(name string) *column {
	col := newColumn(name, len(this.colSlice))
	this.colMap[name] = col
	this.colSlice = append(this.colSlice, col)
	return col
}

// Tries to retrieve existing column or adds it if does not existhis.
// Returns true when new column was added.
func (this *table) getAddColumn(name string) (*column, bool) {
	col, columnExists := this.colMap[name]
	if columnExists {
		return col, false
	}
	return this.addColumn(name), true
}

// Retrieves existing column
func (this *table) getColumn(name string) *column {
	col, ok := this.colMap[name]
	if ok {
		return col
	}
	return nil
}

// Deletes columns starting at particular ordinal.
func (this *table) removeColumns(ordinal int) {
	if len(this.colSlice) <= ordinal {
		return
	}
	tail := this.colSlice[ordinal:]
	for _, col := range tail {
		delete(this.colMap, col.name)
	}
	this.colSlice = this.colSlice[:ordinal]
}

// RECORDS functions

// Creates new record but does not add it to the table.
// Returns new record and to be record id
func (this *table) prepareRecord() (*record, int) {
	id := len(this.records)
	rec := newRecord(len(this.colSlice), id)
	l := len(this.tagedColumns) + 1
	rec.links = make([]link, l)
	return rec, id
}

// adNewRecord add newly created record to the table
func (this *table) addNewRecord(rec *record) {
	this.count++;
	addRecordToSlice(&this.records, rec)
}

// addRecordToSlice generic helper function that adds record to the slice and
// automatically expands the slice
func addRecordToSlice(records *[]*record, rec *record) {
	//check if records slice needs to grow by third
	l := len(*records)
	if cap(*records) == len(*records) {
		temp := *records
		*records = make([]*record, l, l+(l/3))
		copy(*records, temp)
	}
	*records = append(*records, rec)
}

// Returns record by id
func (this *table) getRecord(id int) *record {
	if len(this.records) > id {
		return this.records[id]
	}
	return nil
}

// Returns total number of records in the table
func (this *table) getRecordCount() int {
	return len(this.records)
}

// Delete record from the table.
func (this *table) deleteRecord(rec *record) {
	// delete record tags
	for _, col := range this.tagedColumns {
		this.deleteTag(rec, col)
	}
	// delete record
	if this.records[rec.id()] != nil {
		this.count--;
		this.records[rec.id()] = nil
	}
}

// Looks up record by id.
// Returns record slice with max one elementhis.
func (this *table) getRecordById(val string) []*record {
	idx, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return nil
	}
	if idx < 0 || int64(len(this.records)) <= idx {
		return nil
	}
	records := make([]*record, 1, 1)
	records[0] = this.records[idx]
	return records
}

// Validates sql filter
// Returns errorResponse on error
func (this *table) validateSqlFilter(filter sqlFilter) (response, *column) {
	var col *column
	if len(filter.col) > 0 {
		col = this.getColumn(filter.col)
		if col == nil {
			return newErrorResponse("invalid column: " + filter.col), nil
		}
	}
	if col != nil && col.typ == columnTypeNormal {
		return newErrorResponse("can not use non indexed column " + filter.col + " as valid filter"), nil
	}
	return nil, col
}

// Retrieves records based by column value
func (this *table) getRecordsByValue(val string, col *column) []*record {
	if col == nil {
		// all
		return this.records
	}
	switch col.typ {
	case columnTypeKey:
		return this.getRecordsByTag(val, col)
	case columnTypeTag:
		return this.getRecordsByTag(val, col)
	case columnTypeId:
		return this.getRecordById(val)
	}
	return nil
}

// Retrieves records based on the supplied filter
func (this *table) getRecordsBySqlFilter(filter sqlFilter) ([]*record, response) {
	e, col := this.validateSqlFilter(filter)
	if e != nil {
		return nil, e
	}
	return this.getRecordsByValue(filter.val, col), nil
}

// Looks up records by tag.
func (this *table) getRecordsByTag(val string, col *column) []*record {
	// we need to optimize allocations
	// perhaps its possible to know in advance how manny records
	// will be returned
	records := make([]*record, 0, config.TABLE_GET_RECORDS_BY_TAG_CAPACITY)
	for tag := col.tagmap.getTag(val); tag != nil; tag = tag.next {
		records = append(records, this.records[tag.idx])
		l := len(records)
		if cap(records) == l {
			temp := records
			// grow by 3rd
			records = make([]*record, l, l+(l/3))
			copy(records, temp)
		}
	}
	return records
}

// Bind records values, keys and tags.
func (this *table) bindRecord(cols []*column, colVals []*columnValue, rec *record, id int) {
	for idx, colVal := range colVals {
		col := cols[idx]
		rec.setValue(col.ordinal, colVal.val)
		// update key
		switch col.typ {
		case columnTypeKey:
			this.tagValue(col, id, rec)
		case columnTypeTag:
			this.tagValue(col, id, rec)
		}
	}
}

type pubsubRA struct {
	removed []*pubsub
	added   map[*pubsub]int
}

func newPubsubRA() *pubsubRA {
	return &pubsubRA{
		removed: make([]*pubsub, 0, 3),
		added:   make(map[*pubsub]int),
	}
}

func getIfHasData(ra *pubsubRA) *pubsubRA {
	if ra != nil && (len(ra.removed) > 0 || len(ra.added) > 0) {
		return ra
	}
	return nil
}

func hasWhatToRemove(ra *pubsubRA) bool {
	return ra != nil && len(ra.removed) > 0
}

func hasWhatToAdd(ra *pubsubRA) bool {
	return ra != nil && len(ra.added) > 0
}

func (ra *pubsubRA) toBeRemoved(pubsub *pubsub) {
	if pubsub != nil {
		ra.removed = append(ra.removed, pubsub)
	}
}

func (ra *pubsubRA) toBeAdded(pubsub *pubsub) {
	if pubsub != nil {
		ra.added[pubsub] = 1
	}
}

func (this *table) updateRecordKeyTag(col *column, val string, rec *record, id int, ra **pubsubRA) {
	removed := this.deleteTag(rec, col)
	rec.setValue(col.ordinal, val)
	added := this.tagValue(col, id, rec)
	// updated with the same value ignore this case
	if removed == added {
		return
	}
	if *ra == nil {
		*ra = newPubsubRA()
	}
	ra.toBeRemoved(removed)
	ra.toBeAdded(added)
}

// Updates record with new values, keys and tags.
func (this *table) updateRecord(cols []*column, colVals []*columnValue, rec *record, id int) *pubsubRA {
	var ra *pubsubRA
	for idx, colVal := range colVals {
		col := cols[idx]
		switch col.typ {
		case columnTypeKey:
			this.updateRecordKeyTag(col, colVal.val, rec, id, &ra)
		case columnTypeTag:
			this.updateRecordKeyTag(col, colVal.val, rec, id, &ra)
		case columnTypeNormal:
			rec.setValue(col.ordinal, colVal.val)
		}
	}
	return getIfHasData(ra)
}

// TAGS helper functions

// Add value to non unique indexed column.
func addValueToTags(col *column, val string, idx int) (*tag, *pubsub) {
	return col.tagmap.addTag(val, idx)
}

// Binds tag, pubsub and record.
func (this *table) tagValue(col *column, idx int, rec *record) *pubsub {
	val := rec.getValue(col.ordinal)
	tg, pubsub := addValueToTags(col, val, idx)
	lnk := link{
		tg:     tg,
		pubsub: pubsub,
	}
	if len(rec.links) <= col.tagIndex {
		rec.links = append(rec.links, lnk)
	} else {
		rec.links[col.tagIndex] = lnk
	}
	return pubsub
}

// Deletes tag value for a particular record
func (this *table) deleteTag(rec *record, col *column) *pubsub {
	lnk := &rec.links[col.tagIndex]
	if lnk.tg != nil {
		switch removeTag(lnk.tg) {
		case removeTagLast:
			col.tagmap.removeTag(rec.getValue(col.ordinal))
		case removeTagSlide:
			// we need to retag the slided record
			slidedRecord := this.records[lnk.tg.idx]
			if slidedRecord != nil {
				slidedRecord.links[col.tagIndex].tg = lnk.tg
			}
		}
	}
	ret := lnk.pubsub
	lnk.clear()
	return ret
}

// INSERT sql statement

// Proceses sql insert request by inserting record in the table.
// On success returns sqlInsertResponse.
func (this *table) sqlInsert(req *sqlInsertRequest) response {
	rec, id := this.prepareRecord()
	// validate unique keys constrain
	cols := make([]*column, len(req.colVals))
	originalColLen := len(this.colSlice)
	for idx, colVal := range req.colVals {
		col, _ := this.getAddColumn(colVal.col)
		if col.isKey() && col.keyContainsValue(colVal.val) {
			//remove created columns
			this.removeColumns(originalColLen)
			return newErrorResponse("insert failed due to duplicate column key:" + colVal.col + " value:" + colVal.val)
		}
		cols[idx] = col
	}
	// ready to insert
	this.bindRecord(cols, req.colVals, rec, id)
	this.addNewRecord(rec)
	res := new(sqlInsertResponse)
	this.copyRecordToSqlSelectResponse(&res.sqlSelectResponse, rec)
	this.onInsert(rec)
	return res
}

// SELECT sql statement

func (this *table) copyRecordsToSqlSelectResponse(res *sqlSelectResponse, records []*record, columns []*column) {
	res.columns = columns
	if len(res.columns) == 0 {
		res.columns = this.colSlice
	}
	res.records = make([]*record, 0, len(records))
	for _, rec := range records {
		if rec != nil {
			res.copyRecordData(rec)
		}
	}
}

func (this *table) copyRecordToSqlSelectResponse(res *sqlSelectResponse, rec *record) {
	res.columns = this.colSlice
	res.records = make([]*record, 0, 1)
	res.copyRecordData(rec)
}

// Processes sql select request.
// On success returns sqlSelectResponse.
func (this *table) sqlSelect(req *sqlSelectRequest) response {
	records, errResponse := this.getRecordsBySqlFilter(req.filter)
	if errResponse != nil {
		return errResponse
	}
	// precreate columns
	var columns []*column
	if len(req.cols) > 0 {
		columns = make([]*column, 0, cap(req.cols))
		for _, colName := range req.cols {
			col, _ := this.getAddColumn(colName)
			columns = append(columns, col)
		}
	}
	//
	var res sqlSelectResponse
	this.copyRecordsToSqlSelectResponse(&res, records, columns)
	return &res
}

// UPDATE sql statement

// Processes sql update requesthis.
// On success returns sqlUpdateResponse.
func (this *table) sqlUpdate(req *sqlUpdateRequest) response {
	records, errResponse := this.getRecordsBySqlFilter(req.filter)
	if errResponse != nil {
		return errResponse
	}
	res := &sqlUpdateResponse{updated: 0}
	var onlyRecord *record
	switch len(records) {
	case 0:
		return res
	case 1:
		onlyRecord = records[0]
	}
	// validate duplicate keys
	cols := make([]*column, len(req.colVals)+1)
	originalColLen := len(this.colSlice)
	cols[0] = this.colSlice[0]
	for idx, colVal := range req.colVals {
		col, _ := this.getAddColumn(colVal.col)
		if col.isKey() && col.keyContainsValue(colVal.val) {
			if onlyRecord == nil || onlyRecord != this.getRecordsByTag(colVal.val, col)[0] {
				//remove created columns
				this.removeColumns(originalColLen)
				return newErrorResponse("update failed due to duplicate column key:" + colVal.col + " value:" + colVal.val)
			}
		}
		cols[idx+1] = col
	}
	// all is valid ready to update
	updated := 0
	for _, rec := range records {
		if rec != nil {
			updated++
			ra := this.updateRecord(cols[1:], req.colVals, rec, int(rec.id()))
			if hasWhatToRemove(ra) {
				this.onRemove(ra.removed, rec)
			}
			var added *map[*pubsub]int
			if hasWhatToAdd(ra) {
				added = &ra.added
				this.onAdd(ra.added, rec)
			}
			this.onUpdate(cols, rec, added)
		}
	}
	res.updated = updated
	return res
}

// DELETE sql statement

// Processes sql delete requesthis.
// On success returns sqlDeleteResponse.
func (this *table) sqlDelete(req *sqlDeleteRequest) response {
	records, errResponse := this.getRecordsBySqlFilter(req.filter)
	if errResponse != nil {
		return errResponse
	}
	deleted := 0
	for _, rec := range records {
		if rec != nil {
			deleted++
			this.onDelete(rec)
			this.deleteRecord(rec)
			rec.free()
		}
	}
	return &sqlDeleteResponse{deleted: deleted}
}

// Key sql statement

// Processes sql key requesthis.
// On success returns sqlOkResponse.
func (this *table) sqlKey(req *sqlKeyRequest) response {
	// key is already defined for this column
	col := this.getColumn(req.column)
	if col != nil && col.isIndexed() {
		return newErrorResponse("key or tag already defined for column:" + req.column)
	}
	// new column on existing records
	if col == nil && len(this.records) > 0 {
		return newErrorResponse("can not define key for non existant column due to possible duplicates")
	}
	// new column no records
	if col != nil {
		unique := make(map[string]int, cap(this.records))
		// check if there are duplicates
		for idx, rec := range this.records {
			if rec != nil {
				val := rec.getValue(col.ordinal)
				if _, contains := unique[val]; contains {
					return newErrorResponse("can not define key due to possible duplicates in existing records")
				}
				unique[val] = idx
			}
		}
	}
	//
	this.tagOrKeyColumn(req.column, columnTypeKey)
	return newOkResponse("key")
}

// TAG sql statement

func (this *table) tagOrKeyColumn(c string, coltyp columnType) {
	col, _ := this.getAddColumn(c)
	this.tagedColumns = append(this.tagedColumns, col)
	col.makeTags(len(this.tagedColumns))
	col.typ = coltyp
	// tag existing values
	for idx, rec := range this.records {
		if rec != nil {
			this.tagValue(col, idx, rec)
		}
	}
}

// Processes sql tag requesthis.
// On success returns sqlOkResponse.
func (this *table) sqlTag(req *sqlTagRequest) response {
	// tag is already defined for this column
	col := this.getColumn(req.column)
	if col != nil && col.isIndexed() {
		return newErrorResponse("key or tag already defined for column:" + req.column)
	}
	//
	this.tagOrKeyColumn(req.column, columnTypeTag)
	return newOkResponse("tag")
}

// SUBSCRIBE sql statement

func (this *table) newSubscription(sender *responseSender) *subscription {
	val := atomic.AddUint64(&subid, 1)
	sub := newSubscription(sender, val)
	this.subscriptions.add(sender.connectionId, sub)
	return sub
}

func (this *table) subscribeToTable(sender *responseSender, skip bool) (*subscription, []*record) {
	sub := this.newSubscription(sender)
	this.pubsub.add(sub)
	this.send(sender, newSubscribeResponse(sub))
	var records []*record
	if !skip {
		records = this.records
	}
	return sub, records
}

func (this *table) subscribeToKeyOrTag(col *column, val string, sender *responseSender, skip bool) (*subscription, []*record) {
	sub := this.newSubscription(sender)
	var records []*record
	if !skip {
		records = this.getRecordsByTag(val, col)
	}
	col.tagmap.getAddTagItem(val).pubsub.add(sub)
	this.send(sender, newSubscribeResponse(sub))
	return sub, records
}

func (this *table) subscribeToId(id string, sender *responseSender, skip bool) (*subscription, []*record) {
	records := this.getRecordById(id)
	if len(records) > 0 {
		sub := this.newSubscription(sender)
		records[0].addSubscription(sub)
		this.send(sender, newSubscribeResponse(sub))
		if skip {
			records = nil
		}
		return sub, records
	}
	this.send(sender, newErrorResponse("id: "+id+" does not exist"))
	return nil, nil
}

func (this *table) send(sender *responseSender, res response) {
	res.setRequestId(this.requestId)
	sender.send(res)
}

func (this *table) subscribe(col *column, val string, sender *responseSender, skip bool) (*subscription, []*record) {
	if col == nil {
		return this.subscribeToTable(sender, skip)
	}
	switch col.typ {
	case columnTypeKey:
		return this.subscribeToKeyOrTag(col, val, sender, skip)
	case columnTypeTag:
		return this.subscribeToKeyOrTag(col, val, sender, skip)
	case columnTypeId:
		return this.subscribeToId(val, sender, skip)
	}
	this.send(sender, newErrorResponse("Unexpected logical error"))
	return nil, nil
}

// Processes sql subscribe requesthis.
// Does not return anything, responses are send directly to response this.
func (this *table) sqlSubscribe(req *sqlSubscribeRequest) {
	// validate
	errRes, col := this.validateSqlFilter(req.filter)
	if errRes != nil {
		this.send(req.sender, errRes)
		return
	}
	// subscribe
	sub, records := this.subscribe(col, req.filter.val, req.sender, req.skip)
	if sub != nil && len(records) > 0 && this.count >  0 {
		// publish initial action add
		this.publishActionAdd(sub, records)
	}
}

// PUBSUB helpers
type publishAction func(thisbl *table, sub *subscription, rec *record) bool

func (this *table) visitSubscriptions(rec *record, publishActionFunc publishAction) {
	f := func(sub *subscription) bool {
		return publishActionFunc(this, sub, rec)
	}
	this.pubsub.visit(f)
	for _, lnk := range rec.links {
		if lnk.pubsub != nil {
			lnk.pubsub.visit(f)
		}
	}
}

func (this *table) publishActionAdd(sub *subscription, records []*record) bool {
	res := new(sqlActionAddResponse)
	res.pubsubid = sub.id
	this.copyRecordsToSqlSelectResponse(&res.sqlSelectResponse, records, nil)
	return sub.sender.send(res)
}

func publishActionInsert(this *table, sub *subscription, rec *record) bool {
	res := new(sqlActionInsertResponse)
	res.pubsubid = sub.id
	this.copyRecordToSqlSelectResponse(&res.sqlSelectResponse, rec)
	return sub.sender.send(res)
}

func publishActionDelete(this *table, sub *subscription, rec *record) bool {
	res := new(sqlActionDeleteResponse)
	res.pubsubid = sub.id
	this.copyRecordToSqlSelectResponse(&res.sqlSelectResponse, rec)
	return sub.sender.send(res)
}

func (this *table) onInsert(rec *record) {
	this.visitSubscriptions(rec, publishActionInsert)
}

func (this *table) onDelete(rec *record) {
	this.visitSubscriptions(rec, publishActionDelete)
}

func (this *table) onRemove(pubsubs []*pubsub, rec *record) {
	visitor := func(sub *subscription) bool {
		res := new(sqlActionRemoveResponse)
		res.pubsubid = sub.id
		this.copyRecordToSqlSelectResponse(&res.sqlSelectResponse, rec)
		return sub.sender.send(res)
	}
	for _, pubsub := range pubsubs {
		pubsub.visit(visitor)
	}
}

func (this *table) onAdd(added map[*pubsub]int, rec *record) {
	visitor := func(sub *subscription) bool {
		res := new(sqlActionAddResponse)
		res.pubsubid = sub.id
		this.copyRecordToSqlSelectResponse(&res.sqlSelectResponse, rec)
		return sub.sender.send(res)
	}
	for pubsub, _ := range added {
		pubsub.visit(visitor)
	}
}

func (this *table) onUpdate(cols []*column, rec *record, added *map[*pubsub]int) {
	visitor := func(sub *subscription) bool {
		res := newSqlActionUpdateResponse(sub.id, cols, rec)
		return sub.sender.send(res)
	}
	this.pubsub.visit(visitor)
	for _, lnk := range rec.links {
		if lnk.pubsub != nil {
			// ignore updates for record that was just added
			if added != nil && (*added)[lnk.pubsub] != 0 {
				continue
			}
			lnk.pubsub.visit(visitor)
		}
	}
}

// UNSUBSCRIBE

// Processes sql unsubscribe requesthis.
func (this *table) sqlUnsubscribe(req *sqlUnsubscribeRequest) response {
	// validate
	if len(req.filter.col) > 0 && req.filter.col != "pubsubid" {
		return newErrorResponse("Invalid filter expected pubsubid but got " + req.filter.col)
	}
	// unsubscribe by pubsubid for a given connection
	res := new(sqlUnsubscribeResponse)
	val := req.filter.val
	if len(val) > 0 {
		pubsubid, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return newErrorResponse("Failed to unsubscribe, pubsubid " + val + " is not valid")
		}
		if this.subscriptions.deactivate(req.connectionId, pubsubid) {
			res.unsubscribed = 1
		}
	} else {
		// unsubscribe all subscriptions for a given connection
		res.unsubscribed = this.subscriptions.deactivateAll(req.connectionId)
	}
	return res
}

// run

func (this *table) run() {
	this.quit.Join()
	defer this.quit.Leave()
	for {
		select {
		case item := <-this.requests:
			if this.quit.Done() {
				debug("table quit")
				return
			}
			this.requestId = item.getRequestId()
			this.onSqlRequest(item.req, item.sender)
		case <-this.quit.GetChan():
			debug("table quit")
			return
		}
	}
}

func (this *table) onSqlRequest(req request, sender *responseSender) {
	switch req.(type) {
	case *sqlInsertRequest:
		this.onSqlInsert(req.(*sqlInsertRequest), sender)
	case *sqlSelectRequest:
		this.onSqlSelect(req.(*sqlSelectRequest), sender)
	case *sqlUpdateRequest:
		this.onSqlUpdate(req.(*sqlUpdateRequest), sender)
	case *sqlDeleteRequest:
		this.onSqlDelete(req.(*sqlDeleteRequest), sender)
	case *sqlSubscribeRequest:
		this.onSqlSubscribe(req.(*sqlSubscribeRequest), sender)
	case *sqlUnsubscribeRequest:
		this.onSqlUnsubscribe(req.(*sqlUnsubscribeRequest), sender)
	case *sqlKeyRequest:
		this.onSqlKey(req.(*sqlKeyRequest), sender)
	case *sqlTagRequest:
		this.onSqlTag(req.(*sqlTagRequest), sender)
	}
}

func (this *table) onSqlInsert(req *sqlInsertRequest, sender *responseSender) {
	res := this.sqlInsert(req)
	this.send(sender, res)
}

func (this *table) onSqlSelect(req *sqlSelectRequest, sender *responseSender) {
	this.send(sender, this.sqlSelect(req))
}

func (this *table) onSqlUpdate(req *sqlUpdateRequest, sender *responseSender) {
	this.send(sender, this.sqlUpdate(req))
}

func (this *table) onSqlDelete(req *sqlDeleteRequest, sender *responseSender) {
	this.send(sender, this.sqlDelete(req))
}

func (this *table) onSqlSubscribe(req *sqlSubscribeRequest, sender *responseSender) {
	req.sender = sender
	this.sqlSubscribe(req)
}

func (this *table) onSqlUnsubscribe(req *sqlUnsubscribeRequest, sender *responseSender) {
	req.connectionId = sender.connectionId
	this.send(sender, this.sqlUnsubscribe(req))
}

func (this *table) onSqlKey(req *sqlKeyRequest, sender *responseSender) {
	this.send(sender, this.sqlKey(req))
}

func (this *table) onSqlTag(req *sqlTagRequest, sender *responseSender) {
	this.send(sender, this.sqlTag(req))
}
