package network

import (
	"fmt"
	"github.com/stripe-ctf/octopus/agent"
	"github.com/stripe-ctf/octopus/state"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

// A Link is the representation of the network between two hosts. There is
// exactly one Link between every pair of hosts. A Link can have many
// Connections (a logical socket connection between the hosts that was
// established in either direction).
// Links have a latency, which represents the base latency of the link, a jitter
// value, which controls the amount the actual latency of a given message might
// differ from the link's base latency, and a spike, which is exactly the same
// as latency except that it is controlled by a separate process (to cause load
// spikes along netsplits)
// All connections on a given link fail simultaneously.

type Link struct {
	sync.Mutex
	network                *Network
	latency, spike, jitter uint
	kill                   chan bool
	killcount              uint
	agent1, agent2         uint
}

// Label on a network connection
type label struct {
	src, dst uint
	reverse_ bool
}

func (l *label) reverse() *label {
	return &label{l.src, l.dst, !l.reverse_}
}

func (l *label) String() string {
	if !l.reverse_ {
		return fmt.Sprintf("node%d -> node%d", l.src, l.dst)
	} else {
		return fmt.Sprintf("node%d <- node%d", l.src, l.dst)
	}
}

func (link *Link) IsKilled() bool {
	link.Lock()
	defer link.Unlock()
	select {
	case <-link.kill:
		return true
	default:
		return false
	}
}

func (link *Link) Latency() uint {
	link.Lock()
	defer link.Unlock()
	return link.latency
}
func (link *Link) SetLatency(latency uint) {
	link.Lock()
	defer link.Unlock()
	link.latency = latency
}
func (link *Link) Jitter() uint {
	link.Lock()
	defer link.Unlock()
	return link.jitter
}
func (link *Link) SetJitter(jitter uint) {
	link.Lock()
	defer link.Unlock()
	link.jitter = jitter
}
func (link *Link) Delay() time.Duration {
	link.Lock()
	defer link.Unlock()
	jitter := int64(rand.ExpFloat64() * float64(link.jitter))
	latency := jitter + int64(link.spike) + int64(link.latency)
	return time.Duration(latency) * time.Millisecond
}

func (link *Link) connect(l *label, src *fullDuplex, dest string) {
	delay := link.Delay()

	var accept bool
	if link.IsKilled() {
		log.Printf("[network] Refusing to connect %s in %v", l, link.Delay())
		accept = false
	} else {
		log.Printf("[network] Connecting %s in %v", l, link.Delay())
		accept = true
	}

	time.Sleep(delay)

	if accept {
		conn := &connection{link: link, kill: link.kill, source: src}
		conn.establish(l, dest)
	} else {
		src.Close()
	}
}

func (link *Link) Listen() {
	listen1 := agent.SocketPath(link.agent1, link.agent2)
	target1 := agent.SocketPath(link.agent2, link.agent2)
	listen2 := agent.SocketPath(link.agent2, link.agent1)
	target2 := agent.SocketPath(link.agent1, link.agent1)

	label1 := &label{link.agent1, link.agent2, false}
	label2 := &label{link.agent2, link.agent1, false}

	sock1 := link.listen(label1, listen1)
	sock2 := link.listen(label2, listen2)

	go link.accept(label1, sock1, target1)
	go link.accept(label2, sock2, target2)
}

func (link *Link) listen(l *label, address string) net.Listener {
	sock, err := net.Listen("unix", address)
	if err != nil {
		log.Fatalf("listen: %s", err)
	}
	go func() {
		state.WaitGroup().Wait()
		defer state.WaitGroup().Done()

		sock.Close()
	}()

	if err = os.Chmod(address, 0777); err != nil {
		log.Fatalf("chmod: %s", address)
	}

	return sock
}

func (link *Link) accept(l *label, sock net.Listener, target string) {
	for {
		conn, err := sock.Accept()
		if err != nil {
			return
		} else {
			sock := NewSockWrap(conn)
			link.connect(l, sock, target)
		}
	}
}

func (link *Link) Kill(dt time.Duration) {
	link.GoodbyeForever()
	time.Sleep(dt)
	link.WhyHelloThere()
}

func (link *Link) GoodbyeForever() {
	link.Lock()
	defer link.Unlock()

	if link.killcount == 0 {
		close(link.kill)
	}
	link.killcount++
}

func (link *Link) WhyHelloThere() {
	link.Lock()
	defer link.Unlock()

	link.killcount--
	if link.killcount == 0 {
		link.kill = make(chan bool)
	}
}

func (link *Link) Lag(amount uint, dt time.Duration) {
	link.Lock()
	link.spike += amount
	link.Unlock()
	time.Sleep(dt)
	link.Lock()
	link.spike -= amount
	link.Unlock()
}

func (link *Link) String() string {
	return fmt.Sprintf("Link<%v, %v>", link.agent1, link.agent2)
}

func (link *Link) Agents() (uint, uint) {
	return link.agent1, link.agent2
}
