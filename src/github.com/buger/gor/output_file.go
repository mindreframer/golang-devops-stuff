package main

import (
	"encoding/gob"
	"log"
	"os"
	"time"
)

type RawRequest struct {
	Timestamp int64
	Request   []byte
}

type FileOutput struct {
	path    string
	encoder *gob.Encoder
	file    *os.File
}

func NewFileOutput(path string) (o *FileOutput) {
	o = new(FileOutput)
	o.path = path
	o.Init(path)

	return
}

func (o *FileOutput) Init(path string) {
	var err error

	o.file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)

	if err != nil {
		log.Fatal(o, "Cannot open file %q. Error: %s", path, err)
	}

	o.encoder = gob.NewEncoder(o.file)
}

func (o *FileOutput) Write(data []byte) (n int, err error) {
	raw := RawRequest{time.Now().UnixNano(), data}

	o.encoder.Encode(raw)

	return len(data), nil
}

func (o *FileOutput) String() string {
	return "File output: " + o.path
}
