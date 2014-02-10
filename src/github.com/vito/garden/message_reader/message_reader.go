package message_reader

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"

	"code.google.com/p/gogoprotobuf/proto"

	protocol "github.com/pivotal-cf-experimental/garden/protocol"
)

type WardenError struct {
	Message   string
	Data      string
	Backtrace []string
}

func (e *WardenError) Error() string {
	return e.Message
}

type TypeMismatchError struct {
	Expected protocol.Message_Type
	Received protocol.Message_Type
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf(
		"expected message type %s, got %s\n",
		e.Expected,
		e.Received,
	)
}

func ReadMessage(read *bufio.Reader, response proto.Message) error {
	payload, err := readPayload(read)
	if err != nil {
		return err
	}

	message := &protocol.Message{}
	err = proto.Unmarshal(payload, message)
	if err != nil {
		return err
	}

	// error response from server
	if message.GetType() == protocol.Message_Type(1) {
		errorResponse := &protocol.ErrorResponse{}
		err = proto.Unmarshal(message.Payload, errorResponse)
		if err != nil {
			return errors.New("error unmarshalling error!")
		}

		return &WardenError{
			Message:   errorResponse.GetMessage(),
			Data:      errorResponse.GetData(),
			Backtrace: errorResponse.GetBacktrace(),
		}
	}

	responseType := protocol.TypeForMessage(response)
	if message.GetType() != responseType {
		return &TypeMismatchError{
			Expected: responseType,
			Received: message.GetType(),
		}
	}

	return proto.Unmarshal(message.GetPayload(), response)
}

func ReadRequest(read *bufio.Reader) (proto.Message, error) {
	payload, err := readPayload(read)
	if err != nil {
		return nil, err
	}

	message := &protocol.Message{}
	err = proto.Unmarshal(payload, message)
	if err != nil {
		return nil, err
	}

	request := protocol.RequestMessageForType(message.GetType())

	err = proto.Unmarshal(message.GetPayload(), request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func readPayload(read *bufio.Reader) ([]byte, error) {
	msgHeader, err := read.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	msgLen, err := strconv.ParseUint(string(msgHeader[0:len(msgHeader)-2]), 10, 0)
	if err != nil {
		return nil, err
	}

	payload, err := readNBytes(int(msgLen), read)
	if err != nil {
		return nil, err
	}

	_, err = readNBytes(2, read) // CRLN
	if err != nil {
		return nil, err
	}

	return payload, err
}

func readNBytes(payloadLen int, io *bufio.Reader) ([]byte, error) {
	payload := make([]byte, payloadLen)

	for readCount := 0; readCount < payloadLen; {
		n, err := io.Read(payload[readCount:])
		if err != nil {
			return nil, err
		}

		readCount += n
	}

	return payload, nil
}
