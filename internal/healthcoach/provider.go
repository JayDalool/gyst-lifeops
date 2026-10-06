// Package healthcoach is GYST's provider-neutral Health Coach (Issue #64).
//
// GYST owns the context, the tools and the memory; a Provider only turns a
// request into a reply. Nothing here is trained on Jay's data: every answer
// is built from live, minimal context and allowlisted read-only tools.
//
// Privacy rules enforced in this package:
//   - provider calls are server-side; keys never reach templates, JS or logs
//   - provider errors are replaced by fixed messages before they leave here
//   - prompts and replies are never logged
package healthcoach

import (
	"context"
	"encoding/json"
	"errors"
)

// Message is one visible turn of the conversation.
type Message struct {
	Role string // "user" or "assistant"
	Text string
}

// Tool is an allowlisted, read-only capability the model may call. Run gets
// the raw JSON input and returns a compact JSON result for the model.
type Tool struct {
	Name        string
	Description string
	// Properties is the JSON-schema "properties" object; nil means no input.
	Properties map[string]any
	Required   []string
	Run        func(ctx context.Context, input json.RawMessage) (string, error)
}

// Request is everything a provider needs for one reply.
type Request struct {
	System  string
	History []Message // oldest first; the last entry is the user's question
	Tools   []Tool
}

// Response is the coach's reply. Action, when set, is one of Destinations.
type Response struct {
	Text      string
	Action    string
	ToolsUsed []string
}

// Provider produces one reply. Implementations run the tool loop themselves
// because each vendor's tool format differs.
type Provider interface {
	Name() string
	Respond(ctx context.Context, req Request) (Response, error)
}

// Errors safe to show to the user. Providers map every failure to one of
// these; vendor error text never reaches the page.
var (
	ErrNotConfigured = errors.New("The Health Coach isn't set up yet")
	ErrUnavailable   = errors.New("The Health Coach couldn't answer right now. Try again in a minute")
	ErrBusy          = errors.New("The Health Coach is busy. Try again in a moment")
	ErrDeclined      = errors.New("The Health Coach can't help with that request")
)

// Destinations is the navigation allowlist. The model may only name one of
// these keys; the server maps it to a route. Model-written URLs are never used.
var Destinations = map[string]struct{ Label, Path string }{
	"health_today":            {"Open Health Today", "/health"},
	"health_workout":          {"Open Workout", "/health/training"},
	"health_workout_progress": {"Open Workout progress", "/health/training?view=progress"},
	"health_nutrition":        {"Open Nutrition", "/health/nutrition"},
	"health_more":             {"Open Health More", "/health/more"},
	"health_plan":             {"Open Health plan", "/health/plan"},
	"health_progress":         {"Open weight history", "/health/progress"},
	"health_log_weight":       {"Log weight", "/health/progress#log"},
	"health_foods":            {"Open My foods", "/health/foods"},
	"health_checklist":        {"Open Daily checklist", "/health/checklist"},
}

// ActionTool lets the model attach one allowlisted navigation action.
func ActionTool(chosen *string) Tool {
	keys := make([]string, 0, len(Destinations))
	for k := range Destinations {
		keys = append(keys, k)
	}
	return Tool{
		Name:        "suggest_gyst_page",
		Description: "Attach one button that opens a GYST Health page relevant to your answer. Use at most once, only when the page helps.",
		Properties:  map[string]any{"destination": map[string]any{"type": "string", "enum": keys}},
		Required:    []string{"destination"},
		Run: func(_ context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Destination string `json:"destination"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", errors.New("invalid input")
			}
			if _, ok := Destinations[in.Destination]; !ok {
				return "", errors.New("unknown destination")
			}
			*chosen = in.Destination
			return `{"ok":true}`, nil
		},
	}
}
