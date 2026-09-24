package crypto

import (
	"errors"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := KeyFromSecret("app-key-rahasia")

	encrypted, err := Encrypt("JBSWY3DPEHPK3PXP", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted == "JBSWY3DPEHPK3PXP" {
		t.Fatal("ciphertext tidak boleh sama dengan plaintext")
	}

	plain, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("expected plaintext asli, got %q", plain)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	encrypted, err := Encrypt("rahasia", KeyFromSecret("kunci-a"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if _, err := Decrypt(encrypted, KeyFromSecret("kunci-b")); err == nil {
		t.Fatal("decrypt dengan kunci berbeda seharusnya gagal")
	}
}

func TestDecryptTamperedFails(t *testing.T) {
	key := KeyFromSecret("kunci")
	encrypted, err := Encrypt("rahasia", key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	tampered := encrypted[:len(encrypted)-2] + "AA"
	if _, err := Decrypt(tampered, key); err == nil {
		t.Fatal("ciphertext yang diubah seharusnya gagal didekripsi")
	}
}

func TestDecryptRejectsForeignFormat(t *testing.T) {
	laravelStyle := "eyJpdiI6IkFiYyIsInZhbHVlIjoiWEREIiwi bWFjIjoiWFlaIn0="

	if _, err := Decrypt(laravelStyle, KeyFromSecret("kunci")); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}
