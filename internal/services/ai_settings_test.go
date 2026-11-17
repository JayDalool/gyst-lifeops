package services

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const fakeServiceKey = "sk-test-FAKE0000000000000000000000000000"

func TestAIKeyEncryptionRoundTripAndBinding(t *testing.T) {
	dataKey := bytes.Repeat([]byte{7}, aiDataKeyBytes)
	ct, nonce, err := encryptAIKey(dataKey, AIProviderOpenAI, fakeServiceKey)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, []byte("FAKE0000")) {
		t.Fatal("ciphertext holds the plaintext")
	}
	got, err := decryptAIKey(dataKey, AIProviderOpenAI, ct, nonce)
	if err != nil || got != fakeServiceKey {
		t.Fatalf("round trip: %q %v", got, err)
	}
	// Wrong provider (AAD), wrong key, flipped nonce and flipped ciphertext all fail.
	if _, err := decryptAIKey(dataKey, AIProviderAnthropic, ct, nonce); err == nil {
		t.Error("decrypted under another provider")
	}
	if _, err := decryptAIKey(bytes.Repeat([]byte{8}, aiDataKeyBytes), AIProviderOpenAI, ct, nonce); err == nil {
		t.Error("decrypted with another data key")
	}
	bad := append([]byte(nil), nonce...)
	bad[0] ^= 1
	if _, err := decryptAIKey(dataKey, AIProviderOpenAI, ct, bad); err == nil {
		t.Error("decrypted with a changed nonce")
	}
	if _, err := decryptAIKey(dataKey, AIProviderOpenAI, ct, nonce[:5]); err == nil {
		t.Error("decrypted with a short nonce")
	}
	ct2, nonce2, _ := encryptAIKey(dataKey, AIProviderOpenAI, fakeServiceKey)
	if bytes.Equal(nonce, nonce2) || bytes.Equal(ct, ct2) {
		t.Error("nonce or ciphertext repeated")
	}
}

func TestEnsureDataKeyDanglingSymlinkIsKeyFileError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai_settings.key")
	if err := os.Symlink(filepath.Join(dir, "missing-target"), path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	s := &Service{AIKeyPath: path}
	if _, err := s.ensureDataKey(); !errors.Is(err, ErrAIKeyFile) {
		t.Fatalf("dangling symlink: got %v, want ErrAIKeyFile", err)
	}
}
