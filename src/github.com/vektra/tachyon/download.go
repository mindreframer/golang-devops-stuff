package tachyon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DownloadCmd struct {
	Url       string `tachyon:"url,required"`
	Dest      string `tachyon:"dest"`
	Sha256sum string `tachyon:"sha256sum"`
	Once      bool   `tachyon:"once"`
}

func (d *DownloadCmd) Run(env *CommandEnv) (*Result, error) {
	destPath := d.Dest

	var out *os.File
	var err error

	if destPath == "" {
		out, err = env.Env.TempFile("download")
		destPath = out.Name()

		if err != nil {
			return nil, err
		}
	} else {
		if d.Once {
			fi, err := os.Stat(destPath)
			if err == nil {
				r := NewResult(false)
				r.Data.Set("size", fi.Size())
				r.Data.Set("path", destPath)

				return r, nil
			}
		}

		out, err = os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}

	defer out.Close()

	resp, err := http.Get(d.Url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Unable to download '%s', code: %d", d.Url, resp.StatusCode)
	}

	s := sha256.New()

	tee := io.MultiWriter(out, s)

	n, err := io.Copy(tee, resp.Body)
	if err != nil {
		return nil, err
	}

	r := NewResult(true)
	r.Data.Set("size", n)
	r.Data.Set("path", destPath)
	r.Data.Set("sha256sum", hex.EncodeToString(s.Sum(nil)))

	return r, nil
}

func init() {
	RegisterCommand("download", &DownloadCmd{})
}
