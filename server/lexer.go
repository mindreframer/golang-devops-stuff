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

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// tokenType identifies the type of lex tokens.
type tokenType uint8

const (
	tokenTypeError                   tokenType = iota // error occurred
	tokenTypeEOF                                      // last token
	tokenTypeCmdStatus                                // status
	tokenTypeCmdStop                                  // stop
	tokenTypeCmdClose                                 // close
	tokenTypeSqlTable                                 // table name
	tokenTypeSqlColumn                                // column name
	tokenTypeSqlInsert                                // insert
	tokenTypeSqlInto                                  // into
	tokenTypeSqlUpdate                                // update
	tokenTypeSqlSet                                   // set
	tokenTypeSqlDelete                                // delete
	tokenTypeSqlFrom                                  // from
	tokenTypeSqlSelect                                // select
	tokenTypeSqlSubscribe                             // subscribe
	tokenTypeSqlUnsubscribe                           // unsubscribe
	tokenTypeSqlSkip                                  // skip
	tokenTypeSqlWhere                                 // where
	tokenTypeSqlValues                                // values
	tokenTypeSqlStar                                  // *
	tokenTypeSqlEqual                                 // =
	tokenTypeSqlLeftParenthesis                       // (
	tokenTypeSqlRightParenthesis                      // )
	tokenTypeSqlComma                                 // ,
	tokenTypeSqlValue                                 // 'some string' string or continuous sequence of chars delimited by WHITE SPACE | ' | , | ( | )
	tokenTypeSqlValueWithSingleQuote                  // '' becomes ' inside the string, parser will need to replace the string
	tokenTypeSqlKey                                   // key
	tokenTypeSqlTag                                   // tag
	tokenTypeSqlStream                                // stream
	tokenTypeSqlPush                                  // push
	tokenTypeSqlPop                                   // pop
	tokenTypeSqlPeek                                  // peek
	tokenTypeSqlBack                                  // back
	tokenTypeSqlFront                                 // front
	tokenTypeSqlReturning                             // returning
	tokenTypeSqlTopic                                 // topic
	tokenTypeSqlMysql                                 // mysql
	tokenTypeSqlConnect                               // connect
	tokenTypeSqlDisconnect                            // disconnect
)

// String converts tokenType value to a string.
func (typ tokenType) String() string {
	switch typ {
	case tokenTypeError:
		return "tokenTypeError"
	case tokenTypeEOF:
		return "tokenTypeEOF"
	case tokenTypeCmdStatus:
		return "tokenTypeCmdStatus"
	case tokenTypeCmdStop:
		return "tokenTypeCmdStop"
	case tokenTypeCmdClose:
		return "tokenTypeCmdClose"
	case tokenTypeSqlTable:
		return "tokenTypeSqlTable"
	case tokenTypeSqlColumn:
		return "tokenTypeSqlColumn"
	case tokenTypeSqlInsert:
		return "tokenTypeSqlInsert"
	case tokenTypeSqlInto:
		return "tokenTypeSqlInto"
	case tokenTypeSqlUpdate:
		return "tokenTypeSqlUpdate"
	case tokenTypeSqlSet:
		return "tokenTypeSqlSet"
	case tokenTypeSqlDelete:
		return "tokenTypeSqlDelete"
	case tokenTypeSqlFrom:
		return "tokenTypeSqlFrom"
	case tokenTypeSqlSelect:
		return "tokenTypeSqlSelect"
	case tokenTypeSqlSubscribe:
		return "tokenTypeSqlSubscribe"
	case tokenTypeSqlSkip:
		return "tokenTypeSqlSkip"
	case tokenTypeSqlUnsubscribe:
		return "tokenTypeSqlUnsubscribe"
	case tokenTypeSqlWhere:
		return "tokenTypeSqlWhere"
	case tokenTypeSqlValues:
		return "tokenTypeSqlValues"
	case tokenTypeSqlStar:
		return "tokenTypeSqlStar"
	case tokenTypeSqlEqual:
		return "tokenTypeSqlEqual"
	case tokenTypeSqlLeftParenthesis:
		return "tokenTypeSqlLeftParenthesis"
	case tokenTypeSqlRightParenthesis:
		return "tokenTypeSqlRightParenthesis"
	case tokenTypeSqlComma:
		return "tokenTypeSqlComma"
	case tokenTypeSqlValue:
		return "tokenTypeSqlValue"
	case tokenTypeSqlValueWithSingleQuote:
		return "tokenTypeSqlValueWithSingleQuote"
	case tokenTypeSqlKey:
		return "tokenTypeSqlKey"
	case tokenTypeSqlTag:
		return "tokenTypeSqlTag"
	case tokenTypeSqlStream:
		return "tokenTypeSqlStream"
	case tokenTypeSqlPush:
		return "tokenTypeSqlPush"
	case tokenTypeSqlPop:
		return "tokenTypeSqlPush"
	case tokenTypeSqlPeek:
		return "tokenTypeSqlPeek"
	case tokenTypeSqlBack:
		return "tokenTypeSqlBack"
	case tokenTypeSqlFront:
		return "tokenTypeSqlFront"
	case tokenTypeSqlTopic:
		return "tokenTypeSqlTopic"
	case tokenTypeSqlMysql:
		return "tokenTypeSqlMysql"
	case tokenTypeSqlConnect:
		return "tokenTypeSqlConnect"
	case tokenTypeSqlDisconnect:
		return "tokenTypeSqlDisconnect"
	}
	return "not implemented"
}

// token is a symbol representing lexical unit.
type token struct {
	typ tokenType
	// string identified by lexer as a token based on
	// the pattern rule for the tokenType
	val string
}

// String converts token to a string.
func (this token) String() string {
	if this.typ == tokenTypeEOF {
		return "EOF"
	}
	return this.val
}

// tokenConsumer consumes tokens emitted by lexer.
type tokenConsumer interface {
	Consume(t *token)
}

type tokensProducerConsumer struct {
	idx    int
	tokens []*token
}

// String converts tokensProducerConsumer to a string.
func (this tokensProducerConsumer) String() string {
	return fmt.Sprintf(
		"tokensProducerConsumer: idx=%d; tokens(%d)=%s",
		this.idx,
		len(this.tokens),
		this.tokens)
}

func newTokens() *tokensProducerConsumer {
	return &tokensProducerConsumer{
		idx:    0,
		tokens: make([]*token, 0, config.TOKENS_PRODUCER_CAPACITY),
	}
}

func (this *tokensProducerConsumer) reuse() {
	this.idx = 0
	this.tokens = this.tokens[0:0:cap(this.tokens)]
}

func (this *tokensProducerConsumer) Consume(tok *token) {
	this.tokens = append(this.tokens, tok)
}

func (this *tokensProducerConsumer) Produce() *token {
	if this.idx >= len(this.tokens) {
		return &token{
			typ: tokenTypeEOF,
		}
	}
	tok := this.tokens[this.idx]
	this.idx++
	return tok
}

// lexer holds the state of the scanner.
type lexer struct {
	input  string        // the string being scanned
	start  int           // start position of this item
	pos    int           // currently position in the input
	width  int           // width of last rune read from input
	tokens tokenConsumer // consumed tokens
	err    string        // error message
}

// stateFn represents the state of the lexer
// as a function that returns the next state.
type stateFn func(*lexer) stateFn

// Emits an error token and terminates the scan
// by passing back a nil pointer that will be the next state
// terminating lexer.run function
func (this *lexer) errorToken(format string, args ...interface{}) stateFn {
	this.err = fmt.Sprintf(format, args...)
	this.tokens.Consume(&token{tokenTypeError, this.err})
	return nil
}

// Returns true if scan was a success.
func (this *lexer) ok() bool {
	return len(this.err) > 0
}

// Passes a token to the token consumer.
func (this *lexer) emit(t tokenType) {
	this.tokens.Consume(&token{t, this.current()})
}

// Returns current lexeme string.
func (this *lexer) current() string {
	str := this.input[this.start:this.pos]
	this.start = this.pos
	return str
}

// Returns the next rune in the input.
func (this *lexer) next() (rune int32) {
	if this.pos >= len(this.input) {
		this.width = 0
		return 0
	}
	rune, this.width = utf8.DecodeRuneInString(this.input[this.pos:])
	this.pos += this.width
	return rune
}

// Returns whether end was reached in the input.
func (this *lexer) end() bool {
	if this.pos >= len(this.input) {
		return true
	}
	return false
}

// Skips over the pending input before this point.
func (this *lexer) ignore() {
	this.start = this.pos
}

// Steps back one rune.
func (this *lexer) backup() {
	this.pos -= this.width
}

// Returns but does not consume the next rune in the input.
func (this *lexer) peek() int32 {
	rune := this.next()
	this.backup()
	return rune
}

// Determines if rune is valid unicode space character or 0.
func isWhiteSpace(rune int32) bool {
	return (unicode.IsSpace(rune) || rune == 0)
}

// Reads till first white space character
// as defined by isWhiteSpace function
func (this *lexer) scanTillWhiteSpace() {
	for rune := this.next(); !isWhiteSpace(rune); rune = this.next() {
		// void
	}
}

// Skips white space characters in the input.
func (this *lexer) skipWhiteSpaces() {
	for rune := this.next(); unicode.IsSpace(rune); rune = this.next() {
	}
	this.backup()
	this.ignore()
}

// Scans input and matches against the string.
// Returns true if the expected string was matched.
func (this *lexer) match(str string, skip int) bool {
	done := true
	for _, rune := range str {
		if skip > 0 {
			skip--
			continue
		}
		if rune != this.next() {
			done = false
		}
	}
	if !isWhiteSpace(this.peek()) {
		done = false
		this.scanTillWhiteSpace()
	}
	return done
}

// Scans input and tries to match the expected string.
// Returns true if the expected string was matched.
// Does not advance the input if the string was not matched.
func (this *lexer) tryMatch(val string) bool {
	i := 0
	for _, rune := range val {
		i++
		if rune != this.next() {
			for ; i > 0; i-- {
				this.backup()
			}
			return false
		}
	}
	return true
}

// lexMatch matches expected string value emitting the token on success
// and returning passed state function.
func (this *lexer) lexMatch(typ tokenType, value string, skip int, fn stateFn) stateFn {
	if this.match(value, skip) {
		this.emit(typ)
		return fn
	}
	return this.errorToken("Unexpected token:" + this.current())
}

// lexSqlIndentifier scans input for valid sql identifier emitting the token on success
// and returning passed state function.
func (this *lexer) lexSqlIdentifier(typ tokenType, fn stateFn) stateFn {
	this.skipWhiteSpaces()
	// first rune has to be valid unicode letter
	if !unicode.IsLetter(this.next()) {
		return this.errorToken("identifier must begin with a letter " + this.current())
	}
	for rune := this.next(); unicode.IsLetter(rune) || unicode.IsDigit(rune); rune = this.next() {

	}
	this.backup()
	this.emit(typ)
	return fn
}

// lexSqlLeftParenthesis scans input for '(' emitting the token on success
// and returning passed state function.
func (this *lexer) lexSqlLeftParenthesis(fn stateFn) stateFn {
	this.skipWhiteSpaces()
	if this.next() != '(' {
		return this.errorToken("expected ( ")
	}
	this.emit(tokenTypeSqlLeftParenthesis)
	return fn
}

// lexSqlValue scans input for valid sql value emitting the token on success
// and returning passed state function.
func (this *lexer) lexSqlValue(fn stateFn) stateFn {
	this.skipWhiteSpaces()
	if this.end() {
		return this.errorToken("expected value but go eof")
	}
	rune := this.next()
	typ := tokenTypeSqlValue
	// quoted string
	if rune == '\'' {
		this.ignore()
		for rune = this.next(); ; rune = this.next() {
			if rune == '\'' {
				if !this.end() {
					rune = this.next()
					// check for '''
					if rune == '\'' {
						typ = tokenTypeSqlValueWithSingleQuote
					} else {
						// since we read lookahead after single quote that ends the string
						// for lookahead
						this.backup()
						// for single quote which is not part of the value
						this.backup()
						this.emit(typ)
						// now ignore that single quote
						this.next()
						this.ignore()
						//
						return fn
					}
				} else {
					// at the very end
					this.backup()
					this.emit(typ)
					this.next()
					return fn
				}
			}
			if rune == 0 {
				return this.errorToken("string was not delimited")
			}
		}
		// value
	} else {
		for rune = this.next(); !isWhiteSpace(rune) && rune != ',' && rune != ')'; rune = this.next() {
		}
		this.backup()
		this.emit(typ)
		return fn
	}
	return nil
}

// Tries to match expected value returns next state function depending on the match.
func (this *lexer) lexTryMatch(typ tokenType, val string, fnMatch stateFn, fnNoMatch stateFn) stateFn {
	this.skipWhiteSpaces()
	if this.tryMatch(val) {
		this.emit(typ)
		return fnMatch
	}
	return fnNoMatch
}

// WHERE sql where clause scan state functions.

func lexSqlWhereColumn(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlColumn, lexSqlWhereColumnEqual)
}

func lexSqlWhereColumnEqual(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == '=' {
		this.emit(tokenTypeSqlEqual)
		return lexSqlWhereColumnEqualValue
	}
	return this.errorToken("expected = ")
}

func lexSqlWhereColumnEqualValue(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexSqlValue(lexSqlReturning)
}

func lexEof(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.end() {
		return nil
	}
	return this.errorToken("unexpected token at the end of statement")
}

// BEGINNING SQL

// INSERT sql statement scan state functions.

func lexSqlPushInto(this *lexer) stateFn {
	this.skipWhiteSpaces()
	switch this.next() {
	case 'b':
		return this.lexMatch(tokenTypeSqlBack, "back", 1, lexSqlInsertInto)
	case 'f':
		return this.lexMatch(tokenTypeSqlFront, "front", 1, lexSqlInsertInto)
	case 'i':
		return this.lexMatch(tokenTypeSqlInto, "into", 1, lexSqlInsertIntoTable)
	}
	return this.errorToken("unexpected token expected front, back or into")
}

func lexSqlInsertInto(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexMatch(tokenTypeSqlInto, "into", 0, lexSqlInsertIntoTable)
}

func lexSqlInsertIntoTable(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlTable, lexSqlInsertIntoTableLeftParenthesis)
}

func lexSqlInsertIntoTableLeftParenthesis(this *lexer) stateFn {
	return this.lexSqlLeftParenthesis(lexSqlInsertColumn)
}

func lexSqlInsertColumn(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexSqlIdentifier(tokenTypeSqlColumn, lexSqlInsertColumnCommaOrRightParenthesis)
}

func lexSqlInsertColumnCommaOrRightParenthesis(this *lexer) stateFn {
	this.skipWhiteSpaces()
	switch this.next() {
	case ',':
		this.emit(tokenTypeSqlComma)
		return lexSqlInsertColumn
	case ')':
		this.emit(tokenTypeSqlRightParenthesis)
		return lexSqlInsertValues
	}
	return this.errorToken("expected , or ) ")
}

func lexSqlInsertValues(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexMatch(tokenTypeSqlValues, "values", 0, lexSqlInsertValuesLeftParenthesis)
}

func lexSqlInsertValuesLeftParenthesis(this *lexer) stateFn {
	return this.lexSqlLeftParenthesis(lexSqlInsertVal)
}

func lexSqlInsertVal(this *lexer) stateFn {
	return this.lexSqlValue(lexSqlInsertValueCommaOrRigthParenthesis)
}

func lexSqlInsertValueCommaOrRigthParenthesis(this *lexer) stateFn {
	this.skipWhiteSpaces()
	switch this.next() {
	case ',':
		this.emit(tokenTypeSqlComma)
		return lexSqlInsertVal
	case ')':
		this.emit(tokenTypeSqlRightParenthesis)
		return lexSqlReturning
	}
	return this.errorToken("expected , or ) ")
}

// returning

func lexSqlReturning(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.end() {
		return nil
	}
	return this.lexMatch(tokenTypeSqlReturning, "returning", 0, lexSqlReturningStar)
}

func lexSqlReturningStar(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == '*' {
		this.emit(tokenTypeSqlStar)
		return nil
	}
	this.backup()
	return lexSqlReturningColumn(this)
}

func lexSqlReturningColumn(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlColumn, lexSqlReturningCommaOrEnd)
}

func lexSqlReturningCommaOrEnd(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.end() {
		return nil
	}
	if this.next() == ',' {
		this.emit(tokenTypeSqlComma)
		return lexSqlReturningColumn
	}
	return this.errorToken("expected , ")
}

// SELECT sql statement scan state functions.

func lexSqlSelectColumn(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexSqlIdentifier(tokenTypeSqlColumn, lexSqlSelectColumnCommaOrFrom)
}

func lexSqlSelectColumnCommaOrFrom(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == ',' {
		this.emit(tokenTypeSqlComma)
		return lexSqlSelectColumn
	}
	this.backup()
	return lexSqlFrom(this)
}

func lexSqlSelectStar(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == '*' {
		this.emit(tokenTypeSqlStar)
		return lexSqlFrom
	}
	this.backup()
	return lexSqlSelectColumn(this)
}

func lexSqlPopFrom(this *lexer) stateFn {
	this.skipWhiteSpaces()
	// from
	if this.tryMatch("from") {
		this.emit(tokenTypeSqlFrom)
		return lexSqlFromTable
	}
	// *
	if this.next() == '*' {
		this.emit(tokenTypeSqlStar)
		return lexSqlFrom
	}
	this.backup()
	// back
	if this.tryMatch("back") {
		this.emit(tokenTypeSqlBack)
		return lexSqlSelectStar
	}
	// front
	if this.tryMatch("front") {
		this.emit(tokenTypeSqlFront)
		return lexSqlSelectStar
	}
	// columns
	return lexSqlSelectColumn(this)
}

func lexSqlPeekFrom(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == '*' {
		this.emit(tokenTypeSqlStar)
		return lexSqlFrom
	}
	this.backup()
	// back
	if this.tryMatch("back") {
		this.emit(tokenTypeSqlBack)
		return lexSqlSelectStar
	}
	// front
	if this.tryMatch("front") {
		this.emit(tokenTypeSqlFront)
		return lexSqlSelectStar
	}
	// columns
	return lexSqlSelectColumn(this)
}

// UPDATE sql statement scan state functions.

func lexSqlUpdateTable(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexSqlIdentifier(tokenTypeSqlTable, lexSqlUpdateTableSet)
}

func lexSqlUpdateTableSet(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexMatch(tokenTypeSqlSet, "set", 0, lexSqlColumn)
}

func lexSqlColumn(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.end() {
		return nil
	}
	return this.lexSqlIdentifier(tokenTypeSqlColumn, lexSqlColumnEqual)
}

func lexSqlColumnEqual(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == '=' {
		this.emit(tokenTypeSqlEqual)
		return lexSqlColumnEqualValue
	}
	return this.errorToken("expected = ")
}

func lexSqlColumnEqualValue(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexSqlValue(lexSqlCommaOrWhere)
}

func lexSqlCommaOrWhere(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == ',' {
		this.emit(tokenTypeSqlComma)
		return lexSqlColumn
	}
	this.backup()
	return lexSqlWhere
}

// DELETE sql statement scan state functions.

func lexSqlFrom(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexMatch(tokenTypeSqlFrom, "from", 0, lexSqlFromTable)
}

func lexSqlFromTable(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlTable, lexSqlWhere)
}

func lexSqlWhere(this *lexer) stateFn {
	return this.lexTryMatch(tokenTypeSqlWhere, "where", lexSqlWhereColumn, lexSqlReturning)
}

// KEY and TAG sql statement scan state functions.

func lexSqlKeyTable(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlTable, lexSqlKeyColumn)
}

func lexSqlKeyColumn(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlColumn, nil)
}

// SUBSCRIBE

func lexSqlSubscribeSkip(this *lexer) stateFn {
	return this.lexMatch(tokenTypeSqlSkip, "skip", 0, lexSqlSelectStar)
}

func lexSqlSubscribe(this *lexer) stateFn {
	this.skipWhiteSpaces()
	if this.next() == '*' {
		this.backup()
		return lexSqlSelectStar
	}
	this.backup()
	return this.lexTryMatch(tokenTypeSqlSkip, "skip", lexSqlSelectStar, lexSqlTopic)
}

func lexSqlTopic(this *lexer) stateFn {
	return this.lexSqlIdentifier(tokenTypeSqlTopic, nil)
}

// UNSUBSCRIBE

func lexSqlUnsubscribeFrom(this *lexer) stateFn {
	return lexSqlFrom(this)
}

// CONNECT

func lexSqlConnectValue(this *lexer) stateFn {
	this.skipWhiteSpaces()
	return this.lexSqlValue(nil)
}

// END SQL

// Helper function to process subscribe unsubscribe connect disconnect commands.
func lexCommandMysql(this *lexer) stateFn {
	this.skipWhiteSpaces()
	switch this.next() {
	case 's':
		return this.lexMatch(tokenTypeSqlSubscribe, "subscribe", 1, lexSqlSubscribe)
	case 'u':
		return this.lexMatch(tokenTypeSqlUnsubscribe, "unsubscribe", 1, lexSqlUnsubscribeFrom)
	case 'c':
		return this.lexMatch(tokenTypeSqlConnect, "connect", 1, lexSqlConnectValue)
	case 'd':
		return this.lexMatch(tokenTypeSqlDisconnect, "disconnect", 1, nil)
	}
	return this.errorToken("Invalid command:" + this.current())
}

// Helper function to process status stop start commands.
func lexCommandST(this *lexer) stateFn {
	switch this.next() {
	case 'r':
		return this.lexMatch(tokenTypeSqlStream, "stream", 3, lexCommand)
	case 'a':
		return this.lexMatch(tokenTypeCmdStatus, "status", 3, nil)
	case 'o':
		return this.lexMatch(tokenTypeCmdStop, "stop", 3, nil)
	}
	return this.errorToken("Invalid command:" + this.current())
}

// Helper function to process select subscribe status stop start commands.
func lexCommandS(this *lexer) stateFn {
	switch this.next() {
	case 'e':
		return this.lexMatch(tokenTypeSqlSelect, "select", 2, lexSqlSelectStar)
	case 'u':
		return this.lexMatch(tokenTypeSqlSubscribe, "subscribe", 2, lexSqlSubscribe)
	case 't':
		return lexCommandST(this)
	}
	return this.errorToken("Invalid command:" + this.current())
}

// Helper function to process push, pop, peek commands.
func lexCommandP(this *lexer) stateFn {
	switch this.next() {
	case 'u':
		return this.lexMatch(tokenTypeSqlPush, "push", 2, lexSqlPushInto)
	case 'o':
		return this.lexMatch(tokenTypeSqlPop, "pop", 2, lexSqlPopFrom)
	case 'e':
		return this.lexMatch(tokenTypeSqlPeek, "peek", 2, lexSqlPeekFrom)
	}
	return this.errorToken("Invalid command:" + this.current())
}

// Initial state function.
func lexCommand(this *lexer) stateFn {
	this.skipWhiteSpaces()
	switch this.next() {
	case 'u': // update unsubscribe
		if this.next() == 'p' {
			return this.lexMatch(tokenTypeSqlUpdate, "update", 2, lexSqlUpdateTable)
		}
		return this.lexMatch(tokenTypeSqlUnsubscribe, "unsubscribe", 2, lexSqlUnsubscribeFrom)
	case 's': // select subscribe status stop start stream
		return lexCommandS(this)
	case 'i': // insert
		return this.lexMatch(tokenTypeSqlInsert, "insert", 1, lexSqlInsertInto)
	case 'd': // delete
		return this.lexMatch(tokenTypeSqlDelete, "delete", 1, lexSqlFrom)
	case 'k': // key
		return this.lexMatch(tokenTypeSqlKey, "key", 1, lexSqlKeyTable)
	case 't': // tag
		return this.lexMatch(tokenTypeSqlTag, "tag", 1, lexSqlKeyTable)
	case 'c': // close
		return this.lexMatch(tokenTypeCmdClose, "close", 1, nil)
	case 'p': // pop, push, peek
		return lexCommandP(this)
	case 'm': // mysql
		return this.lexMatch(tokenTypeSqlMysql, "mysql", 1, lexCommandMysql)
	}
	return this.errorToken("Invalid command:" + this.current())
}

// Scans the input by executing state function untithis.
// the state is nil
func (this *lexer) run() {
	for state := lexCommand; state != nil; {
		state = state(this)
	}
	this.emit(tokenTypeEOF)
}

// Scans the input by running lexer.
func lex(input string, tokens tokenConsumer) bool {
	lexer := &lexer{
		input:  input,
		tokens: tokens,
	}
	lexer.run()
	return lexer.ok()
}
