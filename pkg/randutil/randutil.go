// Package randutil provides cryptographically secure random number and string generation helpers.
package randutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Generates a cryptographically secure random numeric string of the designated length.
func Numeric(length int) (string, error) {
	var s string
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("crypto rand generator error: %w", err)
		}
		s += num.String()
	}
	return s, nil
}
