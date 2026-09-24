package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestManagerLifecycle(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	manager := NewManager(client, time.Hour)
	ctx := context.Background()

	id, err := manager.Create(ctx, 42)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if len(id) != 64 {
		t.Fatalf("expected 64-char session id, got %d chars", len(id))
	}

	data, err := manager.Get(ctx, id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if data.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", data.UserID)
	}

	if _, err := manager.Get(ctx, "tidak-ada"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown session, got %v", err)
	}

	if err := manager.Touch(ctx, id); err != nil {
		t.Fatalf("touch session: %v", err)
	}

	if err := manager.Destroy(ctx, id); err != nil {
		t.Fatalf("destroy session: %v", err)
	}
	if _, err := manager.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after destroy, got %v", err)
	}
}

func TestManagerExpiry(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	manager := NewManager(client, time.Minute)
	ctx := context.Background()

	id, err := manager.Create(ctx, 7)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	mr.FastForward(2 * time.Minute)

	if _, err := manager.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected expired session to be not found, got %v", err)
	}
}
