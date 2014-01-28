package state

import (
	"bytes"
	realrand "crypto/rand"
	"crypto/sha1"
	"fmt"
	"github.com/stripe-ctf/octopus/log"
	"math"
	"math/big"
	"math/rand"
)

func NewRand(name string) *rand.Rand {
	source := fmt.Sprintf("%s-%d", name, Seed())
	image := sha1.Sum([]byte(source))
	b := bytes.NewReader(image[:])
	n, err := realrand.Int(b, big.NewInt(math.MaxInt64))
	if err != nil {
		log.Fatalf("Bug in seed derivation: %s", err)
	}
	log.Debugf("Seed for %v (source %v) is %v", name, source, n.Int64())
	return rand.New(rand.NewSource(n.Int64()))
}

var alphabet = "abcdefghijkmnpqrstuvwxyz01234567890"

// There are lots of ways of doing this faster, but this is good
// enough.
func RandomString(rng *rand.Rand, size int) string {
	out := make([]byte, size)
	for i := 0; i < size; i++ {
		out[i] = alphabet[rng.Intn(size)]
	}
	return string(out)
}
