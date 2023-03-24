package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateRandomName produces a random name made up of lower case letters and number, prefixed with the given string
// and seprated with a hyphen. The generated name is limited to 9 characters.
func GenerateRandomName(prefix string) string {
	charset := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	str := make([]byte, 10)
	for i := range str {
		str[i] = charset[rand.Intn(len(charset))] //nolint:gosec
	}
	return fmt.Sprintf("%s-%s", prefix, str)[:9]
}

// StringToPointer returns a pointer to the provided string
func StringToPointer(str string) *string {
	return &str
}
