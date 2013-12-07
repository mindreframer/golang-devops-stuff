package warden

import (
	"bytes"
	"fmt"

	"code.google.com/p/gogoprotobuf/proto"
)

func Messages(msgs ...proto.Message) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	for _, msg := range msgs {
		payload, err := proto.Marshal(msg)
		if err != nil {
			panic(err.Error())
		}

		message := &Message{
			Type:    TypeForMessage(msg).Enum(),
			Payload: payload,
		}

		messagePayload, err := proto.Marshal(message)
		if err != nil {
			panic("failed to marshal message")
		}

		buf.Write([]byte(fmt.Sprintf("%d\r\n%s\r\n", len(messagePayload), messagePayload)))
	}

	return buf
}

func TypeForMessage(msg proto.Message) Message_Type {
	switch msg.(type) {
	case *ErrorResponse:
		return Message_Error

	case *CreateRequest, *CreateResponse:
		return Message_Create
	case *StopRequest, *StopResponse:
		return Message_Stop
	case *DestroyRequest, *DestroyResponse:
		return Message_Destroy
	case *InfoRequest, *InfoResponse:
		return Message_Info

	case *SpawnRequest, *SpawnResponse:
		return Message_Spawn
	case *LinkRequest, *LinkResponse:
		return Message_Link
	case *RunRequest, *RunResponse:
		return Message_Run
	case *StreamRequest, *StreamResponse:
		return Message_Stream

	case *NetInRequest, *NetInResponse:
		return Message_NetIn
	case *NetOutRequest, *NetOutResponse:
		return Message_NetOut

	case *CopyInRequest, *CopyInResponse:
		return Message_CopyIn
	case *CopyOutRequest, *CopyOutResponse:
		return Message_CopyOut

	case *LimitMemoryRequest, *LimitMemoryResponse:
		return Message_LimitMemory
	case *LimitDiskRequest, *LimitDiskResponse:
		return Message_LimitDisk
	case *LimitBandwidthRequest, *LimitBandwidthResponse:
		return Message_LimitBandwidth

	case *PingRequest, *PingResponse:
		return Message_Ping
	case *ListRequest, *ListResponse:
		return Message_List
	case *EchoRequest, *EchoResponse:
		return Message_Echo
	}

	panic("unknown message type")
}

func RequestMessageForType(t Message_Type) proto.Message {
	switch t {
	case Message_Create:
		return &CreateRequest{}
	case Message_Stop:
		return &StopRequest{}
	case Message_Destroy:
		return &DestroyRequest{}
	case Message_Info:
		return &InfoRequest{}

	case Message_Spawn:
		return &SpawnRequest{}
	case Message_Link:
		return &LinkRequest{}
	case Message_Run:
		return &RunRequest{}
	case Message_Stream:
		return &StreamRequest{}

	case Message_NetIn:
		return &NetInRequest{}
	case Message_NetOut:
		return &NetOutRequest{}

	case Message_CopyIn:
		return &CopyInRequest{}
	case Message_CopyOut:
		return &CopyOutRequest{}

	case Message_LimitMemory:
		return &LimitMemoryRequest{}
	case Message_LimitDisk:
		return &LimitDiskRequest{}
	case Message_LimitBandwidth:
		return &LimitBandwidthRequest{}

	case Message_Ping:
		return &PingRequest{}
	case Message_List:
		return &ListRequest{}
	case Message_Echo:
		return &EchoRequest{}
	}

	panic("unknown message type")
}
