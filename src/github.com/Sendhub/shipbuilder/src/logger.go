package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

type (
	MessageWriter struct {
		conn net.Conn
	}
	Logger struct {
		writer               io.Writer
		prefix               func() string
		suffix               func() string
		written              bool
		lastEndedWithNewline bool
		lock                 sync.Mutex
	}
	NilLogger struct{}

	Format byte
)

const (
	DIM    Format = 2
	RED    Format = 31
	GREEN  Format = 32
	YELLOW Format = 33
)

func (this *NilLogger) Write(bs []byte) (int, error) {
	return len(bs), nil
}

func (this *MessageWriter) Write(p []byte) (n int, err error) {
	err = Send(this.conn, Message{Log, string(p)})
	n = len(p)
	return
}

func NewMessageLogger(conn net.Conn) io.Writer {
	return &MessageWriter{conn}
}

func NewLogger(writer io.Writer, prefix string) io.Writer {
	return &Logger{
		writer: writer,
		prefix: func() string {
			return prefix
		},
		suffix: func() string {
			return ""
		},
	}
}

func NewFormatter(writer io.Writer, format Format) io.Writer {
	return &Logger{
		writer: writer,
		prefix: func() string {
			return fmt.Sprint("\033[", format, "m")
		},
		suffix: func() string {
			return fmt.Sprint("\033[0m")
		},
	}
}

func NewTimeLogger(writer io.Writer) io.Writer {
	start := time.Now()
	return &Logger{
		writer: writer,
		prefix: func() string {
			now := time.Now()
			seconds := int(math.Ceil(now.Sub(start).Seconds()))
			minutes := seconds / 60
			seconds -= minutes * 60
			return fmt.Sprintf("%d:%02d ", minutes, seconds)
		},
		suffix: func() string {
			return ""
		},
	}
}

func (this *Logger) Write(bs []byte) (int, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	prefix := this.prefix()
	suffix := this.suffix()

	final := bs
	if !this.written || this.lastEndedWithNewline {
		this.written = true
		final = append([]byte(prefix), final...)
	}
	if bytes.HasSuffix(final, []byte{byte('\n')}) {
		final = final[:len(final)-1]
		this.lastEndedWithNewline = true
	} else {
		this.lastEndedWithNewline = false
	}
	final = bytes.Replace(final, []byte("\r\n"), []byte("\n"), -1)
	final = bytes.Replace(final, []byte("\n"), []byte(suffix+"\n"+prefix), -1)
	if this.lastEndedWithNewline {
		final = append(final, []byte(suffix)...)
		final = append(final, '\n')
	}
	n, err := this.writer.Write(final)
	if n > len(bs) {
		n = len(bs)
	}
	return n, err
}
