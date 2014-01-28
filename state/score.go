package state

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
)

type Score struct {
	bytesRead, bytesWritten, conns, correctQueries int64
	disqualifier                                   string
	duration, nodeCount                            int
}

type Results struct {
	// Pretty-printed
	BytesRead, BytesWritten        string
	Conns, CorrectQueries          int64
	Disqualifier                   string
	QueryPoints, BytePoints, Total float64
	Duration, NodeCount            int
	Pretty                         string
}

var score = &Score{}

func New() *Score {
	return &Score{}
}

func RecordRead(n int) {
	score.bytesRead += int64(n)
}

func RecordWrite(n int) {
	score.bytesWritten += int64(n)
}

func RecordConn() {
	score.conns += 1
}

func RecordCorrectQuery() {
	score.correctQueries += 1
}

func RecordDisqualifier(reason string) {
	score.disqualifier = reason
}

func recordConfig() {
	score.duration = int(conf.duration.Seconds())
	score.nodeCount = int(conf.nodeCount)
}

func PrettyPrintResults() string {
	r := assembleResults()
	return r.Pretty
}

func JSONResults() []byte {
	r := assembleResults()

	s, err := json.Marshal(r)
	if err != nil {
		log.Fatalf("Could not convert results to string: %s", err)
	}
	return s
}

func assembleResults() *Results {
	bytesRead := fmt.Sprintf("%s", byteSize(score.bytesRead))
	bytesWritten := fmt.Sprintf("%s", byteSize(score.bytesWritten))

	r := &Results{
		Duration:       score.duration,
		NodeCount:      score.nodeCount,
		BytesRead:      bytesRead,
		BytesWritten:   bytesWritten,
		Disqualifier:   score.disqualifier,
		Conns:          score.conns,
		CorrectQueries: score.correctQueries,
		QueryPoints:    queryPoints(),
		BytePoints:     bytePoints(),
		Total:          Total(),
	}

	d := ""
	a := ""
	if r.Disqualifier != "" {
		d = fmt.Sprintf(`DISQUALIFIED:

%s

 -- -- -- -- -- -- -- -- -- -- -- -- -- --

`, r.Disqualifier)
		a = " (canceled due to disqualification)"
	}
	r.Pretty = fmt.Sprintf(`%sFinal stats (running with %d nodes for %ds):
Bytes read: %s
Bytes written: %s
Connection attempts: %d
Correct queries: %d

Score breakdown:
%.0f points from queries
%.0f points from network traffic

Total:
%.0f points%s
`,
		d,
		r.NodeCount, r.Duration,
		r.BytesRead, r.BytesWritten,
		r.Conns,
		r.CorrectQueries,
		r.QueryPoints,
		r.BytePoints,
		r.Total,
		a,
	)

	return r
}

func queryPoints() float64 {
	return float64(10 * score.correctQueries)
}

func bytePoints() float64 {
	bytes := score.bytesRead + score.bytesWritten
	return -0.5 * math.Sqrt(float64(bytes))
}

func Total() float64 {
	return queryPoints() + bytePoints()
}
