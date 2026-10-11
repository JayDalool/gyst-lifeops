package services

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/repository"
)

// GYST-VN-077 / 10A D-S5: 30-day rolling expiry, 90-day absolute cap. All
// dates are fictional and every instant comes from the injected clock.

const day = 24 * time.Hour

var capBase = time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)

// capFixture returns a service pinned at capBase plus its raw database for
// rows the service never writes (legacy and corrupt created_at values).
func capFixture(t *testing.T) (*Service, *sql.DB, *time.Time) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	svc := New(repository.New(database))
	now := capBase
	svc.Now = func() time.Time { return now }
	return svc, database, &now
}

func rowCount(t *testing.T, svc *Service) int {
	t.Helper()
	n, err := svc.Repo.CountSessions()
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return n
}

// keepAlive uses the session every 25 days up to +75d so only the cap can end it.
func keepAlive(t *testing.T, svc *Service, now *time.Time, token string) {
	t.Helper()
	for _, d := range []time.Duration{25, 50, 75} {
		*now = capBase.Add(d * day)
		if _, err := svc.CheckSession(token); err != nil {
			t.Fatalf("+%dd: %v", d, err)
		}
	}
}

func TestCreateSessionWritesCreatedAtFromServiceClock(t *testing.T) {
	svc, _, _ := capFixture(t)
	token, expires, err := svc.CreateSession()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	stored, err := svc.Repo.GetSession(hashSessionToken(token))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !stored.CreatedAt.Equal(capBase) {
		t.Errorf("created_at = %v, want %v (service clock, not wall clock)", stored.CreatedAt, capBase)
	}
	if !expires.Equal(capBase.Add(sessionLifetime)) {
		t.Errorf("expiry = %v, want base+30d", expires)
	}
}

func TestIdleSessionLapsesAtThirtyDays(t *testing.T) {
	svc, _, now := capFixture(t)
	kept, _, _ := svc.CreateSession()
	lapsed, _, _ := svc.CreateSession()

	*now = capBase.Add(29 * day)
	if err := svc.ValidateSession(kept); err != nil {
		t.Fatalf("+29d: %v", err)
	}
	*now = capBase.Add(31 * day)
	if err := svc.ValidateSession(lapsed); !errors.Is(err, ErrNoSession) {
		t.Fatalf("+31d unused: got %v, want ErrNoSession", err)
	}
	if _, err := svc.Repo.GetSession(hashSessionToken(lapsed)); err == nil {
		t.Error("lapsed idle row not deleted")
	}
}

func TestActiveSessionRollsButIsCappedAtNinetyDays(t *testing.T) {
	svc, _, now := capFixture(t)
	token, _, _ := svc.CreateSession()
	hash := hashSessionToken(token)

	for _, d := range []time.Duration{20, 40, 60, 80} {
		*now = capBase.Add(d * day)
		renewed, err := svc.CheckSession(token)
		if err != nil {
			t.Fatalf("+%dd: %v", d, err)
		}
		want := now.Add(sessionLifetime)
		if limit := capBase.Add(90 * day); want.After(limit) {
			want = limit
		}
		if !renewed.Equal(want) {
			t.Errorf("+%dd renewed = %v, want %v", d, renewed, want)
		}
	}
	stored, _ := svc.Repo.GetSession(hash)
	if !stored.ExpiresAt.Equal(capBase.Add(90 * day)) {
		t.Errorf("after +80d expires_at = %v, want base+90d", stored.ExpiresAt)
	}

	*now = capBase.Add(89 * day)
	renewed, err := svc.CheckSession(token)
	if err != nil {
		t.Fatalf("+89d: %v", err)
	}
	if !renewed.IsZero() && renewed.After(capBase.Add(90*day)) {
		t.Errorf("+89d renewed = %v, past the cap", renewed)
	}
	stored, _ = svc.Repo.GetSession(hash)
	if !stored.ExpiresAt.Equal(capBase.Add(90 * day)) {
		t.Errorf("+89d expires_at = %v, want base+90d", stored.ExpiresAt)
	}

	// Inside the 30-day idle window but past the cap: refused and deleted.
	*now = capBase.Add(91 * day)
	if err := svc.ValidateSession(token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("+91d: got %v, want ErrNoSession", err)
	}
	if n := rowCount(t, svc); n != 0 {
		t.Errorf("capped row remained: %d", n)
	}
}

func TestRenewalIsClampedAtTheCap(t *testing.T) {
	svc, _, now := capFixture(t)
	token, _, _ := svc.CreateSession()
	hash := hashSessionToken(token)
	keepAlive(t, svc, now, token)
	*now = capBase.Add(89 * day)
	// Last seen 29 days ago, so a refresh is due; the idle window alone would
	// extend it to +119d.
	if err := svc.Repo.TouchSession(hash, capBase.Add(89*day+12*time.Hour), capBase.Add(60*day)); err != nil {
		t.Fatalf("touch: %v", err)
	}
	renewed, err := svc.CheckSession(token)
	if err != nil {
		t.Fatalf("+89d: %v", err)
	}
	if want := capBase.Add(90 * day); !renewed.Equal(want) {
		t.Errorf("renewedUntil = %v, want exactly base+90d (not +119d)", renewed)
	}
}

func TestCapBoundaryExactAndPlusMinusOneSecond(t *testing.T) {
	cases := []struct {
		name string
		at   time.Duration
		ok   bool
	}{
		{"cap-1s", 90*day - time.Second, true},
		{"cap exactly", 90 * day, false},
		{"cap+1s", 90*day + time.Second, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, now := capFixture(t)
			token, _, _ := svc.CreateSession()
			// Keep the session actively used so only the cap can refuse it.
			for d := 25 * day; d < tc.at; d += 25 * day {
				*now = capBase.Add(d)
				if _, err := svc.CheckSession(token); err != nil {
					t.Fatalf("visit at %v: %v", d, err)
				}
			}
			*now = capBase.Add(tc.at)
			err := svc.ValidateSession(token)
			if tc.ok && err != nil {
				t.Errorf("refused inside the cap: %v", err)
			}
			if !tc.ok && !errors.Is(err, ErrNoSession) {
				t.Errorf("accepted at the cap: %v", err)
			}
		})
	}
}

// Durations are absolute, so a zone change between creation and use (clock
// reported in a DST-observing zone) must not move the cap by an hour.
func TestCapIsAbsoluteAcrossDST(t *testing.T) {
	toronto, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Skipf("tz data unavailable: %v", err)
	}
	// Created just before spring-forward (2026-03-08 02:00 local).
	start := time.Date(2026, 3, 7, 12, 0, 0, 0, toronto)
	for _, tc := range []struct {
		name string
		at   time.Duration
		ok   bool
	}{{"cap-1s", 90*day - time.Second, true}, {"cap", 90 * day, false}} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, now := capFixture(t)
			*now = start
			token, _, _ := svc.CreateSession()
			for d := 25 * day; d < tc.at; d += 25 * day {
				*now = start.Add(d)
				if _, err := svc.CheckSession(token); err != nil {
					t.Fatalf("visit: %v", err)
				}
			}
			*now = start.Add(tc.at) // wall clock now differs by an hour
			err := svc.ValidateSession(token)
			if tc.ok != (err == nil) {
				t.Errorf("ok=%v err=%v", tc.ok, err)
			}
		})
	}
}

func TestLegacyDefaultFormatRowIsCappedFromRealCreation(t *testing.T) {
	svc, database, now := capFixture(t)
	// Written as the old schema DEFAULT did: UTC, 'YYYY-MM-DD HH:MM:SS'.
	insert := func(name string) string {
		hash := hashSessionToken(name)
		_, err := database.Exec(
			`INSERT INTO sessions (token_hash, created_at, expires_at, last_seen_at) VALUES (?, ?, ?, ?)`,
			hash, "2026-01-05 12:00:00",
			// expires_at far past the cap: must never be honoured.
			capBase.Add(200*day).Format("2006-01-02T15:04:05.000000000Z"),
			capBase.Add(60*day).Format("2006-01-02T15:04:05.000000000Z"))
		if err != nil {
			t.Fatalf("insert legacy: %v", err)
		}
		return name
	}
	inside, past := insert("legacy-inside"), insert("legacy-past")

	*now = capBase.Add(89 * day)
	renewed, err := svc.CheckSession(inside)
	if err != nil {
		t.Fatalf("inside window: %v", err)
	}
	if want := capBase.Add(90 * day); !renewed.Equal(want) {
		t.Errorf("legacy refresh = %v, want clamp to %v", renewed, want)
	}

	*now = capBase.Add(90*day + time.Second)
	if err := svc.ValidateSession(past); !errors.Is(err, ErrNoSession) {
		t.Errorf("legacy past cap: got %v, want ErrNoSession", err)
	}
}

func TestUnparseableCreatedAtFailsClosed(t *testing.T) {
	svc, database, _ := capFixture(t)
	for i, bad := range []string{"not a time", "", "2026-01-05", "0000-00-00 00:00:00"} {
		name := "bad-" + string(rune('a'+i))
		_, err := database.Exec(
			`INSERT INTO sessions (token_hash, created_at, expires_at, last_seen_at) VALUES (?, ?, ?, ?)`,
			hashSessionToken(name), bad,
			capBase.Add(10*day).Format("2006-01-02T15:04:05.000000000Z"),
			capBase.Format("2006-01-02T15:04:05.000000000Z"))
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
		if err := svc.ValidateSession(name); err == nil {
			t.Errorf("created_at %q accepted; must fail closed", bad)
		}
	}
}

func TestSweepRemovesAClampedRow(t *testing.T) {
	svc, _, now := capFixture(t)
	token, _, _ := svc.CreateSession()
	keepAlive(t, svc, now, token)
	*now = capBase.Add(89 * day)
	if _, err := svc.CheckSession(token); err != nil {
		t.Fatalf("+89d: %v", err)
	}
	*now = capBase.Add(91 * day) // row's expires_at is the cap, not +119d
	n, err := svc.SweepExpiredSessions()
	if err != nil || n != 1 {
		t.Fatalf("sweep = %d, %v; want 1 removed", n, err)
	}
}
