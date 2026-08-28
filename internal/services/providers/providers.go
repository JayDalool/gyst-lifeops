package providers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"lifeops-tracker/internal/models"
)

var ErrNotConfigured = errors.New("provider credentials are not configured")

type LinkSnapshot struct {
	Title      string
	StatusText string
}

type Provider interface {
	Fetch(context.Context, models.ExternalLink) (LinkSnapshot, error)
}

func Client() *http.Client { return &http.Client{Timeout: 8 * time.Second} }
