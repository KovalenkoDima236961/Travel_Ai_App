package push

import (
	"crypto/sha256"
	"encoding/hex"
)

// EndpointHash returns a short, stable hash safe for logs/metrics labels.
func EndpointHash(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:])[:16]
}
