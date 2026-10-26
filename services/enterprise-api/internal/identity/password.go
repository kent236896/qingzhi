package identity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen = 6
	maxPasswordLen = 32
)

func hashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func checkPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func validatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < minPasswordLen || length > maxPasswordLen {
		return invalid(fmt.Sprintf("密码长度必须在 %d 到 %d 个字符之间", minPasswordLen, maxPasswordLen))
	}
	return nil
}

func randomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// Keep within password length bounds while remaining high-entropy.
	return base64.RawURLEncoding.EncodeToString(buf)[:32], nil
}
