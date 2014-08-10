package main

import (
	"fmt"
	"io"
	"time"
)

type Limiter struct {
	writer io.Writer
	limit  int

	currentRPS  int
	currentTime int64
}

func NewLimiter(writer io.Writer, limit int) (l *Limiter) {
	l = new(Limiter)
	l.limit = limit
	l.writer = writer
	l.currentTime = time.Now().UnixNano()

	return
}

func (l *Limiter) Write(data []byte) (n int, err error) {
	if (time.Now().UnixNano() - l.currentTime) > time.Second.Nanoseconds() {
		l.currentTime = time.Now().UnixNano()
		l.currentRPS = 0
	}

	if l.currentRPS >= l.limit {
		return 0, nil
	}

	n, err = l.writer.Write(data)

	l.currentRPS++

	return
}

func (l *Limiter) String() string {
	return fmt.Sprintf("Limiting %s to: %d", l.writer, l.limit)
}
