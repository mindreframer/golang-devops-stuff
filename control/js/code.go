package js

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
)

type CodeGetter interface {
	GetCode() (string, error)
	GetHash() (string, error)
}

type StringGetter struct {
	code string
	hash string
}

func NewStringGetter(code string) *StringGetter {
	h := md5.New()
	h.Write([]byte(code))
	return &StringGetter{
		code: code,
		hash: hex.EncodeToString(h.Sum(nil)),
	}
}

func (sg *StringGetter) GetCode() (string, error) {
	return sg.code, nil
}

func (sg *StringGetter) GetHash() (string, error) {
	return sg.hash, nil
}

type FileGetter struct {
	path     string
	hash     string
	contents string
	laststat os.FileInfo
}

func NewFileGetter(path string) *FileGetter {
	return &FileGetter{
		path: path,
	}
}

func (fg *FileGetter) GetCode() (string, error) {
	_, err := fg.GetHash()

	if err != nil {
		return "", err
	}

	return fg.contents, nil
}

const SKIP_FILE_CACHE = false

func (fg *FileGetter) GetHash() (string, error) {
	st, err := os.Stat(fg.path)
	if err != nil {
		return "", err
	}

	if SKIP_FILE_CACHE || fg.laststat == nil || st.Size() != fg.laststat.Size() || st.ModTime() != fg.laststat.ModTime() {
		fg.laststat = st

		code, err := ioutil.ReadFile(fg.path)
		if err != nil {
			return "", err
		}
		fg.contents = string(code)
		h := md5.New()
		h.Write([]byte(code))
		fg.hash = hex.EncodeToString(h.Sum(nil))
	}

	return fg.hash, nil
}
