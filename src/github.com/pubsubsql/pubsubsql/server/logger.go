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
	"fmt"
	"log"
	"os"
)

var debugLogger = log.New(os.Stderr, "debug: ", log.LstdFlags)
var infoLogger = log.New(os.Stderr, "info: ", log.LstdFlags)
var warnLogger = log.New(os.Stderr, "warning: ", log.LstdFlags)
var errLogger = log.New(os.Stderr, "error: ", log.LstdFlags)

func debug(v ...interface{}) {
	if config.LOG_DEBUG {
		debugLogger.Output(2, fmt.Sprintln(v...))
	}
}

func loginfo(v ...interface{}) {
	if config.LOG_INFO {
		info(v...)
	}
}

func logwarn(v ...interface{}) {
	if config.LOG_WARN {
		warnLogger.Output(2, fmt.Sprintln(v...))
	}
}

func logerror(v ...interface{}) {
	if config.LOG_ERROR {
		errorx(v...)
	}
}

func info(v ...interface{}) {
	infoLogger.Output(2, fmt.Sprintln(v...))
}

func errorx(v ...interface{}) {
	errLogger.Output(2, fmt.Sprintln(v...))
}
