package rand

import (
	"crypto/rand"
	"encoding/base64"
)

const (
	RememberTokenBytes = 32
)

// Creates a RememberToken with the default size of 32 bytes
func RememberToken() (string, error) {
	return String(RememberTokenBytes)
}

func Bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// String creates a slice of n random bytes (using crypto/rand) and encodes them to a base64 string.
func String(n int) (string, error) {
	bytes, err := Bytes(n)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func NumberOfBytes(base64String string) (int, error) {
	b, err := base64.URLEncoding.DecodeString(base64String)
	if err != nil {
		return -1, err
	}
	return len(b), nil
}