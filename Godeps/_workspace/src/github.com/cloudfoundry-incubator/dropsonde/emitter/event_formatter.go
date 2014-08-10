package emitter

import (
	"code.google.com/p/gogoprotobuf/proto"
	"errors"
	"github.com/cloudfoundry-incubator/dropsonde/events"
)

func Wrap(e events.Event, origin string) (*events.Envelope, error) {
	if origin == "" {
		return nil, errors.New("Event not emitted due to missing origin information")
	}

	envelope := &events.Envelope{Origin: proto.String(origin)}

	switch e.(type) {
	case *events.Heartbeat:
		envelope.EventType = events.Envelope_Heartbeat.Enum()
		envelope.Heartbeat = e.(*events.Heartbeat)
	case *events.HttpStart:
		envelope.EventType = events.Envelope_HttpStart.Enum()
		envelope.HttpStart = e.(*events.HttpStart)
	case *events.HttpStop:
		envelope.EventType = events.Envelope_HttpStop.Enum()
		envelope.HttpStop = e.(*events.HttpStop)
	default:
		return nil, errors.New("Cannot create envelope for unknown event type")
	}

	return envelope, nil
}
