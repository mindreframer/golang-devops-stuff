package yagnats

import (
	"encoding/json"
	"fmt"
)

type Packet interface {
	Encode() []byte
}

type PingPacket struct{}

func (p *PingPacket) Encode() []byte {
	return []byte("PING\r\n")
}

type PongPacket struct{}

func (p *PongPacket) Encode() []byte {
	return []byte("PONG\r\n")
}

type InfoPacket struct {
	Payload string
}

func (p *InfoPacket) Encode() []byte {
	return []byte(fmt.Sprintf("INFO %s\r\n", p.Payload))
}

type ConnectPacket struct {
	User string
	Pass string
}

type connectionPayload struct {
	User     string `json:"user"`
	Pass     string `json:"pass"`
	Verbose  bool   `json:"verbose"`
	Pedantic bool   `json:"pedantic"`
}

func (p *ConnectPacket) Encode() []byte {
	payload := connectionPayload{
		Verbose:  true,
		Pedantic: true,
		User:     p.User,
		Pass:     p.Pass,
	}

	json, err := json.Marshal(payload)
	if err != nil {
		panic("invalid JSON connect payload")
	}

	return []byte(fmt.Sprintf("CONNECT %s\r\n", json))
}

type OKPacket struct{}

func (p *OKPacket) Encode() []byte {
	return []byte("+OK\r\n")
}

type ERRPacket struct {
	Message string
}

func (p *ERRPacket) Encode() []byte {
	return []byte(fmt.Sprintf("-ERR '%s'\r\n", p.Message))
}

type SubPacket struct {
	Subject string
	Queue   string
	ID      int64
}

func (p *SubPacket) Encode() []byte {
	if p.Queue != "" {
		return []byte(fmt.Sprintf("SUB %s %s %d\r\n", p.Subject, p.Queue, p.ID))
	} else {
		return []byte(fmt.Sprintf("SUB %s %d\r\n", p.Subject, p.ID))
	}
}

type UnsubPacket struct {
	ID int64
}

func (p *UnsubPacket) Encode() []byte {
	return []byte(fmt.Sprintf("UNSUB %d\r\n", p.ID))
}

type PubPacket struct {
	Subject string
	ReplyTo string
	Payload []byte
}

func (p *PubPacket) Encode() []byte {
	if p.ReplyTo != "" {
		return []byte(
			fmt.Sprintf(
				"PUB %s %s %d\r\n%s\r\n",
				p.Subject, p.ReplyTo, len(p.Payload), p.Payload,
			),
		)
	} else {
		return []byte(
			fmt.Sprintf(
				"PUB %s %d\r\n%s\r\n",
				p.Subject, len(p.Payload), p.Payload,
			),
		)
	}
}

type MsgPacket struct {
	Subject string
	SubID   int64
	ReplyTo string
	Payload []byte
}

func (p *MsgPacket) Encode() []byte {
	if p.ReplyTo != "" {
		return []byte(
			fmt.Sprintf(
				"MSG %s %d %s %d\r\n%s\r\n",
				p.Subject, p.SubID, p.ReplyTo, len(p.Payload), p.Payload,
			),
		)
	} else {
		return []byte(
			fmt.Sprintf(
				"MSG %s %d %d\r\n%s\r\n",
				p.Subject, p.SubID, len(p.Payload), p.Payload,
			),
		)
	}
}
