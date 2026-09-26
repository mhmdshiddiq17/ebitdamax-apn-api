package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestJSONStateRoundtrip(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	manager := NewManager(client, time.Hour)
	ctx := context.Background()

	type payload struct {
		UserID   int64 `json:"user_id"`
		Attempts int   `json:"attempts"`
	}

	if err := manager.PutJSON(ctx, "twofactor:challenge:abc", payload{UserID: 7, Attempts: 0}, time.Minute); err != nil {
		t.Fatalf("put: %v", err)
	}

	var loaded payload
	if err := manager.GetJSON(ctx, "twofactor:challenge:abc", &loaded); err != nil {
		t.Fatalf("get: %v", err)
	}
	if loaded.UserID != 7 {
		t.Fatalf("payload tidak cocok: %+v", loaded)
	}

	if err := manager.DeleteKey(ctx, "twofactor:challenge:abc"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := manager.GetJSON(ctx, "twofactor:challenge:abc", &loaded); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Kadaluarsa
	if err := manager.PutJSON(ctx, "twofactor:challenge:exp", payload{UserID: 1}, time.Minute); err != nil {
		t.Fatalf("put: %v", err)
	}
	mr.FastForward(2 * time.Minute)
	if err := manager.GetJSON(ctx, "twofactor:challenge:exp", &loaded); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected state kadaluarsa, got %v", err)
	}
}

func TestNewToken(t *testing.T) {
	first, err := NewToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if len(first) != 64 || first == second {
		t.Fatalf("token tidak valid: %q vs %q", first, second)
	}
}
