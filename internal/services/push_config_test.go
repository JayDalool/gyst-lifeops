package services

import (
	"context"
	"testing"
	"time"

	"lifeops-tracker/internal/config"
)

func TestPushPublicKeyIsTheInjectedKey(t *testing.T) {
	s := testService(t)
	if got := s.PushPublicKey(); got != "" {
		t.Fatalf("unconfigured key = %q", got)
	}
	s.Push = config.Push{PublicKey: "pub-fictional", PrivateKey: "priv-fictional", Subscriber: "mailto:a@example.test"}
	if got := s.PushPublicKey(); got != "pub-fictional" {
		t.Fatalf("key = %q", got)
	}
}

// With no VAPID configuration the scheduler returns at once instead of
// looping, so it never touches the clock or the subscriptions.
func TestPushSchedulerDoesNotStartWhenUnconfigured(t *testing.T) {
	s := testService(t)
	done := make(chan struct{})
	go func() { s.RunPushScheduler(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler is running without VAPID configuration")
	}
}
