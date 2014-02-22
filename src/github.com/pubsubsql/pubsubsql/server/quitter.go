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
	"sync"
	"sync/atomic"
	"time"
)

//
// IQuitter is a generic interface called from a participating goroutine to determine if it should quit.
type IQuitter interface {
	Done() bool
}

//
// Quitter implements a quit (shutdown) protocol.
// When the quit protocol is in progress, all participating goroutines should quit.
type Quitter struct {
	IQuitter
	counter int64
	channel chan int
	done    bool
	mutex   sync.Mutex
}

// NewQuitter returns a new Quitter.
func NewQuitter() *Quitter {
	return &Quitter{
		counter: 0,
		channel: make(chan int),
		done:    false,
	}
}

// Done returns true if the quit protocol is in progress.
func (this *Quitter) Done() bool {
	return this.done
}

// Join causes the calling goroutine to participate in the quit protocol.
func (this *Quitter) Join() {
	atomic.AddInt64(&this.counter, 1)
}

// Leave signals that the participating goroutine has quit.
// Should be called with defer semantics.
func (this *Quitter) Leave() {
	atomic.AddInt64(&this.counter, -1)
}

// Quit notifies all participating goroutines that the quit protocol is in progress.
// It waits until all participating goroutines signal that they have quit or until a timeout occurs.
// Returns false when a timeout has occurred.
func (this *Quitter) Quit(timeout time.Duration) bool {
	this.quit()
	return this.Wait(timeout)
}

// quit is a helper function.
func (this *Quitter) quit() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if !this.done {
		this.done = true
		close(this.channel)
	}
}

// GetChan returns the channel to be used in a go select statement in order to receive a Quit notification.
func (this *Quitter) GetChan() chan int {
	return this.channel
}

// Wait waits until all participating goroutines signal that they have quit or until a timeout occurs.
// Returns false when a timeout has occurred.
func (this *Quitter) Wait(timeout time.Duration) bool {
	if timeout == 0 {
		return atomic.LoadInt64(&this.counter) == 0
	}
	now := time.Now()
	for atomic.LoadInt64(&this.counter) > 0 {
		time.Sleep(time.Millisecond * 10)
		if time.Since(now) > timeout {
			return false
		}
	}
	return true
}

// GoRoutines returns the number of participating goroutines in the quit protocol.
func (this *Quitter) GoRoutines() int64 {
	return atomic.LoadInt64(&this.counter)
}
