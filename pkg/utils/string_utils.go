package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateRandomName(prefix string) string {
	charset := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	str := make([]byte, 10)
	for i := range str {
		str[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s-%s", prefix, str)[:9]
}

func StringToPointer(str string) *string {
	return &str
}
