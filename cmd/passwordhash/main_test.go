package main

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"lifeops-tracker/internal/services"
)

func TestRunCost12AndOnlyHashLineOnOut(t *testing.T) {
	var out, prompt bytes.Buffer
	if err := run(strings.NewReader("fictional pass phrase\n"), &out, &prompt); err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSuffix(out.String(), "\n")
	hash, ok := strings.CutPrefix(line, "GYST_PASSWORD_HASH=")
	if !ok || strings.Contains(line, "\n") {
		t.Fatalf("out must be exactly one GYST_PASSWORD_HASH= line")
	}
	if cost, err := bcrypt.Cost([]byte(hash)); err != nil || cost != 12 {
		t.Fatalf("cost = %d, err %v; want 12", cost, err)
	}
	if err := (services.AuthConfig{PasswordHash: hash}).VerifyPassword("fictional pass phrase"); err != nil {
		t.Errorf("verify: %v", err)
	}
	if !strings.HasPrefix(prompt.String(), "Password: ") || strings.Contains(prompt.String(), "fictional") {
		t.Errorf("unexpected prompt output %q", prompt.String())
	}
}

func TestRunKeepsEdgeSpaces(t *testing.T) {
	var out, prompt bytes.Buffer
	if err := run(strings.NewReader(" edge spaces \r\n"), &out, &prompt); err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(strings.TrimPrefix(out.String(), "GYST_PASSWORD_HASH="))
	config := services.AuthConfig{PasswordHash: hash}
	if err := config.VerifyPassword(" edge spaces "); err != nil {
		t.Errorf("exact string rejected: %v", err)
	}
	if err := config.VerifyPassword("edge spaces"); err == nil {
		t.Error("trimmed string must not verify")
	}
}

func TestRunRejectsBadInput(t *testing.T) {
	for _, in := range []string{"\n", "  \t\n", "", strings.Repeat("a", 73) + "\n"} {
		var out, prompt bytes.Buffer
		if err := run(strings.NewReader(in), &out, &prompt); err == nil {
			t.Errorf("input %q: want error", in)
		}
		if out.Len() != 0 {
			t.Errorf("input %q: wrote to out", in)
		}
	}
}
