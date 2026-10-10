package services

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordUsesCost12(t *testing.T) {
	hash, err := HashPassword("fictional pass phrase")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil || cost != 12 || PasswordHashCost != 12 {
		t.Fatalf("cost = %d (err %v), const %d; want 12", cost, err, PasswordHashCost)
	}
	config := AuthConfig{PasswordHash: hash}
	if err := config.VerifyPassword("fictional pass phrase"); err != nil {
		t.Errorf("round trip failed: %v", err)
	}
}

func TestHashPasswordRejectsBlank(t *testing.T) {
	for _, in := range []string{"", " ", "   \t ", "\n"} {
		if hash, err := HashPassword(in); !errors.Is(err, ErrEmptyPassword) || hash != "" {
			t.Errorf("HashPassword(%q) = %q, %v; want ErrEmptyPassword", in, hash, err)
		}
	}
}

func TestHashPasswordKeepsEdgeSpaces(t *testing.T) {
	hash, err := HashPassword(" edge spaces ")
	if err != nil {
		t.Fatal(err)
	}
	config := AuthConfig{PasswordHash: hash}
	if err := config.VerifyPassword(" edge spaces "); err != nil {
		t.Errorf("exact string rejected: %v", err)
	}
	if err := config.VerifyPassword("edge spaces"); err == nil {
		t.Error("trimmed string must not verify")
	}
}

func TestHashPasswordTooLongIsAnError(t *testing.T) {
	if hash, err := HashPassword(strings.Repeat("a", 73)); err == nil || hash != "" {
		t.Errorf("73 bytes: got %q, %v; want an error and no hash", hash, err)
	}
}

func TestVerifyPasswordAcceptsOtherCosts(t *testing.T) {
	old, err := bcrypt.GenerateFromPassword([]byte("fictional pass phrase"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	config := AuthConfig{PasswordHash: string(old)}
	if err := config.VerifyPassword("fictional pass phrase"); err != nil {
		t.Errorf("cost-10 hash no longer verifies: %v", err)
	}
}
