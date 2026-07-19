package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateShortID() string {
	return generateRandomString(6)
}

// GenerateHostToken returns a high-entropy single-use host claim secret.
func GenerateHostToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fallbackID(32)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateRandomString(length int) string {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return fallbackID(length)
		}
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result)
}

func fallbackID(length int) string {
	n := time.Now().UnixNano()
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		out[i] = charset[int(n%int64(len(charset)))]
		n = n/int64(len(charset)) + int64(i+1)*1103515245
	}
	if length >= 6 {
		return string(out)
	}
	return fmt.Sprintf("%s%x", string(out), time.Now().UnixNano())
}
