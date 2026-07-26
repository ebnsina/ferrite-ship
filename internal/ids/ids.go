// Package ids generates short, readable identifiers.
package ids

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns an identifier like "srv_9f2a1c0b44de3711". The prefix makes IDs
// self-describing in logs and URLs.
func New(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic("ids: could not read random bytes: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
