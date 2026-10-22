package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/repository"
)

// Audit events (GYST-VN-005). RecordAudit is the only writer. It checks
// every field against fixed lists, stamps `at` itself and strips anything
// that looks like a secret from the free-text detail.

// ErrAuditInvalid means an event failed validation. Nothing was written.
var ErrAuditInvalid = errors.New("invalid audit event")

const (
	AuditActionLoginFailed       = "login.failed"
	AuditActionLoginSucceeded    = "login.succeeded"
	AuditActionLogout            = "logout"
	AuditActionSessionRevokeAll  = "session.revoked_all"
	AuditActionReauthSucceeded   = "reauth.succeeded"
	AuditActionReauthFailed      = "reauth.failed"
	AuditActionAccessDenied      = "access.denied"
	AuditActionHealthSyncRefused = "health.sync_refused"

	AuditActorHuman   = "human"
	AuditActorMachine = "machine"
	AuditActorSystem  = "system"

	AuditIngressPublic   = "public"
	AuditIngressTailnet  = "tailnet"
	AuditIngressInternal = "internal"
	AuditIngressJob      = "job"

	AuditOutcomeSuccess = "success"
	AuditOutcomeDenied  = "denied"
	AuditOutcomeFailed  = "failed"

	auditDetailMax   = 200
	auditRedacted    = "redacted"
	auditActorRefMax = 120
	auditTargetMax   = 200
	auditRequestMax  = 32
)

// The action list lives in Go, not in a SQL CHECK, so later tickets can add
// actions without rebuilding the table. agent.*, github.* and server.*
// actions are owned by Command Center and stay out; automation.* is not
// added yet.
var auditActions = map[string]bool{
	AuditActionLoginFailed: true, AuditActionLoginSucceeded: true, AuditActionLogout: true,
	AuditActionSessionRevokeAll: true, AuditActionReauthSucceeded: true,
	AuditActionReauthFailed: true, AuditActionAccessDenied: true,
	AuditActionHealthSyncRefused: true,
}

var (
	auditActorTypes = map[string]bool{AuditActorHuman: true, AuditActorMachine: true, AuditActorSystem: true}
	auditIngresses  = map[string]bool{AuditIngressPublic: true, AuditIngressTailnet: true, AuditIngressInternal: true, AuditIngressJob: true}
	auditOutcomes   = map[string]bool{AuditOutcomeSuccess: true, AuditOutcomeDenied: true, AuditOutcomeFailed: true}

	auditRefPattern     = regexp.MustCompile(`^[A-Za-z0-9:._/#@+-]*$`)
	auditRequestPattern = regexp.MustCompile(`^[a-f0-9]*$`)
)

// Detail containing any of these (case-insensitive, after normalisation) is
// replaced entirely.
var auditSecretMarkers = []string{
	"password", "passwd", "pwd", "secret", "token", "bearer", "basic", "authorization",
	"cookie", "set-cookie", "session", "gyst_session", "api_key", "api-key", "apikey",
	"x-api-key", "private_key", "private-key", "$2a$", "$2b$", "$2y$",
}

// Value shapes that look like a credential whatever the words around them.
var auditSecretShapes = regexp.MustCompile(
	`[A-Za-z0-9+/=_.-]{32,}` + // long opaque run: base64, base64url, hex, hashes
		`|(?i:eyj[a-z0-9_-]*\.[a-z0-9_-]*\.)` + // JWT (matched on lower-cased text, so case-insensitive)
		`|(?i:(?:^|[^a-z0-9])(?:sk-|ghp_|gho_|github_pat_|xox[abp]-))` + // provider prefixes (sk-ant- starts with sk-)
		`|\$2[aby]\$` + // bcrypt
		`|[0-9a-fA-F]{40,}`)

// stripAuditRunes removes control, format (Cf: zero-width, bidi overrides)
// and other non-graphic runes; any space separator becomes a plain space.
func stripAuditRunes(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ':
			return r
		case unicode.Is(unicode.Zs, r):
			return ' '
		case unicode.Is(unicode.Cf, r), !unicode.IsGraphic(r):
			return -1
		}
		return r
	}, s)
}

// foldFullwidth maps U+FF01-U+FF5E to ASCII.
func foldFullwidth(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0xFF01 && r <= 0xFF5E {
			return r - 0xFEE0
		}
		return r
	}, s)
}

// auditPercentEscape is one %XX escape. Decoding is lenient: a stray or
// invalid % sequence elsewhere in the text must not stop valid ones decoding.
var auditPercentEscape = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

func decodePercentLenient(s string) string {
	return auditPercentEscape.ReplaceAllStringFunc(s, func(m string) string {
		b, err := strconv.ParseUint(m[1:], 16, 8)
		if err != nil {
			return m
		}
		return string([]byte{byte(b)})
	})
}

// normalizeForAudit is the form detail is matched in: invisible runes gone,
// percent-decoded once, fullwidth folded, lower-cased.
func normalizeForAudit(s string) string {
	s = stripAuditRunes(s)
	s = stripAuditRunes(strings.ToValidUTF8(decodePercentLenient(s), ""))
	return strings.ToLower(foldFullwidth(s))
}

func validateAudit(e repository.AuditEvent) error {
	switch {
	case !auditActions[e.Action],
		!auditActorTypes[e.ActorType],
		!auditIngresses[e.Ingress],
		!auditOutcomes[e.Outcome],
		len(e.ActorRef) > auditActorRefMax || !auditRefPattern.MatchString(e.ActorRef),
		len(e.TargetRef) > auditTargetMax || !auditRefPattern.MatchString(e.TargetRef),
		len(e.RequestID) > auditRequestMax || !auditRequestPattern.MatchString(e.RequestID):
		return ErrAuditInvalid
	}
	return nil
}

// sanitizeAuditDetail removes invisible and control characters, redacts
// anything secret looking (by marker word or by value shape, matched on the
// normalised form), then cuts to 200 runes (never mid-character).
func sanitizeAuditDetail(s string) string {
	s = stripAuditRunes(strings.ToValidUTF8(s, ""))
	norm := normalizeForAudit(s)
	for _, m := range auditSecretMarkers {
		if strings.Contains(norm, m) {
			return auditRedacted
		}
	}
	if auditSecretShapes.MatchString(norm) {
		return auditRedacted
	}
	if utf8.RuneCountInString(s) > auditDetailMax {
		s = string([]rune(s)[:auditDetailMax])
	}
	return s
}

// SessionActorRef names a session in the audit log without exposing it:
// "session:" plus the first 8 hex characters of the stored token hash.
func SessionActorRef(rawToken string) string {
	if rawToken == "" {
		return "session:none"
	}
	return "session:" + hashSessionToken(rawToken)[:8]
}

// RecordAudit validates e and writes one row. The caller cannot set At.
// The insert ignores cancellation of ctx so a client that disconnects
// cannot drop a security record. Callers log a failure and carry on: an
// audit failure must never change an auth decision.
func (s *Service) RecordAudit(ctx context.Context, e repository.AuditEvent) error {
	if err := validateAudit(e); err != nil {
		return err
	}
	e.ID = 0
	e.Detail = sanitizeAuditDetail(e.Detail)
	e.At = clock.FormatInstant(s.Now())
	return s.Repo.InsertAuditEvent(context.WithoutCancel(ctx), e)
}

// Retention (GYST-VN-084). Security rows (everything not listed in
// lifecycleActions) are kept 365 days, lifecycle rows 180 days. Both cutoffs
// are half-open: a row exactly at the cutoff is kept.
const (
	securityRetention  = 365 * 24 * time.Hour
	lifecycleRetention = 180 * 24 * time.Hour
)

// lifecycleActions are the audit actions kept for the shorter period. It is
// empty today: agent.* and automation.run are not GYST actions. Add a name
// here when a lifecycle action joins auditActions. Tests replace it.
var lifecycleActions = map[string]bool{}

// auditPruneInterval is how often RunAuditPruneLoop runs after the first pass.
// A variable only so tests can shorten it.
var auditPruneInterval = 24 * time.Hour

func lifecycleActionList() []string {
	out := make([]string, 0, len(lifecycleActions))
	for a := range lifecycleActions {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// ErrAuditPruneSkipped means the clock-jump guard refused to delete anything.
var ErrAuditPruneSkipped = errors.New("audit prune skipped")

// PruneAuditEvents deletes expired audit rows and returns how many security
// and lifecycle rows went. Each statement is bounded by `at` and, for the
// lifecycle pass, by an action list; an empty list skips that DELETE.
func (s *Service) PruneAuditEvents(ctx context.Context) (security, lifecycle int64, err error) {
	now := s.Now()
	lifecycleList := lifecycleActionList()
	securityCutoff := clock.FormatInstant(now.Add(-securityRetention))
	lifecycleCutoff := clock.FormatInstant(now.Add(-lifecycleRetention))
	// Clock-jump guard: the shortest retention in use decides. If its cutoff is
	// later than the newest row, this run would delete every row of that kind
	// (a clock far in the future), so skip the WHOLE run before any DELETE.
	// With no lifecycle actions the security retention is the shortest. An
	// empty table is a no-op.
	guardCutoff := securityCutoff
	if len(lifecycleList) > 0 {
		guardCutoff = lifecycleCutoff
	}
	latest, err := s.Repo.LatestAuditEventAt(ctx)
	if err != nil {
		return 0, 0, err
	}
	if latest == "" {
		return 0, 0, nil
	}
	if guardCutoff > latest {
		return 0, 0, fmt.Errorf("%w: cutoff %s is later than the newest row; clock may be wrong", ErrAuditPruneSkipped, guardCutoff)
	}
	security, err = s.Repo.DeleteAuditEventsBefore(ctx, securityCutoff, lifecycleList, false)
	if err != nil {
		return 0, 0, err
	}
	if len(lifecycleList) == 0 {
		return security, 0, nil
	}
	lifecycle, err = s.Repo.DeleteAuditEventsBefore(ctx, lifecycleCutoff, lifecycleList, true)
	return security, lifecycle, err
}

// pruneAuditAndLog logs counts only, never row content. After shutdown
// begins the database may be closed, so it logs nothing.
func (s *Service) pruneAuditAndLog(ctx context.Context) {
	sec, life, err := s.PruneAuditEvents(ctx)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		slog.Error("audit prune failed", "event", "audit.prune_failed", "error", err.Error())
		return
	}
	slog.Info("audit prune", "event", "audit.pruned", "deleted_security", sec, "deleted_lifecycle", life)
}

// PruneAuditAtStart is the first pass. The app calls it synchronously before
// starting the loop, like SweepSessionsAtStart, so it cannot race Close.
func (s *Service) PruneAuditAtStart(ctx context.Context) {
	s.pruneAuditAndLog(ctx)
}

// RunAuditPruneLoop prunes every auditPruneInterval until ctx is cancelled.
// A failed run is retried on the next tick.
func (s *Service) RunAuditPruneLoop(ctx context.Context) {
	ticker := time.NewTicker(auditPruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pruneAuditAndLog(ctx)
		}
	}
}
