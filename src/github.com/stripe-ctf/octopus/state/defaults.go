package state

import (
	"bitbucket.org/kardianos/osext"
	"crypto/rand"
	"github.com/stripe-ctf/octopus/log"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func scriptDir() string {
	executable, err := osext.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Dir(executable)
	return dir
}

func isExecutable(file string) bool {
	stat, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false
	}
	mode := stat.Mode()
	return mode.IsRegular() && mode&0111 != 0
}

func sqlclusterPath() string {
	dir := scriptDir()

	// If you're using our build script, sqlcluster will be a
	// directory down from octopus.
	p := filepath.Clean(filepath.Join(dir, "../sqlcluster"))
	if isExecutable(p) {
		return p
	}

	// See whether you built it somewhere in your GOPATH
	search := strings.Split(os.Getenv("GOPATH"), ":")
	for _, d := range search {
		p := filepath.Clean(filepath.Join(d, "src/stripe-ctf.com/sqlcluster/sqlcluster"))
		if isExecutable(p) {
			return p
		}

		p = filepath.Clean(filepath.Join(d, "bin/sqlcluster"))
		if isExecutable(p) {
			return p
		}
	}

	// As a last ditch, see whether it's in our path
	p, err := exec.LookPath("sqlcluster")
	if err == nil {
		return p
	}

	return ""
}

func randomSeed() int64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	return n.Int64()
}
