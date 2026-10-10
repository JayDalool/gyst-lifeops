package handlers

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"lifeops-tracker/internal/authlimit"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// Audit rows for sign-in and sign-out (GYST-VN-007). Every row is written
// after the decision it records, and no response depends on the write: a
// failure is logged and the login or logout carries on unchanged.

// Fixed detail values. detail is a closed set, never error text, a URL, a
// header or anything the client sent.
const (
	auditAccountTarget  = "account:owner" // GYST has one account and no username
	auditDetailThrottle = "throttled"
	auditDetailWrongPW  = "wrong-credentials" // not "wrong-password": the sanitiser redacts the word password

	throttleAuditMaxKeys = 1000
)

// throttleAuditGate lets a throttled refusal into the audit table at most
// once per key per authlimit.DefaultPolicy.KeyBlock. A refusal costs an
// attacker nothing (no bcrypt), so without this the table could be flooded.
// It holds at most max keys and has no goroutine: expired keys are dropped
// only when the map is full.
type throttleAuditGate struct {
	mu    sync.Mutex
	until map[string]time.Time
	max   int
}

func newThrottleAuditGate(max int) *throttleAuditGate {
	return &throttleAuditGate{until: make(map[string]time.Time), max: max}
}

// allow reports whether a row may be written for key at now. It is true when
// the key is new or now >= its recorded time (half-open, like authlimit),
// and the key is then held for one KeyBlock. When the map is full it drops
// expired keys; if it is still full the row is skipped. A nil gate skips.
func (g *throttleAuditGate) allow(key string, now time.Time) bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	until, known := g.until[key]
	if known && now.Before(until) {
		return false
	}
	if !known && len(g.until) >= g.max {
		for k, u := range g.until {
			if !now.Before(u) {
				delete(g.until, k)
			}
		}
		if len(g.until) >= g.max {
			return false
		}
	}
	g.until[key] = now.Add(authlimit.DefaultPolicy.KeyBlock)
	return true
}

// audit writes one row for the account, filling the ingress, request id and
// the fixed target. A failure logs one ERROR with the action and request id
// only; the error text is withheld.
func (h *Handler) audit(r *http.Request, e repository.AuditEvent, ingress string) {
	e.ActorType = services.AuditActorHuman
	e.Ingress = ingress
	e.RequestID = requestIDFrom(r.Context())
	e.TargetRef = auditAccountTarget
	if err := h.Service.RecordAudit(r.Context(), e); err != nil {
		h.log().LogAttrs(r.Context(), slog.LevelError, "audit write failed",
			slog.String("action", e.Action),
			slog.String("request_id", e.RequestID),
		)
	}
}

// clientRef is the actor_ref for a request that has no session: the same
// truncated /24 or /48 label the auth log lines use, never the full address.
func clientRef(label string) string { return "client:" + label }
