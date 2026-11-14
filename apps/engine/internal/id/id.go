package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a prefixed random id like dlv_9f2c...; n is the number of
// random hex characters after the prefix.
func New(prefix string, n int) string {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)[:n]
}

// Token returns a random hex token without prefix, used for API keys.
func Token(n int) string {
	return New("", n)
}
