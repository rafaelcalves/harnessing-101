// Package idsource is the implementation of ports.IDSource (outbound
// port 10).
package idsource

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Random generates opaque, workspace-unique IDs from a cryptographically
// random source. No path, PID, display name, or timestamp is encoded,
// per boundaries.md port 10.
type Random struct{}

func (Random) NewID(ctx context.Context) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
