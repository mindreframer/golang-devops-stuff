package emitter

import (
	"code.google.com/p/gogoprotobuf/proto"
	"fmt"
	"github.com/cloudfoundry-incubator/dropsonde/events"
)

type EventEmitter interface {
	Emit(events.Event) error
	Close()
}

type eventEmitter struct {
	innerEmitter ByteEmitter
	origin       string
}

func NewEventEmitter(byteEmitter ByteEmitter, origin string) EventEmitter {
	return &eventEmitter{innerEmitter: byteEmitter, origin: origin}
}

func (e *eventEmitter) Emit(event events.Event) error {
	envelope, err := Wrap(event, e.origin)
	if err != nil {
		return fmt.Errorf("Wrap: %v", err)
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("Marshal: %v", err)
	}

	return e.innerEmitter.Emit(data)
}

func (e *eventEmitter) Close() {
	e.innerEmitter.Close()
}
