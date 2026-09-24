package twofactor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"

	"agrinaspangan/ebitda-api/internal/crypto"
	"agrinaspangan/ebitda-api/internal/session"
)

func newTestService(t *testing.T) (*Service, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	manager := session.NewManager(client, time.Hour)

	return NewService(nil, manager, crypto.KeyFromSecret("test-key"), "EBITDA Max APN Test"), mr
}

func TestGenerateRecoveryCodes(t *testing.T) {
	codes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(codes) != recoveryCodeCount {
		t.Fatalf("expected %d kode, got %d", recoveryCodeCount, len(codes))
	}

	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		if len(code) != recoveryCodeChars+1 || code[5] != '-' {
			t.Fatalf("format kode salah: %q", code)
		}
		if seen[code] {
			t.Fatalf("kode duplikat: %q", code)
		}
		seen[code] = true

		for _, r := range strings.ReplaceAll(code, "-", "") {
			if !strings.ContainsRune(recoveryCharset, r) {
				t.Fatalf("karakter di luar charset: %q", code)
			}
		}
	}
}

func TestMatchRecoveryCode(t *testing.T) {
	codes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	hashes, err := hashRecoveryCodes(codes)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	messy := strings.ToLower(codes[0][:5] + " " + codes[0][6:])
	remaining, ok := matchRecoveryCode(hashes, messy)
	if !ok {
		t.Fatal("recovery code dengan spasi/huruf kecil seharusnya cocok")
	}
	if len(remaining) != len(hashes)-1 {
		t.Fatalf("expected %d sisa hash, got %d", len(hashes)-1, len(remaining))
	}

	if _, ok := matchRecoveryCode(remaining, codes[0]); ok {
		t.Fatal("kode yang sudah terpakai tidak boleh cocok lagi")
	}

	if _, ok := matchRecoveryCode(hashes, "KODE-SALAH"); ok {
		t.Fatal("kode salah tidak boleh cocok")
	}
}

func TestChallengeLifecycle(t *testing.T) {
	service, mr := newTestService(t)
	ctx := context.Background()

	token, err := service.CreateChallenge(ctx, 99)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	challenge, err := service.ResolveChallenge(ctx, token)
	if err != nil {
		t.Fatalf("resolve challenge: %v", err)
	}
	if challenge.UserID != 99 {
		t.Fatalf("expected user 99, got %d", challenge.UserID)
	}

	// 4 kegagalan pertama: ErrInvalidCode
	for i := 0; i < maxChallengeTries-1; i++ {
		if err := service.FailChallenge(ctx, token, challenge); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("percobaan %d: expected ErrInvalidCode, got %v", i+1, err)
		}
	}
	// Kegagalan ke-5: challenge dihapus
	if err := service.FailChallenge(ctx, token, challenge); !errors.Is(err, ErrTooManyTries) {
		t.Fatalf("expected ErrTooManyTries, got %v", err)
	}
	if _, err := service.ResolveChallenge(ctx, token); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected challenge terhapus, got %v", err)
	}

	// Challenge dengan TTL habis
	token2, err := service.CreateChallenge(ctx, 7)
	if err != nil {
		t.Fatalf("create challenge 2: %v", err)
	}
	mr.FastForward(ChallengeTTL + time.Minute)
	if _, err := service.ResolveChallenge(ctx, token2); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected challenge kadaluarsa, got %v", err)
	}
}

func TestClearChallenge(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	token, err := service.CreateChallenge(ctx, 5)
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	if err := service.ClearChallenge(ctx, token); err != nil {
		t.Fatalf("clear challenge: %v", err)
	}
	if _, err := service.ResolveChallenge(ctx, token); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestTOTPValidation(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Test", AccountName: "user@example.com"})
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	code, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	if !totp.Validate(code, key.Secret()) {
		t.Fatal("kode TOTP yang baru dibuat harus valid")
	}
	if totp.Validate("000000", key.Secret()) {
		t.Fatal("kode TOTP salah tidak boleh valid")
	}
}
