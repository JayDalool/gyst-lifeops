package secretary

import (
	"context"
	"encoding/json"
	"errors"

	"lifeops-tracker/internal/healthcoach"
)

// ToolSpec is one read-only tool. Run returns a value that is encoded as
// compact JSON; its errors must already be fixed strings, because the text of
// an error can reach the model (and the model's vendor).
type ToolSpec struct {
	Name        string
	Description string
	// Properties is the JSON-schema "properties" object; nil means no input.
	Properties map[string]any
	Required   []string
	// Sources are the freshness sources ("hevy", "calendar", "mail") checked
	// before Run. Tools that read no synced copy leave it empty.
	Sources []string
	// Validate, when set, checks the input before any freshness check runs, so
	// malformed or out-of-range input answers with its fixed error and starts
	// no sync and no write.
	Validate func(input json.RawMessage) error
	Run      func(ctx context.Context, input json.RawMessage) (any, error)
}

// errDataUnavailable is what the Coach's tools have always answered when the
// data could not be read (GYST Issue #64). The cause is never shown.
var errDataUnavailable = errors.New("data unavailable")

// encode turns a tool result into the compact JSON the model receives.
func encode(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// tool adapts a spec to the provider's Tool type. before, when set, runs first
// (the Catalogue passes its freshness check).
func (s ToolSpec) tool(before func(ctx context.Context, sources []string)) healthcoach.Tool {
	run := s.Run
	return healthcoach.Tool{
		Name:        s.Name,
		Description: s.Description,
		Properties:  s.Properties,
		Required:    s.Required,
		Run: func(ctx context.Context, input json.RawMessage) (string, error) {
			if s.Validate != nil {
				if err := s.Validate(input); err != nil {
					return "", err
				}
			}
			if before != nil && len(s.Sources) > 0 {
				before(ctx, s.Sources)
			}
			v, err := run(ctx, input)
			if err != nil {
				return "", err
			}
			return encode(v)
		},
	}
}
