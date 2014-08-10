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
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type configuration struct {

	// logger
	LOG_DEBUG bool
	LOG_INFO  bool
	LOG_WARN  bool
	LOG_ERROR bool

	// resources
	CHAN_RESPONSE_SENDER_BUFFER_SIZE          int
	CHAN_TABLE_REQUESTS_BUFFER_SIZE           int
	CHAN_DATASERVICE_REQUESTS_BUFFER_SIZE     int
	PARSER_SQL_INSERT_REQUEST_COLUMN_CAPACITY int
	PARSER_SQL_UPDATE_REQUEST_COLUMN_CAPACITY int
	PARSER_SQL_SELECT_REQUEST_COLUMN_CAPACITY int
	TOKENS_PRODUCER_CAPACITY                  int
	TABLE_COLUMNS_CAPACITY                    int
	TABLE_RECORDS_CAPACITY                    int
	TABLE_GET_RECORDS_BY_TAG_CAPACITY         int
	WAIT_MILLISECOND_SERVER_SHUTDOWN          time.Duration
	WAIT_MILLISECOND_CLI_SHUTDOWN             time.Duration
	DATA_BATCH_SIZE                           int
	NET_READWRITE_BUFFER_SIZE                 int

	// command
	COMMAND string

	// network
	IP   string
	PORT uint

	// run mode
	CLI    bool
	SERVER bool

	flags *flag.FlagSet
}

func defaultConfig() configuration {
	return configuration{

		// logger
		LOG_DEBUG: false,
		LOG_INFO:  true,
		LOG_WARN:  true,
		LOG_ERROR: true,

		// resources
		CHAN_RESPONSE_SENDER_BUFFER_SIZE:          10000,
		CHAN_TABLE_REQUESTS_BUFFER_SIZE:           1000,
		CHAN_DATASERVICE_REQUESTS_BUFFER_SIZE:     1000,
		PARSER_SQL_INSERT_REQUEST_COLUMN_CAPACITY: 10,
		PARSER_SQL_UPDATE_REQUEST_COLUMN_CAPACITY: 10,
		PARSER_SQL_SELECT_REQUEST_COLUMN_CAPACITY: 10,
		TOKENS_PRODUCER_CAPACITY:                  30,
		TABLE_COLUMNS_CAPACITY:                    10,
		TABLE_RECORDS_CAPACITY:                    1000,
		TABLE_GET_RECORDS_BY_TAG_CAPACITY:         20,
		WAIT_MILLISECOND_SERVER_SHUTDOWN:          3000,
		WAIT_MILLISECOND_CLI_SHUTDOWN:             1000,
		DATA_BATCH_SIZE:                           100,
		NET_READWRITE_BUFFER_SIZE:                 2048,

		// command
		COMMAND: "start",

		// network
		IP:   "",
		PORT: 7777,
	}
}

var config = defaultConfig()

var validCommands = map[string]string{
	"start": "",
	"cli":   "",
	"help":  "",
	"stop":  "",
}

func validCommandsUsageString() string {
	str := "["
	for command, _ := range validCommands {
		str += " " + command
	}
	str += " ]"
	return str
}

func (this *configuration) netAddress() string {
	return net.JoinHostPort(this.IP, strconv.Itoa(int(this.PORT)))
}

func (this *configuration) setLogLevel(loglevel string) bool {
	this.LOG_DEBUG = false
	this.LOG_INFO = false
	this.LOG_WARN = false
	this.LOG_ERROR = false
	logLevels := strings.Split(loglevel, ",")
	for _, s := range logLevels {
		switch s {
		case "debug":
			this.LOG_DEBUG = true
		case "info":
			this.LOG_INFO = true
		case "warn":
			this.LOG_WARN = true
		case "error":
			this.LOG_ERROR = true
		default:
			return false
		}
	}
	return true
}

func (this *configuration) processCommandLine(args []string) bool {

	// set up flags
	this.flags = flag.NewFlagSet("pubsubsql", flag.ContinueOnError)
	var loglevel string
	this.flags.StringVar(&loglevel, "loglevel", "info,warn,error", `logging level "debug,info,warn,error"`)
	this.flags.StringVar(&this.IP, "ip", config.IP, "ip address")
	this.flags.UintVar(&this.PORT, "port", config.PORT, "port number")

	// set command
	if len(args) > 0 {
		first := args[0]
		if first[0] != '-' {
			// slide up args
			if len(args) > 1 {
				args = args[1:]
			} else {
				args = nil
			}
			this.COMMAND = first
		}
	}
	if _, contains := validCommands[this.COMMAND]; !contains {
		fmt.Println("invalid command ", this.COMMAND, "\nvalid commands ", validCommandsUsageString())
		return false
	}

	// parse options
	if len(args) > 0 && this.flags.Parse(args) != nil {
		return false
	}

	// set loglevel
	if !this.setLogLevel(loglevel) {
		fmt.Println("invalid --loglevel \"" + loglevel + "\"\n" + this.flags.Lookup("loglevel").Usage)
		return false
	}

	// check if there is extra stuff
	if this.flags.NArg() > 0 {
		fmt.Println("invalid command line arrguments")
		fmt.Println("Usage of pubsubsql: ")
		this.flags.PrintDefaults()
		return false
	}

	return true
}
