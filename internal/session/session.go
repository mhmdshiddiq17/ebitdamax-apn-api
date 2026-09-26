package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound dikembalikan saat key state tidak ada / sudah berakhir.
var ErrNotFound = errors.New("session not found")

// Manager mengelola state sementara autentikasi di Redis
// (challenge 2FA/passkey; refresh token menyusul di Pkg 2 Sprint 13).
type Manager struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewManager membuat manager state autentikasi.
func NewManager(client *redis.Client, ttl time.Duration) *Manager {
	return &Manager{redis: client, ttl: ttl}
}

// PutJSON menyimpan value JSON dengan TTL (state sementara, mis. challenge 2FA).
func (m *Manager) PutJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, key, payload, ttl).Err()
}

// GetJSON mengambil value JSON ke dest; ErrNotFound jika tidak ada.
func (m *Manager) GetJSON(ctx context.Context, key string, dest any) error {
	raw, err := m.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

// DeleteKey menghapus key state sementara.
func (m *Manager) DeleteKey(ctx context.Context, key string) error {
	return m.redis.Del(ctx, key).Err()
}

// NewToken membuat token acak 32 byte (hex 64 karakter).
func NewToken() (string, error) {
	return randomID()
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
