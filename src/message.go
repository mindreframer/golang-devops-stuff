package main

import (
	"encoding/binary"
	"fmt"
	"io"
)

type (
	MessageType byte
	Message     struct {
		Type MessageType
		Body string
	}
)

const (
	Error MessageType = iota + 1
	Log
	Call
	Hijack
	ReadLineRequest
	ReadLineResponse
)

func write(dst io.Writer, args ...interface{}) error {
	var err error
	for _, arg := range args {
		switch val := arg.(type) {
		// Write strings as length prefixed byte arrays
		case string:
			err = write(dst, uint64(len(val)))
			if err != nil {
				return err
			}
			_, err = dst.Write([]byte(val))
			if err != nil {
				return err
			}
			// Any other types are handle with binary
		default:
			err := binary.Write(dst, binary.BigEndian, arg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func read(src io.Reader, args ...interface{}) error {
	var err error
	for _, arg := range args {
		switch val := arg.(type) {
		// Read strings as length prefixed byte arrays
		case *string:
			var n uint64
			err = read(src, &n)
			if err != nil {
				return err
			}
			bs := make([]byte, int(n))
			_, err = io.ReadAtLeast(src, bs, int(n))
			if err != nil {
				return err
			}
			*val = string(bs)
			// Any other types are handle with binary
		default:
			err := binary.Read(src, binary.BigEndian, val)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Errorf(dst io.Writer, msg string, args ...interface{}) error {
	return write(dst, Error, fmt.Sprintf(msg, args...))
}

func Logf(dst io.Writer, msg string, args ...interface{}) error {
	return write(dst, Log, fmt.Sprintf(msg, args...))
}

func Send(dst io.Writer, msg Message) error {
	return write(dst, msg.Type, msg.Body)
}

func Receive(src io.Reader) (Message, error) {
	var msg Message
	err := read(src, &msg.Type, &msg.Body)
	return msg, err
}
