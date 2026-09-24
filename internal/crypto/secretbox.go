// Package crypto menyediakan enkripsi simetris untuk kredensial at-rest
// (secret TOTP 2FA) memakai AES-256-GCM dengan kunci turunan APP_KEY.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

// ErrUnsupportedFormat dikembalikan untuk nilai yang bukan hasil Encrypt
// (mis. secret terenkripsi aplikasi lama/Laravel).
var ErrUnsupportedFormat = errors.New("format terenkripsi tidak didukung")

const prefix = "gcm:"

// KeyFromSecret menurunkan kunci AES-256 (32 byte) dari APP_KEY.
func KeyFromSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// Encrypt mengenkripsi plaintext; hasil selalu berprefix "gcm:".
func Encrypt(plain string, key []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt membuka hasil Encrypt. Format asing (bukan "gcm:") ditolak
// dengan ErrUnsupportedFormat.
func Decrypt(encoded string, key []byte) (string, error) {
	if !strings.HasPrefix(encoded, prefix) {
		return "", ErrUnsupportedFormat
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, prefix))
	if err != nil {
		return "", err
	}

	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	if len(raw) < gcm.NonceSize() {
		return "", ErrUnsupportedFormat
	}

	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
