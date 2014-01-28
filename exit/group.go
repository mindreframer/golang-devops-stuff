package exit

import (
	"sync"
)

type WaitGroup struct {
	sync.WaitGroup
	Quit chan bool
}

func NewWaitGroup() *WaitGroup {
	return &WaitGroup{Quit: make(chan bool)}
}

func (g *WaitGroup) Wait() {
	g.WaitGroup.Add(1)
	<-g.Quit
}

func (g *WaitGroup) Exit() {
	g.close()
	g.WaitGroup.Wait()
}

// Close the quit channel, swallowing a panic if multiple threads
// happen to try this.
func (g *WaitGroup) close() {
	defer func() {
		recover()
	}()
	close(g.Quit)
}
