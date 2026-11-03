package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/tzdb"
)

func tzinfoAt(t *testing.T, value string) time.Time {
	t.Helper()
	instant, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return instant
}

func TestTzinfoRequested(t *testing.T) {
	if !tzinfoRequested([]string{"lifeops", "tzinfo"}) {
		t.Fatal("tzinfo should match")
	}
	for _, args := range [][]string{
		nil, {"lifeops"}, {"lifeops", "backup"}, {"lifeops", "healthcheck"}, {"lifeops", "x", "tzinfo"},
	} {
		if tzinfoRequested(args) {
			t.Fatalf("%v must not match", args)
		}
	}
}

// 00:30 on 2026-11-02 in Winnipeg (permanent UTC-05). Before #193 a host
// without zone files said 23:30 on the 1st.
func TestTzinfoReportsPinnedDatabaseAndLocalTime(t *testing.T) {
	var out bytes.Buffer
	if err := RunTzinfo(&out, []string{"-check"}, "", tzinfoAt(t, "2026-11-02T05:30:00Z")); err != nil {
		t.Fatalf("tzinfo -check: %v\n%s", err, out.String())
	}
	for _, want := range []string{
		"tzdb: " + tzdb.Version() + ", pinned in the binary (zoneinfo.zip sha256 " + tzdb.SHA256() + ")",
		"application timezone: America/Winnipeg (GYST_TIMEZONE unset, default America/Winnipeg)",
		"now: 2026-11-02T05:30:00Z UTC = 2026-11-02 00:30:00 local, UTC offset -05:00, local day 2026-11-02",
		"probe: America/Winnipeg, 14 known UTC offsets (history and permanent UTC-05): OK",
		"(not used by GYST)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestTzinfoCustomZoneAndHistory(t *testing.T) {
	var out bytes.Buffer
	if err := RunTzinfo(&out, []string{"-check"}, " America/Toronto ", tzinfoAt(t, "2026-01-15T12:00:00Z")); err != nil {
		t.Fatal(err)
	}
	if want := `application timezone: America/Toronto (GYST_TIMEZONE "America/Toronto")`; !strings.Contains(out.String(), want) {
		t.Fatalf("output lacks %q:\n%s", want, out.String())
	}
	if want := "2026-01-15 07:00:00 local, UTC offset -05:00, local day 2026-01-15"; !strings.Contains(out.String(), want) {
		t.Fatalf("output lacks %q:\n%s", want, out.String())
	}
}

// -check fails on an invalid GYST_TIMEZONE; without -check the report still
// prints and exits cleanly.
func TestTzinfoCheckFailsOnInvalidZone(t *testing.T) {
	now := tzinfoAt(t, "2026-11-02T05:30:00Z")
	var out bytes.Buffer
	if err := RunTzinfo(&out, []string{"-check"}, "Mars/Base", now); err == nil {
		t.Fatalf("tzinfo -check accepted Mars/Base:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "application timezone: INVALID") {
		t.Fatalf("no INVALID line:\n%s", out.String())
	}
	out.Reset()
	if err := RunTzinfo(&out, nil, "Mars/Base", now); err != nil {
		t.Fatalf("tzinfo without -check: %v", err)
	}
	// A typo such as "tzinfo check" must fail, not pass silently.
	for _, args := range [][]string{{"check"}, {"-check", "extra"}} {
		out.Reset()
		if err := RunTzinfo(&out, args, "Mars/Base", now); err == nil {
			t.Fatalf("tzinfo %v accepted a stray argument:\n%s", args, out.String())
		}
	}
}

// tzinfo is read-only and prints no secret: it is never given the
// environment, and the secrets set here never appear in its output.
func TestTzinfoPrintsNoSecrets(t *testing.T) {
	const sentinel = "tzinfo-secret-sentinel-7f3a"
	for _, key := range []string{"GYST_PASSWORD_HASH", "GYST_HEALTH_SYNC_TOKEN", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "VAPID_PRIVATE_KEY", "DB_PATH"} {
		t.Setenv(key, sentinel)
	}
	var out bytes.Buffer
	if err := RunTzinfo(&out, []string{"-check"}, "", tzinfoAt(t, "2026-11-02T05:30:00Z")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), sentinel) {
		t.Fatalf("tzinfo printed an environment secret:\n%s", out.String())
	}
}

func TestHostZoneinfoReport(t *testing.T) {
	dir := t.TempDir()
	if got := hostZoneinfo(filepath.Join(dir, "missing")); got != "absent" {
		t.Fatalf("missing dir = %q", got)
	}
	if got := hostZoneinfo(dir); got != "present, release unknown (no tzdata.zi)" {
		t.Fatalf("dir without tzdata.zi = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "tzdata.zi"), []byte("# version 2025b\nR d 1916 o - May 1 0 1 S\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := hostZoneinfo(dir); got != "present, tzdb 2025b" {
		t.Fatalf("dir with tzdata.zi = %q", got)
	}
}
