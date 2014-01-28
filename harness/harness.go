package harness

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stripe-ctf/octopus/agent"
	"github.com/stripe-ctf/octopus/log"
	"github.com/stripe-ctf/octopus/sql"
	"github.com/stripe-ctf/octopus/state"
	"github.com/stripe-ctf/octopus/unix"
	"github.com/stripe-ctf/octopus/util"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Harness struct {
	agents  agent.List
	sql     *sql.SQL
	request chan *request
	result  chan *result

	nextSequenceNumber int
}

type result struct {
	reqid int
	node  *agent.Agent
	query string
	start time.Time
	end   time.Time
	resp  []byte // unparsed response
	body  []byte // parsed response
}

type request struct {
	id    int
	node  *agent.Agent
	query string
}

var unixClient = http.Client{
	Transport: &http.Transport{Dial: unix.Dialer},
}

func New(agents agent.List) *Harness {
	sqlPath := filepath.Join(state.Root(), "storage.sql")
	util.EnsureAbsent(sqlPath)

	return &Harness{
		result:  make(chan *result, 5),
		request: make(chan *request, 5),
		sql:     sql.NewSQL(sqlPath),
		agents:  agents,
	}
}

func (h *Harness) Start() {
	if !h.boot() {
		return
	}
	go h.startQueryThreads()
	go h.queryGenerator()
	go h.resultHandler()
}

func (h *Harness) boot() bool {
	rng := state.NewRand("boot")
	bootQuery := h.generateInitialQuery()
	i := rng.Intn(len(h.agents))
	node := h.agents[i]

	for !h.issueQuery(0, node, bootQuery) {
		time.Sleep(100 * time.Millisecond)
		select {
		case <-state.WaitGroup().Quit:
			return false
		default:
		}
	}
	return true
}

func (h *Harness) startQueryThreads() {
	threads := 4 * state.NodeCount()
	for i := 0; i < threads; i++ {
		go h.queryThread()
	}
}

func (h *Harness) queryGenerator() {
	rng := state.NewRand("querier")
	for i := 1; true; i++ {
		query := h.generateQuery(rng)

		agentNo := rng.Intn(len(h.agents))
		node := h.agents[agentNo]
		h.request <- &request{
			id:    i,
			node:  node,
			query: query,
		}
		state.SetLastGeneratedRequest(i)
	}
}

func (h *Harness) queryThread() {
	for {
		req := <-h.request
		for !h.issueQuery(req.id, req.node, req.query) {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

var people = []string{"siddarth", "gdb", "christian", "andy", "carl"}

func (h *Harness) generateInitialQuery() string {
	insert := make([]string, len(people))
	for i, name := range people {
		insert[i] = fmt.Sprintf("INSERT INTO ctf3 (name) VALUES (\"%s\");", name)
	}
	peopleFmt := strings.Join(insert, " ")

	query := fmt.Sprintf(`
CREATE TABLE ctf3
(name STRING PRIMARY KEY,
friendCount INT DEFAULT 0,
requestCount INT DEFAULT 0,
favoriteWord CHAR(15) DEFAULT "");
%s`,
		peopleFmt)
	return strings.TrimLeft(query, "\n")
}

func (h *Harness) generateQuery(rng *rand.Rand) string {
	amount := rng.Intn(100) + 1
	i := rng.Intn(len(people))
	person := people[i]
	word := state.RandomString(rng, 15)
	query := fmt.Sprintf(`UPDATE ctf3 SET friendCount=friendCount+%d, requestCount=requestCount+1, favoriteWord="%s" WHERE name="%s"; SELECT * FROM ctf3;`, amount, word, person)
	return query
}

func (h *Harness) issueQuery(reqid int, node *agent.Agent, query string) bool {
	log.Debugf("[harness] Making request to %v: %#v", node, query)

	b := strings.NewReader(query)
	url := node.ConnectionString + "/sql"

	start := time.Now()
	resp, err := unixClient.Post(url, "application/octet-stream", b)
	end := time.Now()

	if err != nil {
		log.Printf("[harness] Sleeping 100ms after request error from %s (in response to %#v): %s", node, query, err)
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log.Printf("[harness] Error reading body from %v (in response to %#v): %s", node, query, err)
	}

	if resp.StatusCode != 200 {
		log.Printf("[harness] Sleeping 100ms after HTTP %d status code from %s (in response to %#v): %s", resp.StatusCode, node, query, body)
		return false
	}

	log.Debugf("[harness] Received response to %v (%#v): %s", node, query, body)

	h.result <- &result{
		reqid: reqid,
		node:  node,
		query: query,
		start: start,
		end:   end,
		resp:  body,
	}
	return true
}

// Rules for sequence numbers:
//
// - Gaps are temporarily OK, but not in the long run.
func (h *Harness) resultHandler() {
	results := make(map[int]*result)
	for {
		result := <-h.result
		sequenceNumber, body, err := h.parseResponse(result.resp)
		if err != nil {
			h.lose(err.Error())
			return
		}
		result.body = body

		if sequenceNumber < h.nextSequenceNumber {
			h.losef(`[%d] Received an already-processed sequence number from %v in response to %s

Output: %s`, sequenceNumber, result.node, result.query, util.FmtOutput(result.resp))
			return
		}

		if old, ok := results[sequenceNumber]; ok {
			h.losef(`[%d] Received a still-pending sequence number from %v in response to %s

Output: %s

This sequence number was originally received in response to %s

Original output: %s`, sequenceNumber, result.node, result.query, util.FmtOutput(result.resp), old.query, util.FmtOutput(old.resp))
			return
		}

		if sequenceNumber > h.nextSequenceNumber {
			log.Printf("[%d] Result from %v waiting on sequence number %d", sequenceNumber, result.node, h.nextSequenceNumber)
		}

		// Notify SPOF monkey that we got a valid request
		select {
		case state.GotRequest() <- result.reqid:
		default:
		}

		results[sequenceNumber] = result
		h.processPending(results)
	}
}

func (h *Harness) processPending(results map[int]*result) {
	for {
		result, ok := results[h.nextSequenceNumber]
		if !ok {
			return
		}

		output, err := h.sql.Execute("harness", result.query)
		if err != nil {
			h.losef("[%d] Could not execute statement that %v claimed was fine: %s", h.nextSequenceNumber, result.node, err)
		}

		if !bytes.Equal(result.body, output.Stdout) {
			h.losef(`[%d] Received incorrect output from %v for query %s

Output: %s

Correct output: %s`, h.nextSequenceNumber, result.node, result.query, util.FmtOutput(result.body), util.FmtOutput(output.Stdout))
		} else {
			state.RecordCorrectQuery()
			log.Printf(`[harness] [%d] Received correct output from %v for query %s

Output: %s`, h.nextSequenceNumber, result.node, result.query, util.FmtOutput(result.body))
		}

		// Update bookkeeping
		delete(results, h.nextSequenceNumber)
		h.nextSequenceNumber += 1
	}
}

func (h *Harness) lose(msg string) {
	h.losef("%s", msg)
}

func (h *Harness) losef(msg string, v ...interface{}) {
	disqualifier := fmt.Sprintf(msg, v...)
	state.RecordDisqualifier(disqualifier)
	state.WaitGroup().Exit()
}

var matcher *regexp.Regexp = regexp.MustCompile("^(?s)SequenceNumber: (\\d+)\n(.*)$")

func (h *Harness) parseResponse(res []byte) (int, []byte, error) {
	matches := matcher.FindSubmatch(res)
	if matches != nil {
		sequenceNumber, err := strconv.Atoi(string(matches[1]))
		return sequenceNumber, matches[2], err
	} else {
		msg := fmt.Sprintf("Could not parse response: %#v", string(res))
		return 0, nil, errors.New(msg)
	}
}
