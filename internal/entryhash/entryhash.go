// Package entryhash derives the non-guessable per-poster token used by the
// "alternate method of entry" URL (skips the survey, goes straight to
// contest entry). The token is not a capability secret on its own — it's
// derived deterministically from the poster ID plus a server-side secret —
// so anyone who reads the source can see the algorithm, but only the
// server can compute valid tokens.
package entryhash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

const length = 8

// Hash returns the 8 lowercase hex character token for posterID.
func Hash(secret string, posterID int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(posterID, 10)))
	sum := hex.EncodeToString(mac.Sum(nil))
	return sum[:length]
}

// Verify reports whether provided is the correct token for posterID,
// using a constant-time comparison to avoid leaking timing information.
func Verify(secret string, posterID int64, provided string) bool {
	if len(provided) != length {
		return false
	}
	expected := Hash(secret, posterID)
	return hmac.Equal([]byte(expected), []byte(provided))
}
