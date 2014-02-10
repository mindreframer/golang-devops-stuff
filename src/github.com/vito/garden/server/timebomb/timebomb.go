package timebomb

import (
	"time"

	"github.com/pivotal-cf-experimental/garden/drain"
)

type TimeBomb struct {
	countdown time.Duration
	detonate  func()

	reset    chan bool
	defuse   chan bool
	cooldown *drain.Drain
}

func New(countdown time.Duration, detonate func()) *TimeBomb {
	return &TimeBomb{
		countdown: countdown,
		detonate:  detonate,

		reset:    make(chan bool),
		defuse:   make(chan bool),
		cooldown: drain.New(),
	}
}

func (b *TimeBomb) Strap() {
	go func() {
		for {
			cool := b.waitForCooldown()
			if !cool {
				continue
			}

			select {
			case <-time.After(b.countdown):
				b.detonate()
				return
			case <-b.reset:
			case <-b.defuse:
				return
			}
		}
	}()
}

func (b *TimeBomb) Pause() {
	b.cooldown.Incr()
	b.reset <- true
}

func (b *TimeBomb) Defuse() {
	b.defuse <- true
}

func (b *TimeBomb) Unpause() {
	b.cooldown.Decr()
}

func (b *TimeBomb) waitForCooldown() bool {
	ready := make(chan bool, 1)

	go func() {
		b.cooldown.Wait()
		ready <- true
	}()

	select {
	case <-ready:
		return true
	case <-b.reset:
		return false
	}
}
