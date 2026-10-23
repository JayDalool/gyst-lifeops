package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"lifeops-tracker/internal/config"
)

// PushConfig is the VAPID trio, injected at construction (s.Push).
type PushConfig = config.Push

func (s *Service) PushPublicKey() string { return s.Push.PublicKey }

func (s *Service) RunPushScheduler(ctx context.Context) {
	config := s.Push
	if !config.Configured() {
		return
	}
	s.sendDuePrayerPushes(ctx, config, s.Now())
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.sendDuePrayerPushes(ctx, config, now)
		}
	}
}

func (s *Service) sendDuePrayerPushes(ctx context.Context, config PushConfig, now time.Time) {
	schedule, err := s.PrayerTimes(now)
	if err != nil {
		return
	}
	for _, prayer := range schedule.Times {
		if now.Before(prayer.At) || now.Sub(prayer.At) >= 2*time.Minute {
			continue
		}
		s.sendPrayerPush(ctx, config, schedule.Date, prayer.Name)
	}
}

func (s *Service) sendPrayerPush(ctx context.Context, config PushConfig, date, prayer string) {
	subscriptions, err := s.Repo.PushSubscriptions()
	if err != nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"title": prayer + " prayer", "body": "It is time for " + prayer + ".", "url": "/today", "tag": "prayer-" + date + "-" + strings.ToLower(prayer)})
	for _, subscription := range subscriptions {
		claimed, err := s.Repo.ClaimPushDelivery(subscription.ID, date, prayer)
		if err != nil || !claimed {
			continue
		}
		response, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{Endpoint: subscription.Endpoint, Keys: webpush.Keys{P256dh: subscription.P256DH, Auth: subscription.Auth}}, &webpush.Options{Subscriber: config.Subscriber, VAPIDPublicKey: config.PublicKey, VAPIDPrivateKey: config.PrivateKey, TTL: 120})
		if err != nil {
			// The error text can carry the subscription endpoint URL, which is
			// a capability URL: log the subscription id only.
			slog.Warn("provider failure", "event", "provider.failed", "provider", "webpush", "subscription_id", subscription.ID, "reason", "send failed")
			_ = s.Repo.ReleasePushDelivery(subscription.ID, date, prayer)
			continue
		}
		response.Body.Close()
		if response.StatusCode == http.StatusGone || response.StatusCode == http.StatusNotFound {
			_ = s.Repo.DeletePushSubscription(subscription.Endpoint)
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			slog.Warn("provider failure", "event", "provider.failed", "provider", "webpush", "subscription_id", subscription.ID, "status", response.StatusCode)
			_ = s.Repo.ReleasePushDelivery(subscription.ID, date, prayer)
		}
	}
}
