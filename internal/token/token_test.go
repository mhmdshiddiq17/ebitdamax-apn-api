package token

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndVerify(t *testing.T) {
	service := NewService("rahasia-test", "ebitda-max-apn", time.Hour)

	raw, expiresAt, err := service.IssueAccess(42, "sesi-abc")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expiresAt harus di masa depan")
	}
	if strings.Count(raw, ".") != 2 {
		t.Fatalf("token JWT harus memiliki 3 bagian, got %q", raw)
	}

	claims, err := service.Verify(raw)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != 42 || claims.SessionID != "sesi-abc" {
		t.Fatalf("claims tidak cocok: %+v", claims)
	}
	if claims.JTI == "" || claims.ExpiresAt.IsZero() {
		t.Fatalf("claims jti/exp kosong: %+v", claims)
	}
}

func TestVerifyExpired(t *testing.T) {
	service := NewService("rahasia-test", "ebitda-max-apn", -2*time.Minute)

	raw, _, err := service.IssueAccess(1, "sesi")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := service.Verify(raw); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestVerifyTampered(t *testing.T) {
	service := NewService("rahasia-test", "ebitda-max-apn", time.Hour)

	raw, _, err := service.IssueAccess(1, "sesi")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	tampered := raw[:len(raw)-2] + "xx"
	if _, err := service.Verify(tampered); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid untuk token diubah, got %v", err)
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	issuer := NewService("rahasia-a", "ebitda-max-apn", time.Hour)
	verifier := NewService("rahasia-b", "ebitda-max-apn", time.Hour)

	raw, _, err := issuer.IssueAccess(1, "sesi")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := verifier.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid untuk secret berbeda, got %v", err)
	}
}

func TestVerifyRejectsWrongAlgorithm(t *testing.T) {
	service := NewService("rahasia-test", "ebitda-max-apn", time.Hour)

	claims := jwt.MapClaims{
		"sub": "1",
		"iss": "ebitda-max-apn",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte("rahasia-test"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := service.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid untuk alg HS384, got %v", err)
	}
}

func TestVerifyRejectsWrongIssuer(t *testing.T) {
	issuer := NewService("rahasia-test", "issuer-lain", time.Hour)
	verifier := NewService("rahasia-test", "ebitda-max-apn", time.Hour)

	raw, _, err := issuer.IssueAccess(1, "sesi")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := verifier.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid untuk issuer berbeda, got %v", err)
	}
}
