package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newRefreshStore(t *testing.T) (*RefreshStore, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewRefreshStore(client, time.Hour), mr
}

func TestRefreshLifecycle(t *testing.T) {
	store, mr := newRefreshStore(t)
	ctx := context.Background()

	token, err := store.Create(ctx, 7, "family-a")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	data, err := store.Get(ctx, token)
	if err != nil || data.UserID != 7 || data.FamilyID != "family-a" {
		t.Fatalf("get: %+v, %v", data, err)
	}

	if err := store.Revoke(ctx, token); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := store.Get(ctx, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound setelah revoke, got %v", err)
	}

	// Kadaluarsa
	expiring, err := store.Create(ctx, 9, "family-b")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	mr.FastForward(2 * time.Hour)
	if _, err := store.Get(ctx, expiring); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected refresh kadaluarsa, got %v", err)
	}
}

func TestRotateInvalidatesOldToken(t *testing.T) {
	store, _ := newRefreshStore(t)
	ctx := context.Background()

	token, err := store.Create(ctx, 3, "family")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newToken, data, err := store.Rotate(ctx, token)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if data.UserID != 3 || newToken == token {
		t.Fatalf("rotate tidak menghasilkan token baru: %+v", data)
	}

	if _, err := store.Get(ctx, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("token lama harus tidak valid, got %v", err)
	}
	if _, err := store.Get(ctx, newToken); err != nil {
		t.Fatalf("token baru harus valid: %v", err)
	}
}

func TestRotateReplayRevokesAllUserSessions(t *testing.T) {
	store, _ := newRefreshStore(t)
	ctx := context.Background()

	first, err := store.Create(ctx, 5, "family-1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	second, err := store.Create(ctx, 5, "family-2")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rotated, _, err := store.Rotate(ctx, first)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	// Pakai ulang token lama → indikasi replay → semua sesi user dicabut.
	if _, _, err := store.Rotate(ctx, first); !errors.Is(err, ErrReplay) {
		t.Fatalf("expected ErrReplay, got %v", err)
	}

	for _, token := range []string{rotated, second} {
		if _, err := store.Get(ctx, token); !errors.Is(err, ErrNotFound) {
			t.Fatalf("seluruh sesi harus dicabut, token %q masih valid", token)
		}
	}
}

func TestRevokeUserOnlyAffectsTarget(t *testing.T) {
	store, _ := newRefreshStore(t)
	ctx := context.Background()

	targetA, _ := store.Create(ctx, 1, "family-a")
	targetB, _ := store.Create(ctx, 1, "family-b")
	other, _ := store.Create(ctx, 2, "family-c")

	if err := store.RevokeUser(ctx, 1); err != nil {
		t.Fatalf("revoke user: %v", err)
	}

	for _, token := range []string{targetA, targetB} {
		if _, err := store.Get(ctx, token); !errors.Is(err, ErrNotFound) {
			t.Fatalf("sesi user 1 harus dicabut, token %q masih valid", token)
		}
	}
	if _, err := store.Get(ctx, other); err != nil {
		t.Fatalf("sesi user lain tidak boleh terpengaruh: %v", err)
	}
}

func TestTouchExtendsTTL(t *testing.T) {
	store, mr := newRefreshStore(t)
	ctx := context.Background()

	token, err := store.Create(ctx, 4, "family")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	mr.FastForward(50 * time.Minute)
	if err := store.Touch(ctx, token); err != nil {
		t.Fatalf("touch: %v", err)
	}
	mr.FastForward(50 * time.Minute)

	if _, err := store.Get(ctx, token); err != nil {
		t.Fatalf("token harus masih valid setelah touch: %v", err)
	}
}
