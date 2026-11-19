package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunConfigRejectsStrayArguments(t *testing.T) {
	var out bytes.Buffer
	err := RunConfig([]string{"db", "data/x.db"}, testEnv(), noFile, &out)
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("err = %v, want unexpected arguments", err)
	}
	if strings.Contains(out.String(), "config: OK") {
		t.Fatal("a mistyped preflight must not print OK")
	}
	if err := RunConfig([]string{"-seed=false"}, testEnv(), noFile, &out); err != nil {
		t.Fatalf("flags only must still pass: %v", err)
	}
}

func TestPasswordHashFlagRequested(t *testing.T) {
	cases := map[string]bool{
		"-passwordhash":             true,
		"--passwordhash":            true,
		"-passwordhash=true":        true,
		"-passwordhash=false":       false,
		"-seed=false -passwordhash": true,
		"-db -passwordhash":         false,
		"-db x.db -passwordhash":    true,
		"-addr :1":                  false,
		"extra -passwordhash":       false,
		"-- -passwordhash":          false,
		"":                          false,
	}
	for args, want := range cases {
		if got := passwordHashFlagRequested(strings.Fields(args)); got != want {
			t.Errorf("passwordHashFlagRequested(%q) = %v, want %v", args, got, want)
		}
	}
}
