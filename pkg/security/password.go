package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

func HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", fmt.Errorf("пароль должен быть не короче 6 символов")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	const iterations = 120000
	hash := derivePasswordHash([]byte(password), salt, iterations)
	return fmt.Sprintf(
		"v1$%d$%s$%s",
		iterations,
		base64.RawURLEncoding.EncodeToString(salt),
		base64.RawURLEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "v1" {
		return false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 {
		return false
	}

	salt, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}

	expected, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	actual := derivePasswordHash([]byte(password), salt, iterations)
	return hmac.Equal(actual, expected)
}

func derivePasswordHash(password, salt []byte, iterations int) []byte {
	state := sha256.Sum256(append(append([]byte{}, salt...), password...))
	for i := 1; i < iterations; i++ {
		block := append([]byte{}, state[:]...)
		block = append(block, salt...)
		block = append(block, password...)
		state = sha256.Sum256(block)
	}
	return state[:]
}
