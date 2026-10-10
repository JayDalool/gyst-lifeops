package providers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"lifeops-tracker/internal/models"
)

var ErrNotConfigured = errors.New("provider credentials are not configured")

// ErrHostNotAllowed means a link does not point at the configured provider host,
// so no request (and no credential) was sent.
var ErrHostNotAllowed = errors.New("link host is not the configured Jira host")

type LinkSnapshot struct {
	Title      string
	StatusText string
}

type Provider interface {
	Fetch(context.Context, models.ExternalLink) (LinkSnapshot, error)
}

func Client() *http.Client { return &http.Client{Timeout: 8 * time.Second} }
