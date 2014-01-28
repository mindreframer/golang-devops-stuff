package director

import (
	"fmt"
	"github.com/stripe-ctf/octopus/state"
	"log"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"time"
)

const MONKEY_PERIOD = float64(30*time.Second) / (2 * math.Pi)

// We use reflection to look up monkeys and avoid writing a bunch of boilerplate
func (d *Director) monkey(name string) func(*rand.Rand, float64) bool {
	v := reflect.ValueOf(d)
	name = strings.Title(name)
	method := v.MethodByName(fmt.Sprintf("%sMonkey", name))
	return func(rng *rand.Rand, intensity float64) bool {
		args := []reflect.Value{
			reflect.ValueOf(rng),
			reflect.ValueOf(intensity),
		}
		ret := method.Call(args)
		if len(ret) != 1 {
			return true
		} else {
			return ret[0].Bool()
		}
	}
}

func (d *Director) spawn(name string) {
	rng := state.NewRand("monkey " + name)

	c := d.config[name]
	monkey := d.monkey(name)

	start := time.Now()
	keep_going := true
	time.Sleep(c.offset)

	for keep_going {
		time.Sleep(time.Duration(rng.ExpFloat64() * float64(c.frequency)))
		dt := time.Now().Sub(start)
		intensity := (1 - math.Cos(float64(dt)/MONKEY_PERIOD)) / 2
		keep_going = monkey(rng, intensity)
	}
}

// LatencyMonkey manipulates the latencies of single network links. It does not
// respond to changing intensity.
func (d *Director) LatencyMonkey(rng *rand.Rand, intensity float64) {
	target := d.randomLink(rng)
	latency := d.makeLatency(rng, 1)
	log.Printf("[monkey] Setting latency for %v to %v", target, latency)
	target.SetLatency(latency)
}

// JitterMonkey manipulates the jitter on individual network links. It does not
// respond to changing intensity.
func (d *Director) JitterMonkey(rng *rand.Rand, intensity float64) {
	target := d.randomLink(rng)
	jitter := d.makeJitter(rng, 1)
	log.Printf("[monkey] Setting jitter for %v to %v", target, jitter)
	target.SetJitter(jitter)
}

// LagsplitMonkey creates temporary latency partitions in the network: i.e.,
// links across the partition will be subject to a (potentially large) latency
// penalty for some amount of time.
func (d *Director) LagsplitMonkey(rng *rand.Rand, intensity float64) {
	targets := d.randomPartition(rng)
	latency := d.makeLatency(rng, 5*intensity)
	duration := d.makeDuration(rng, 1000, intensity)
	log.Printf("[monkey] Creating %vms lag split on partition %v for %v",
		latency, targets, duration)
	for _, target := range targets {
		go target.Lag(latency, duration)
	}
}

// LinkMonkey temporarily kills individual network links, dropping traffic that
// might have been in-flight.
func (d *Director) LinkMonkey(rng *rand.Rand, intensity float64) {
	if intensity < 0.1 {
		return
	}
	target := d.randomLink(rng)
	duration := d.makeDuration(rng, 500, intensity)
	log.Printf("[monkey] Killing %v for %v", target, duration)
	go target.Kill(duration)
}

// NetsplitMonkey creates temporary network partitions: i.e., links across the
// partition will be destroyed for some amount of time.
func (d *Director) NetsplitMonkey(rng *rand.Rand, intensity float64) {
	if intensity < 0.1 {
		return
	}
	targets := d.randomPartition(rng)
	duration := d.makeDuration(rng, 1000, intensity)
	log.Printf("[monkey] Killing links in partition %v for %v", targets, duration)
	for _, target := range targets {
		go target.Kill(duration)
	}
}

// UnneighborlyMonkey creates temporary partial partitions in which, for every
// node, its connection to at least one other node is interrupted.
func (d *Director) UnneighborlyMonkey(rng *rand.Rand, intensity float64) {
	if intensity < 0.1 {
		return
	}
	targets := d.randomNeighborLinks(rng)
	duration := d.makeDuration(rng, 1000, intensity)
	log.Printf("[monkey] Killing neighbors %v for %v", targets, duration)
	for _, target := range targets {
		go target.Kill(duration)
	}
}

// FreezeMonkey temporarily freezes agents (similar to pressing Ctrl-Z).
func (d *Director) FreezeMonkey(rng *rand.Rand, intensity float64) {
	if intensity < 0.1 {
		return
	}
	target := d.randomAgent(rng)
	duration := d.makeDuration(rng, 1000, intensity)
	log.Printf("[monkey] Freezing %v for %v", target, duration)
	go target.Stop(duration)
}

// MurderMonkey temporarily terminates agents.
func (d *Director) MurderMonkey(rng *rand.Rand, intensity float64) {
	if intensity < 0.1 {
		return
	}
	target := d.randomAgent(rng)
	duration := d.makeDuration(rng, 1000, intensity)
	log.Printf("[monkey] Murdering %v for %v", target, duration)
	go target.Kill(duration)
}

var spofOrder []int
var spofIndex int

// SpofMonkey detects single points of failure by netsplitting one node at a
// time away from the rest of the cluster and ensuring that the cluster
// continues to make progress.
func (d *Director) SpofMonkey(rng *rand.Rand, intensity float64) bool {
	if spofIndex >= state.NodeCount() {
		return false
	} else if len(spofOrder) != state.NodeCount() {
		// Unfortunately we can't do this in an init() because we defer
		// the parsing of flag arguments until later.
		spofOrder = rng.Perm(state.NodeCount())
	}

	i := spofOrder[spofIndex]
	log.Printf("[monkey] Testing if %v is a single point of failure",
		d.agents[i])

	netsplit := d.net.FindPerimeter([]uint{uint(i)})
	spofIndex++

	d.agents[i].Freeze()

	for _, target := range netsplit {
		target.GoodbyeForever()
	}

	// We need to make sure that the cluster is capable of servicing
	// requests that no member of the cluster has ever seen before in order
	// to ensure that the cluster is making progress. To do this, we
	// determine the request ID of the last request that has been created,
	// and make sure that we see a request that was created *after* that
	// (any requests generated after targetRequestId were necessarily
	// generated after the actions above).
	targetRequestId := state.LastGeneratedRequest()
	for <-state.GotRequest() <= targetRequestId {
	}

	log.Printf("[monkey] %v is (probably) not a single point of failure!",
		d.agents[i])

	for _, target := range netsplit {
		target.WhyHelloThere()
	}

	d.agents[i].Thaw()

	return true
}
