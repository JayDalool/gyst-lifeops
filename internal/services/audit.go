package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
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
	AuditActionLoginFailed      = "login.failed"
	AuditActionLoginSucceeded   = "login.succeeded"
	AuditActionLogout           = "logout"
	AuditActionSessionRevokeAll = "session.revoked_all"
	AuditActionReauthSucceeded  = "reauth.succeeded"
	AuditActionReauthFailed     = "reauth.failed"
	AuditActionAccessDenied     = "access.denied"

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
}

var (
	auditActorTypes = map[string]bool{AuditActorHuman: true, AuditActorMachine: true, AuditActorSystem: true}
	auditIngresses  = map[string]bool{AuditIngressPublic: true, AuditIngressTailnet: true, AuditIngressInternal: true, AuditIngressJob: true}
	auditOutcomes   = map[string]bool{AuditOutcomeSuccess: true, AuditOutcomeDenied: true, AuditOutcomeFailed: true}

	auditRefPattern     = regexp.MustCompile(`^[A-Za-z0-9:._/#@+-]*$`)
	auditRequestPattern = regexp.MustCompile(`^[a-f0-9]*$`)
)

// Detail containing any of these (case-insensitive) is replaced entirely.
var auditSecretMarkers = []string{
	"password", "passwd", "secret", "token", "bearer", "authorization", "cookie",
	"gyst_session", "api_key", "apikey", "$2a$", "$2b$", "$2y$",
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

// sanitizeAuditDetail drops control characters, redacts anything secret
// looking, then cuts to 200 runes (never mid-character).
func sanitizeAuditDetail(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	lower := strings.ToLower(s)
	for _, m := range auditSecretMarkers {
		if strings.Contains(lower, m) {
			return auditRedacted
		}
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
