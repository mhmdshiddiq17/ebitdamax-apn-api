package passkey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"

	"agrinaspangan/ebitda-api/internal/session"
)

func newTestService(t *testing.T) (*Service, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	manager := session.NewManager(client, time.Hour)

	service, err := NewService(nil, manager, Config{
		RPID:          "localhost",
		RPDisplayName: "Test",
		RPOrigins:     []string{"http://localhost:3000"},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	return service, mr
}

func TestDisabledWithoutConfig(t *testing.T) {
	service, err := NewService(nil, nil, Config{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if service.Enabled() {
		t.Fatal("service tanpa config harus nonaktif")
	}
	if _, _, err := service.BeginRegistration(context.Background(), nil); !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

func TestEnabledWithConfig(t *testing.T) {
	service, _ := newTestService(t)
	if !service.Enabled() {
		t.Fatal("service dengan config valid harus aktif")
	}
}

func TestChallengeRoundtrip(t *testing.T) {
	service, mr := newTestService(t)
	ctx := context.Background()

	data := &webauthn.SessionData{Challenge: "abc123"}
	token, err := service.storeChallenge(ctx, registerKeyPrefix, data)
	if err != nil {
		t.Fatalf("store challenge: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("token harus 64 karakter, got %d", len(token))
	}

	loaded, err := service.loadChallenge(ctx, registerKeyPrefix, token)
	if err != nil {
		t.Fatalf("load challenge: %v", err)
	}
	if loaded.Challenge != "abc123" {
		t.Fatalf("challenge tidak cocok: %q", loaded.Challenge)
	}

	// Token tak dikenal
	if _, err := service.loadChallenge(ctx, registerKeyPrefix, "tidak-ada"); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}

	// Token kosong
	if _, err := service.loadChallenge(ctx, registerKeyPrefix, ""); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound untuk token kosong, got %v", err)
	}

	// Kadaluarsa
	mr.FastForward(ceremonyTTL + time.Minute)
	if _, err := service.loadChallenge(ctx, registerKeyPrefix, token); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected challenge kadaluarsa, got %v", err)
	}
}

func TestWebAuthnUserAdapter(t *testing.T) {
	adapter := &webauthnUser{
		id:          42,
		name:        "manager@agrinas.test",
		displayName: "Manager KDKMP",
	}

	if string(adapter.WebAuthnID()) != "42" {
		t.Fatalf("WebAuthnID salah: %q", adapter.WebAuthnID())
	}
	if adapter.WebAuthnName() != "manager@agrinas.test" {
		t.Fatalf("WebAuthnName salah: %q", adapter.WebAuthnName())
	}
	if adapter.WebAuthnDisplayName() != "Manager KDKMP" {
		t.Fatalf("WebAuthnDisplayName salah: %q", adapter.WebAuthnDisplayName())
	}
	if len(adapter.WebAuthnCredentials()) != 0 {
		t.Fatal("credentials awal harus kosong")
	}
}

func TestEncodeCredentialID(t *testing.T) {
	encoded := encodeCredentialID([]byte{1, 2, 3, 250, 251, 252})
	if encoded != "AQID-vv8" {
		t.Fatalf("base64url salah: %q", encoded)
	}
}
