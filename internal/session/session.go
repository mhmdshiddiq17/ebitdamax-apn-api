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

// ErrNotFound dikembalikan saat session tidak ada / sudah berakhir.
var ErrNotFound = errors.New("session not found")

const keyPrefix = "session:"

// Data adalah payload session yang disimpan di Redis.
type Data struct {
	UserID int64 `json:"user_id"`
}

// Manager mengelola session berbasis Redis (stateful, sliding expiration).
type Manager struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewManager membuat session manager baru.
func NewManager(client *redis.Client, ttl time.Duration) *Manager {
	return &Manager{redis: client, ttl: ttl}
}

// Create membuat session baru dan mengembalikan id-nya.
func (m *Manager) Create(ctx context.Context, userID int64) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(Data{UserID: userID})
	if err != nil {
		return "", err
	}

	if err := m.redis.Set(ctx, keyPrefix+id, payload, m.ttl).Err(); err != nil {
		return "", err
	}

	return id, nil
}

// Get mengambil payload session; ErrNotFound jika tidak ada.
func (m *Manager) Get(ctx context.Context, id string) (*Data, error) {
	raw, err := m.redis.Get(ctx, keyPrefix+id).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// Touch memperpanjang masa berlaku session (sliding expiration).
func (m *Manager) Touch(ctx context.Context, id string) error {
	return m.redis.Expire(ctx, keyPrefix+id, m.ttl).Err()
}

// Destroy menghapus session (logout).
func (m *Manager) Destroy(ctx context.Context, id string) error {
	return m.redis.Del(ctx, keyPrefix+id).Err()
}

// DestroyUserSessions mencabut seluruh sesi aktif milik satu akun.
func (m *Manager) DestroyUserSessions(ctx context.Context, userID int64) error {
	// ponytail: password changes scan active sessions; add a per-user index if session volume makes this slow.
	var cursor uint64
	for {
		keys, next, err := m.redis.Scan(ctx, cursor, keyPrefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			values, err := m.redis.MGet(ctx, keys...).Result()
			if err != nil {
				return err
			}

			toDelete := make([]string, 0, len(keys))
			for index, value := range values {
				raw, ok := value.(string)
				if !ok {
					continue
				}
				var data Data
				if json.Unmarshal([]byte(raw), &data) == nil && data.UserID == userID {
					toDelete = append(toDelete, keys[index])
				}
			}
			if len(toDelete) > 0 {
				if err := m.redis.Del(ctx, toDelete...).Err(); err != nil {
					return err
				}
			}
		}

		cursor = next
		if cursor == 0 {
			return nil
		}
	}
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
