package emitter_performance

import (
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"strings"
	"testing"
	"time"
)

const (
	SECOND = float64(1 * time.Second)
)

type messageFixture struct {
	name                string
	message             string
	logMessageExpected  float64
	logEnvelopeExpected float64
}

func (mf *messageFixture) getExpected(isEnvelope bool) float64 {
	if isEnvelope {
		return mf.logEnvelopeExpected
	}
	return mf.logMessageExpected
}

var messageFixtures = []*messageFixture{
	{"long message", longMessage(), 1 * SECOND, 2 * SECOND},
	{"message with newlines", messageWithNewlines(), 3 * SECOND, 5 * SECOND},
	{"message worst case", longMessage() + "\n", 1 * SECOND, 1 * SECOND},
}

func longMessage() string {
	return strings.Repeat("a", emitter.MAX_MESSAGE_BYTE_SIZE*2)
}

func messageWithNewlines() string {
	return strings.Repeat(strings.Repeat("a", 6*1024)+"\n", 10)
}

type MockLoggregatorClient struct {
	received chan *[]byte
}

func (m MockLoggregatorClient) Send(data []byte) {
	m.received <- &data
}

func (m MockLoggregatorClient) Emit() instrumentation.Context {
	return instrumentation.Context{}
}

func BenchmarkLogEnvelopeEmit(b *testing.B) {
	received := make(chan *[]byte, 1)
	e, _ := emitter.NewEmitter("localhost:3457", "ROUTER", "42", "secret", nil)
	e.LoggregatorClient = &MockLoggregatorClient{received}

	testEmitHelper(b, e, received, true)
}

func testEmitHelper(b *testing.B, e emitter.Emitter, received chan *[]byte, isEnvelope bool) {
	go func() {
		for {
			<-received
		}
	}()

	for _, fixture := range messageFixtures {
		startTime := time.Now().UnixNano()

		for i := 0; i < b.N; i++ {
			e.Emit("appid", fixture.message)
		}
		elapsedTime := float64(time.Now().UnixNano() - startTime)

		expected := fixture.getExpected(isEnvelope)
		if elapsedTime > expected {
			b.Errorf("Elapsed time for %s should have been below %vs, but was %vs", fixture.name, expected/SECOND, float64(elapsedTime)/SECOND)
		}
	}
}
