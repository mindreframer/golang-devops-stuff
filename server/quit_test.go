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
	"runtime"
	"testing"
	"time"
)

func Test(t *testing.T) {
	quit := NewQuitter()
	if !quit.Quit(0) {
		t.Errorf("quit.Quit() expected true but got false")
	}
}

func testStoper(quit *Quitter, level int, perLevel int) {
	quit.Join()
	defer quit.Leave()
	level--
	if level < 0 {
		return
	}
	//start other go routines
	for i := 0; i < perLevel; i++ {
		go testStoper(quit, level, perLevel)
	}
	//wait for stop event
	c := quit.GetChan()
	<-c
}

func TestMultiGoroutines(t *testing.T) {
	quit := NewQuitter()
	levels := 5
	perLevel := 5
	go testStoper(quit, levels, perLevel)
	time.Sleep(time.Millisecond * 500)
	debug(fmt.Sprint("goroutines in progress:", quit.GoRoutines()))
	if !quit.Quit(time.Millisecond * 1000) {
		t.Errorf("quit.Quit() expected true but got false")
	}
	debug(fmt.Sprint("goroutines in progress:", quit.GoRoutines()))
}

func TestMultiGoroutinesMultiCores(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	quit := NewQuitter()
	levels := 6
	perLevel := 6
	go testStoper(quit, levels, perLevel)
	time.Sleep(time.Millisecond * 500)
	debug(fmt.Sprint("goroutines in progress:", quit.GoRoutines()))
	if !quit.Quit(time.Millisecond * 1000) {
		t.Errorf("quit.Quit() expected true but got false")
	}
	debug(fmt.Sprint("goroutines in progress:", quit.GoRoutines()))
}
