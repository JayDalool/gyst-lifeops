package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"lifeops-tracker/internal/services"
)

// Structured request logging and sanitised client errors (GYST-VN-015).
//
// One middleware gives every wrapped request a fresh id, returns it in the
// X-Request-Id header and the error text, and writes exactly one ERROR line
// for each response that is a 5xx or carries a recorded error. The line never
// holds a body, form value, query string, cookie, token or header value.
//
// serverError keeps its (w, err) signature so its call sites stay untouched:
// it finds the request id on the response writer and hands the error to the
// middleware's wrapper, which is the only place the detail is written.

const requestIDHeader = "X-Request-Id"

// maxLoggedError caps the error text in a log line.
const maxLoggedError = 300

type requestIDKey struct{}

// newRequestID returns 16 lowercase hex characters from crypto/rand. It never
// reads a client-supplied value.
func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail on supported platforms; a fixed id still
		// lets the request be served and logged.
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}

// requestIDFrom returns the id RequestLog attached to ctx, or "".
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// respRecorder wraps the ResponseWriter to learn the status and to receive the
// error behind a failure. Unwrap keeps http.ResponseController working.
type respRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	err         error
}

func (r *respRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *respRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.status = http.StatusOK
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(p)
}

func (r *respRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// noteError hands err to the middleware for the log line. It is a no-op when w
// is not the middleware's wrapper (a handler called directly in a test).
func noteError(w http.ResponseWriter, err error) {
	if rec, ok := w.(*respRecorder); ok && err != nil && rec.err == nil {
		rec.err = err
	}
}

// writeError answers with a fixed public message and the request reference,
// and records err for the log. No text from err reaches the response.
func writeError(w http.ResponseWriter, status int, publicMessage string, err error) {
	noteError(w, err)
	if id := w.Header().Get(requestIDHeader); id != "" {
		publicMessage += " Reference: " + id + "."
	}
	http.Error(w, publicMessage, status)
}

var (
	safeSegment = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,23}$`)
	allDigits   = regexp.MustCompile(`^[0-9]+$`)
)

// routeLabel reduces a URL path to a low-cardinality label with no ids or
// secrets: numeric segments become :id, the calendar feed token becomes
// :token and any segment that is not a short lowercase word becomes :param.
// The query string is never part of it.
func routeLabel(path string) string {
	var segments []string
	for _, seg := range strings.Split(path, "/") {
		if seg != "" {
			segments = append(segments, seg)
		}
	}
	if len(segments) == 0 {
		return "/"
	}
	for i, seg := range segments {
		switch {
		case i == 2 && segments[0] == "calendar" && segments[1] == "feed":
			segments[i] = ":token"
		case allDigits.MatchString(seg):
			segments[i] = ":id"
		case safeSegment.MatchString(seg):
		default:
			segments[i] = ":param"
		}
	}
	return "/" + strings.Join(segments, "/")
}

// cappedError renders err for a log line: one value, bounded in length.
func cappedError(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	if len(text) > maxLoggedError {
		text = text[:maxLoggedError]
		for !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
	}
	return text
}

// RequestLog wraps next with a request id and the one-line failure log. A nil
// logger means slog.Default().
func RequestLog(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set(requestIDHeader, id)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id))
		rec := &respRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.status < 500 && rec.err == nil {
			return
		}
		attrs := []slog.Attr{
			slog.String("request_id", id),
			slog.String("method", r.Method),
			slog.String("route", routeLabel(r.URL.Path)),
			slog.Int("status", rec.status),
		}
		if rec.err != nil {
			attrs = append(attrs, slog.String("error", cappedError(rec.err)))
		}
		logger.LogAttrs(r.Context(), slog.LevelError, "request failed", attrs...)
	})
}

// log returns the handler's logger.
func (h *Handler) log() *slog.Logger {
	if h.Logger != nil {
		return h.Logger
	}
	return slog.Default()
}

// authEvent writes one audit line for a sign-in, failed sign-in or sign-out.
// It carries the event name, the ingress bucket, the truncated client label
// and the request id; never a password, cookie, token or header.
func (h *Handler) authEvent(r *http.Request, event, ingress, client string, level slog.Level) {
	h.log().LogAttrs(r.Context(), level, "auth",
		slog.String("event", event),
		slog.String("ingress", ingress),
		slog.String("client", client),
		slog.String("request_id", requestIDFrom(r.Context())),
	)
}

// userError answers with err's own text when it is a services.UserError
// (fixed, browser-safe text) and otherwise with a generic message plus the
// request reference; the detail goes only to the log.
func userError(w http.ResponseWriter, err error, status int, generic string) {
	if message, ok := services.UserMessage(err); ok {
		http.Error(w, message, status)
		return
	}
	writeError(w, status, generic, err)
}
