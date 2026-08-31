package oauth_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/bit-issues/backend/internal/oauth"
)

func testEncryptor(t *testing.T) *oauth.Encryptor {
	t.Helper()
	enc, err := oauth.NewEncryptorFromConfig("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=") // 32-byte base64
	if err != nil {
		t.Fatalf("failed to build encryptor: %v", err)
	}
	return enc
}

func TestEncryptor_RoundTrip(t *testing.T) {
	enc := testEncryptor(t)

	cases := []string{
		"secret-access-token",
		"refresh-token-with-symbols-/+=",
		strings.Repeat("x", 500),
	}
	for _, plain := range cases {
		ct, err := enc.Encrypt(plain)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		if ct == plain {
			t.Fatalf("ciphertext equals plaintext: %q", plain)
		}
		got, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if got != plain {
			t.Fatalf("round trip mismatch: got %q want %q", got, plain)
		}
	}
}

func TestEncryptor_Empty(t *testing.T) {
	enc := testEncryptor(t)
	ct, err := enc.Encrypt("")
	if err != nil || ct != "" {
		t.Fatalf("empty encrypt: got %q err %v", ct, err)
	}
	got, err := enc.Decrypt("")
	if err != nil || got != "" {
		t.Fatalf("empty decrypt: got %q err %v", got, err)
	}
}

func TestEncryptor_TamperFails(t *testing.T) {
	enc := testEncryptor(t)
	ct, err := enc.Encrypt("top-secret")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	tampered := ct
	if tampered[0] == 'A' {
		tampered = "B" + tampered[1:]
	} else {
		tampered = "A" + tampered[1:]
	}
	if _, derr := enc.Decrypt(tampered); derr == nil {
		t.Fatalf("expected decryption failure on tampered ciphertext")
	}
}

func TestNewEncryptorFromConfig_Errors(t *testing.T) {
	if _, err := oauth.NewEncryptorFromConfig(""); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if _, err := oauth.NewEncryptorFromConfig("not-a-valid-key!!"); err == nil {
		t.Fatalf("expected error for invalid key encoding")
	}
	if _, err := oauth.NewEncryptor([]byte("tooshort")); err == nil {
		t.Fatalf("expected error for short raw key")
	}
	// Hex 32-byte key is accepted.
	hexKey := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	if _, err := oauth.NewEncryptorFromConfig(hexKey); err != nil {
		t.Fatalf("valid hex key rejected: %v", err)
	}
}

func TestNewEncryptorFromConfig_HexKeyPrecedence(t *testing.T) {
	// A 32-char hex string is also valid base64; ensure it is interpreted as
	// raw hex bytes (16 bytes -> AES-128), not as 24 base64-decoded bytes.
	hexKey := "000102030405060708090a0b0c0d0e0f"
	enc, err := oauth.NewEncryptorFromConfig(hexKey)
	if err != nil {
		t.Fatalf("hex key rejected: %v", err)
	}

	raw, derr := hex.DecodeString(hexKey)
	if derr != nil {
		t.Fatalf("internal: hex decode failed: %v", derr)
	}
	direct, derr := oauth.NewEncryptor(raw)
	if derr != nil {
		t.Fatalf("internal: direct encryptor failed: %v", derr)
	}

	ct, eerr := enc.Encrypt("secret")
	if eerr != nil {
		t.Fatalf("encrypt failed: %v", eerr)
	}
	got, derr := direct.Decrypt(ct)
	if derr != nil {
		t.Fatalf("direct decrypt failed: %v", derr)
	}
	if got != "secret" {
		t.Fatalf("hex key not interpreted as raw bytes: got %q want %q", got, "secret")
	}
}
