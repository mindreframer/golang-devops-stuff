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

import "testing"
import "strconv"

func ASSERT_TRUE(t *testing.T, value bool, message string) {
	if !value {
		t.Error(message + ": Expected true but got false")
	}
}

func ASSERT_FALSE(t *testing.T, value bool, message string) {
	if value {
		t.Error(message + ": Expected false but got true")
	}
}

func TestConfigCommand(t *testing.T) {
	// connect
	args := []string{"cli"}
	c := new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.COMMAND == "cli", "command cli")
	// help
	args = []string{"help"}
	c = new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.COMMAND == "help", "command help")
	// start
	args = []string{"start"}
	c = new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.COMMAND == "start", "command start")
	// empty
	c = new(configuration)
	ASSERT_FALSE(t, c.processCommandLine(nil), "processCommandLine")
	ASSERT_TRUE(t, c.COMMAND == "", "empty command")
	// invalid command
	args = []string{"dosomething"}
	c = new(configuration)
	ASSERT_FALSE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.COMMAND == "dosomething", "command dosomething")
}

func TestConfigLogLevel(t *testing.T) {
	// debug
	args := []string{"start", "--loglevel", "debug"}
	c := new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.LOG_DEBUG == true, "debug: debug")
	ASSERT_TRUE(t, c.LOG_INFO == false, "debug: info")
	ASSERT_TRUE(t, c.LOG_WARN == false, "debug: warn")
	ASSERT_TRUE(t, c.LOG_ERROR == false, "debug: error")
	// info
	args = []string{"start", "--loglevel", "info"}
	c = new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.LOG_DEBUG == false, "info: debug")
	ASSERT_TRUE(t, c.LOG_INFO == true, "info: info")
	ASSERT_TRUE(t, c.LOG_WARN == false, "info: warn")
	ASSERT_TRUE(t, c.LOG_ERROR == false, "info: error")
	// warn
	args = []string{"start", "--loglevel", "warn"}
	c = new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.LOG_DEBUG == false, "warn: debug")
	ASSERT_TRUE(t, c.LOG_INFO == false, "warn: info")
	ASSERT_TRUE(t, c.LOG_WARN == true, "warn: warn")
	ASSERT_TRUE(t, c.LOG_ERROR == false, "warn: error")
	// error
	args = []string{"start", "--loglevel", "error"}
	c = new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.LOG_DEBUG == false, "error: debug")
	ASSERT_TRUE(t, c.LOG_INFO == false, "error: info")
	ASSERT_TRUE(t, c.LOG_WARN == false, "error: warn")
	ASSERT_TRUE(t, c.LOG_ERROR == true, "error: error")
	// all
	args = []string{"start", "--loglevel", "error,warn,info,debug"}
	c = new(configuration)
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.LOG_DEBUG == true, "debug")
	ASSERT_TRUE(t, c.LOG_INFO == true, "info")
	ASSERT_TRUE(t, c.LOG_WARN == true, "warn")
	ASSERT_TRUE(t, c.LOG_ERROR == true, "error")
	// invalid
	args = []string{"start", "--loglevel", "error,warn,aaaaa,info,debug"}
	c = new(configuration)
	ASSERT_FALSE(t, c.processCommandLine(args), "processCommandLine")
}

func TestConfigNetwork(t *testing.T) {
	ip := "255.255.255.6"
	port := 1230
	args := []string{"--ip", ip, "--port", strconv.Itoa(port)}
	c := defaultConfig()
	ASSERT_TRUE(t, c.processCommandLine(args), "processCommandLine")
	ASSERT_TRUE(t, c.IP == ip, "ip")
	ASSERT_TRUE(t, int(c.PORT) == port, "port")
}

func TestConfigInvalid(t *testing.T) {
	args := []string{"--option1"}
	c := defaultConfig()
	ASSERT_FALSE(t, c.processCommandLine(args), "invalid option")
	//
	args = []string{"-port", "1212", "345", "hello", "bla bla bal"}
	c = defaultConfig()
	ASSERT_FALSE(t, c.processCommandLine(args), "invalid arguments")
}
