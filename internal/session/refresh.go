package session

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrReplay dikembalikan saat refresh token yang sudah dirotasi dipakai ulang.
var ErrReplay = errors.New("refresh token dipakai ulang")

const (
	refreshPrefix     = "refresh:"
	refreshUsedPrefix = "refresh_used:"
	userRefreshPrefix = "user_refresh:"

	// refreshReplayTTL adalah masa simpan tombstone untuk mendeteksi replay.
	refreshReplayTTL = 10 * time.Minute
)

// RefreshData adalah payload refresh token di Redis.
type RefreshData struct {
	UserID    int64     `json:"user_id"`
	FamilyID  string    `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshStore mengelola refresh token (dapat dicabut, rotasi single-use)
// beserta index per user agar pencabutan semua sesi O(jumlah sesi user).
type RefreshStore struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewRefreshStore membuat refresh token store.
func NewRefreshStore(client *redis.Client, ttl time.Duration) *RefreshStore {
	return &RefreshStore{redis: client, ttl: ttl}
}

// TTL mengembalikan masa berlaku refresh token.
func (s *RefreshStore) TTL() time.Duration {
	return s.ttl
}

// Create membuat refresh token baru untuk user pada satu family sesi.
func (s *RefreshStore) Create(ctx context.Context, userID int64, familyID string) (string, error) {
	token, err := NewToken()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(RefreshData{UserID: userID, FamilyID: familyID, CreatedAt: time.Now()})
	if err != nil {
		return "", err
	}

	pipe := s.redis.Pipeline()
	pipe.Set(ctx, refreshPrefix+token, payload, s.ttl)
	pipe.SAdd(ctx, userRefreshPrefix+itoa(userID), token)
	pipe.Expire(ctx, userRefreshPrefix+itoa(userID), s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return token, nil
}

// Get mengambil payload refresh token; ErrNotFound jika tidak ada/berakhir.
func (s *RefreshStore) Get(ctx context.Context, token string) (*RefreshData, error) {
	raw, err := s.redis.Get(ctx, refreshPrefix+token).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var data RefreshData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// Touch memperpanjang masa berlaku refresh token (sliding expiration).
func (s *RefreshStore) Touch(ctx context.Context, token string) error {
	return s.redis.Expire(ctx, refreshPrefix+token, s.ttl).Err()
}

// Rotate menukar refresh token lama dengan yang baru (single-use).
// Token lama dihapus dan ditandai tombstone; bila token lama dipakai lagi
// setelah rotasi, seluruh sesi user dicabut (indikasi replay) dan ErrReplay dikembalikan.
func (s *RefreshStore) Rotate(ctx context.Context, token string) (string, *RefreshData, error) {
	data, err := s.Get(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", nil, s.handleReplay(ctx, token)
		}
		return "", nil, err
	}

	newToken, err := NewToken()
	if err != nil {
		return "", nil, err
	}

	payload, err := json.Marshal(RefreshData{UserID: data.UserID, FamilyID: data.FamilyID, CreatedAt: time.Now()})
	if err != nil {
		return "", nil, err
	}

	indexKey := userRefreshPrefix + itoa(data.UserID)
	tombstone, err := json.Marshal(data)
	if err != nil {
		return "", nil, err
	}

	pipe := s.redis.Pipeline()
	pipe.Del(ctx, refreshPrefix+token)
	pipe.SRem(ctx, indexKey, token)
	pipe.Set(ctx, refreshPrefix+newToken, payload, s.ttl)
	pipe.SAdd(ctx, indexKey, newToken)
	pipe.Expire(ctx, indexKey, s.ttl)
	pipe.Set(ctx, refreshUsedPrefix+token, tombstone, refreshReplayTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", nil, err
	}

	return newToken, data, nil
}

// handleReplay mencabut semua sesi user bila token yang dipakai sudah dirotasi.
func (s *RefreshStore) handleReplay(ctx context.Context, token string) error {
	raw, err := s.redis.Get(ctx, refreshUsedPrefix+token).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	var data RefreshData
	if err := json.Unmarshal(raw, &data); err != nil {
		return ErrNotFound
	}

	if err := s.RevokeUser(ctx, data.UserID); err != nil {
		return err
	}

	return ErrReplay
}

// Revoke menghapus satu refresh token (logout perangkat ini).
func (s *RefreshStore) Revoke(ctx context.Context, token string) error {
	data, err := s.Get(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	pipe := s.redis.Pipeline()
	pipe.Del(ctx, refreshPrefix+token)
	pipe.SRem(ctx, userRefreshPrefix+itoa(data.UserID), token)
	_, err = pipe.Exec(ctx)
	return err
}

// RevokeUser mencabut seluruh refresh token milik user (logout semua perangkat).
func (s *RefreshStore) RevokeUser(ctx context.Context, userID int64) error {
	indexKey := userRefreshPrefix + itoa(userID)

	tokens, err := s.redis.SMembers(ctx, indexKey).Result()
	if err != nil {
		return err
	}

	if len(tokens) > 0 {
		keys := make([]string, 0, len(tokens))
		for _, token := range tokens {
			keys = append(keys, refreshPrefix+token)
		}
		if err := s.redis.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}

	return s.redis.Del(ctx, indexKey).Err()
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
