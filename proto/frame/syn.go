package frame

import (
	"io"
)

const (
	maxSynBodySize  = 8
	maxSynFrameSize = headerSize + maxSynBodySize
)

type RStreamSyn struct {
	Header
	body [maxSynBodySize]byte
}

// RelatedStreamId returns the related stream's id.
// A zero value means no related stream was specified
func (f *RStreamSyn) RelatedStreamId() StreamId {
	// XXX: this is wrong, yay, needs to consult flag to determine if anything was read!
	//return protoError("STREAM_SYN flags require frame length at least %d, but length is %d", expectedLength, f.length)
	//return StreamId(order.Uint32(f.body[:4]) & streamMask)
	return StreamId(0)
}

// StreamPriority returns the stream priority set on this frame
func (f *RStreamSyn) StreamPriority() StreamPriority {
	// XXX: this is wrong, yay, needs to consult flag to determine if anything was read!
	//return StreamPriority(order.Uint32(f.body[4:8]) & priorityMask)
	return StreamPriority(0)
}

func (f *RStreamSyn) readFrom(d deserializer) (err error) {
	if _, err = io.ReadFull(d, f.body[:f.Length()]); err != nil {
		return err
	}
	return
}

type WStreamSyn struct {
	Header
	data [maxSynFrameSize]byte
}

func (f *WStreamSyn) writeTo(s serializer) (err error) {
	_, err = s.Write(f.data[:headerSize+f.Length()])
	return
}

func (f *WStreamSyn) Set(streamId, relatedStreamId StreamId, streamPriority StreamPriority, fin bool) (err error) {
	var (
		flags flagsType
		//length int
	)

	// set fin bit
	if fin {
		flags.Set(flagFin)
	}

	/*
		// XXX: fix this
		// validate the related stream
		if relatedStreamId != 0 {
			if relatedStreamId > streamMask {
				err = protoError("Related stream id %d is out of range", relatedStreamId)
				return
			}

			flags.Set(flagRelatedStream)
			length += 4
		}

		// validate the stream priority
		if streamPriority != 0 {
			if streamPriority > priorityMask {
				err = protoError("Priority %d is out of range", streamPriority)
				return
			}

			flags.Set(flagStreamPriority)
			length += 4
		}
	*/

	// make the frame
	if err = f.Header.SetAll(TypeStreamSyn, 0, streamId, flags); err != nil {
		return
	}
	return
}

func NewWStreamSyn() (f *WStreamSyn) {
	f = new(WStreamSyn)
	f.Header = Header(f.data[:headerSize])
	return
}
