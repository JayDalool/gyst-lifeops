package app

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"lifeops-tracker/internal/services"
)

func hashFromOutput(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if hash, ok := strings.CutPrefix(line, "GYST_PASSWORD_HASH="); ok {
			return hash
		}
	}
	t.Fatalf("no GYST_PASSWORD_HASH= line in output")
	return ""
}

func TestWritePasswordHashCost12AndVerifies(t *testing.T) {
	var out bytes.Buffer
	if err := writePasswordHash(strings.NewReader("fictional pass phrase\r\n"), &out); err != nil {
		t.Fatal(err)
	}
	hash := hashFromOutput(t, out.String())
	if cost, err := bcrypt.Cost([]byte(hash)); err != nil || cost != 12 {
		t.Fatalf("cost = %d, err %v; want 12", cost, err)
	}
	if err := (services.AuthConfig{PasswordHash: hash}).VerifyPassword("fictional pass phrase"); err != nil {
		t.Errorf("verify: %v", err)
	}
	if strings.Contains(out.String(), "fictional pass phrase") {
		t.Error("password was written to output")
	}
	for _, want := range []string{"Enter password: ", "\nPassword hash:\n", "\nAdd this to your "} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output format changed: missing %q", want)
		}
	}
}

func TestWritePasswordHashKeepsEdgeSpaces(t *testing.T) {
	var out bytes.Buffer
	if err := writePasswordHash(strings.NewReader(" edge spaces \n"), &out); err != nil {
		t.Fatal(err)
	}
	config := services.AuthConfig{PasswordHash: hashFromOutput(t, out.String())}
	if err := config.VerifyPassword(" edge spaces "); err != nil {
		t.Errorf("exact string rejected: %v", err)
	}
	if err := config.VerifyPassword("edge spaces"); err == nil {
		t.Error("trimmed string must not verify")
	}
}

func TestWritePasswordHashRejectsBadInput(t *testing.T) {
	for _, in := range []string{"\n", "   \n", "", strings.Repeat("a", 73) + "\n"} {
		var out bytes.Buffer
		if err := writePasswordHash(strings.NewReader(in), &out); err == nil {
			t.Errorf("input %q: want error", in)
		}
		if strings.Contains(out.String(), "GYST_PASSWORD_HASH=") {
			t.Errorf("input %q: a hash was printed", in)
		}
	}
}
