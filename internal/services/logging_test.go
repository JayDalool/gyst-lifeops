package services

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

// Provider-failure and session-revocation behaviour for GYST-VN-015. Values
// are fictional; the push endpoint is a documentation host.

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return buf
}

func TestPushFailureLogsSubscriptionIDNotEndpoint(t *testing.T) {
	buf := captureLog(t)
	s := testService(t)
	endpoint := "https://push.example.test/capability-fictional-123"
	if err := s.Repo.UpsertPushSubscription(models.PushSubscription{Endpoint: endpoint, P256DH: "x", Auth: "y"}); err != nil {
		t.Fatal(err)
	}
	s.sendPrayerPush(context.Background(), PushConfig{PublicKey: "x", PrivateKey: "y", Subscriber: "mailto:a@example.test"}, "2026-10-09", "Fajr")

	logged := buf.String()
	if !strings.Contains(logged, `"provider":"webpush"`) || !strings.Contains(logged, `"subscription_id":1`) {
		t.Fatalf("no provider failure line: %q", logged)
	}
	if strings.Contains(logged, "capability-fictional-123") || strings.Contains(logged, "push.example.test") {
		t.Fatalf("endpoint URL reached the log: %s", logged)
	}
}

func TestTrainingFailureLogsCuratedProviderLine(t *testing.T) {
	buf := captureLog(t)
	s := testService(t)
	s.Training = &TrainingSync{Provider: &trainingFixture{fail: true}}
	err := s.SyncTraining(context.Background())
	if message, ok := UserMessage(err); !ok || message != "Unavailable" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(buf.String(), `"provider":"hevy"`) {
		t.Fatalf("no provider line: %s", buf.String())
	}
}

func TestUserMessageOnlyForUserErrors(t *testing.T) {
	if _, ok := UserMessage(errors.New("sql: no rows in result set")); ok {
		t.Fatal("a plain error is not browser-safe")
	}
	if message, ok := UserMessage(errors.Join(errors.New("ctx"), userError("fixed"))); !ok || message != "fixed" {
		t.Fatalf("wrapped UserError lost: %q %v", message, ok)
	}
}

func TestRevokeSessionReportsOnlyRealRevocations(t *testing.T) {
	s := testService(t)
	if revoked, err := s.RevokeSession("never-issued-fictional"); err != nil || revoked {
		t.Fatalf("unknown token: revoked=%v err=%v", revoked, err)
	}
	if revoked, err := s.RevokeSession(""); err != nil || revoked {
		t.Fatalf("empty token: revoked=%v err=%v", revoked, err)
	}
}
