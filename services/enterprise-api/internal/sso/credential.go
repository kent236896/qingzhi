package sso

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// Vault encrypts and decrypts federation secrets with AES-GCM.
type Vault struct {
	key []byte
}

// NewVault builds a vault from SSO_SECRETS_KEY (preferred) or a domain-separated
// derivation of JWT_SECRET for local development only.
func NewVault(secretsKey, jwtSecret string) (*Vault, error) {
	raw := strings.TrimSpace(secretsKey)
	if raw == "" {
		if strings.TrimSpace(jwtSecret) == "" {
			return nil, fmt.Errorf("SSO_SECRETS_KEY or JWT_SECRET is required to encrypt federation secrets")
		}
		sum := sha256.Sum256([]byte("qingzhi/sso/secrets/v1|" + jwtSecret))
		return &Vault{key: sum[:]}, nil
	}
	key, err := decodeSecretsKey(raw)
	if err != nil {
		return nil, err
	}
	return &Vault{key: key}, nil
}

func decodeSecretsKey(raw string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) >= 32 {
		sum := sha256.Sum256(decoded)
		return sum[:], nil
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("SSO_SECRETS_KEY must be at least 32 characters or 32-byte base64")
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

// Encrypt returns ciphertext that includes a random nonce prefix.
func (v *Vault) Encrypt(plaintext []byte) ([]byte, error) {
	if v == nil || len(v.key) != 32 {
		return nil, notReady("联邦密钥服务未配置")
	}
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt.
func (v *Vault) Decrypt(ciphertext []byte) ([]byte, error) {
	if v == nil || len(v.key) != 32 {
		return nil, notReady("联邦密钥服务未配置")
	}
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, invalid("密文无效")
	}
	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

func secretHint(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	runes := []rune(secret)
	if len(runes) <= 4 {
		return "****"
	}
	return "****" + string(runes[len(runes)-4:])
}

func randomClientID() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return "qz_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomClientSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
