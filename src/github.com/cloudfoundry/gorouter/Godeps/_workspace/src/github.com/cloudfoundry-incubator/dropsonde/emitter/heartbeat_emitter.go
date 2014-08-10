package emitter

import (
	"code.google.com/p/gogoprotobuf/proto"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var HeartbeatInterval = 1 * time.Second

func init() {
	intervalOverride, err := strconv.ParseFloat(os.Getenv("DROPSONDE_HEARTBEAT_INTERVAL_SECS"), 64)
	if err == nil {
		HeartbeatInterval = time.Duration(intervalOverride*1000) * time.Millisecond
	}
}

type heartbeatEmitter struct {
	instrumentedEmitter InstrumentedEmitter
	innerHbEmitter      ByteEmitter
	stopChan            chan struct{}
	origin              string
	sync.Mutex
	closed bool
}

func NewHeartbeatEmitter(emitter ByteEmitter, origin string) (ByteEmitter, error) {
	instrumentedEmitter, err := NewInstrumentedEmitter(emitter)
	if err != nil {
		return nil, err
	}

	hbEmitter := &heartbeatEmitter{
		instrumentedEmitter: instrumentedEmitter,
		innerHbEmitter:      emitter,
		origin:              origin,
		stopChan:            make(chan struct{}),
	}

	go hbEmitter.generateHeartbeats(HeartbeatInterval)
	runtime.SetFinalizer(hbEmitter, (*heartbeatEmitter).Close)

	return hbEmitter, nil
}

func (e *heartbeatEmitter) Emit(data []byte) error {
	return e.instrumentedEmitter.Emit(data)
}

func (e *heartbeatEmitter) Close() {
	e.Lock()
	defer e.Unlock()

	if e.closed {
		return
	}

	e.closed = true
	close(e.stopChan)
}

func (e *heartbeatEmitter) generateHeartbeats(heartbeatInterval time.Duration) {
	defer e.instrumentedEmitter.Close()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			hbEvent := e.instrumentedEmitter.GetHeartbeatEvent()
			hbEnvelope, err := Wrap(hbEvent, e.origin)
			if err != nil {
				log.Printf("Failed to wrap heartbeat event: %v\n", err)
				break
			}

			hbData, err := proto.Marshal(hbEnvelope)
			if err != nil {
				log.Printf("Failed to marshal heartbeat event: %v\n", err)
				break
			}

			err = e.innerHbEmitter.Emit(hbData)
			if err != nil {
				log.Printf("Problem while emitting heartbeat data: %v\n", err)
			}
		}
	}
}
