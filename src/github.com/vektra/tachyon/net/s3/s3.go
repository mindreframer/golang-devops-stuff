package s3

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/vektra/tachyon"
	"io"
	"os"
	"time"
)

type S3 struct {
	Bucket      string `tachyon:"bucket,required"`
	PutFile     string `tachyon:"put_file"`
	GetFile     string `tachyon:"get_file"`
	At          string `tachyon:"at"`
	Public      bool   `tachyon:"public"`
	ContentType string `tachyon:"content_type"`
	Writable    bool   `tachyon:"writable"`
	GZip        bool   `tachyon:"gzip"`
}

func (s *S3) Run(env *tachyon.CommandEnv) (*tachyon.Result, error) {
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	c := s3.New(auth, aws.USWest2)
	b := c.Bucket(s.Bucket)

	res := tachyon.NewResult(true)

	res.Add("bucket", s.Bucket)
	res.Add("remote", s.At)

	if s.PutFile != "" {
		path := env.Paths.File(s.PutFile)

		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		if f == nil {
			return nil, fmt.Errorf("Unknown local file %s", s.PutFile)
		}

		defer f.Close()

		var perm s3.ACL

		if s.Public {
			if s.Writable {
				perm = s3.PublicReadWrite
			} else {
				perm = s3.PublicRead
			}
		} else {
			perm = s3.Private
		}

		ct := s.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}

		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}

		var (
			input io.Reader
			opts  s3.Options
			size  int64
		)

		h := md5.New()

		if s.GZip {
			var buf bytes.Buffer

			z := gzip.NewWriter(io.MultiWriter(h, &buf))

			_, err = io.Copy(z, f)
			if err != nil {
				return nil, err
			}

			z.Close()

			opts.ContentEncoding = "gzip"

			input = &buf
			size = int64(buf.Len())
		} else {
			input = io.TeeReader(f, h)
			size = fi.Size()
		}

		err = b.PutReader(s.At, input, size, ct, perm, opts)

		rep, err := b.Head(s.At, nil)
		if err != nil {
			return nil, err
		}

		localMD5 := hex.EncodeToString(h.Sum(nil))

		res.Add("wrote", size)
		res.Add("local", s.PutFile)
		res.Add("md5", localMD5)

		etag := rep.Header.Get("ETag")
		if etag != "" {
			etag = etag[1 : len(etag)-1]

			if localMD5 != etag {
				return nil, fmt.Errorf("corruption uploading file detected")
			}
		}

	} else if s.GetFile != "" {
		f, err := os.OpenFile(s.GetFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		defer f.Close()

		i, err := b.GetReader(s.At)
		if err != nil {
			return nil, err
		}

		defer i.Close()

		n, err := io.Copy(f, i)
		if err != nil {
			return nil, err
		}

		res.Add("read", n)
		res.Add("local", s.GetFile)
	} else {
		return nil, fmt.Errorf("Specify put_file or get_file")
	}

	return res, nil
}

func init() {
	tachyon.RegisterCommand("s3", &S3{})
}
