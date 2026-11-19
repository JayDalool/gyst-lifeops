package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// A Config holds every secret as a plain string, so a stray print, log or JSON
// encode of the struct (or of Push, Providers, HealthAI) must not dump them.
// Each type prints only names, states and non-secret values: String and
// GoString cover %v and %#v, Format covers every verb (%s, %d, %t, %w ...),
// LogValue covers slog, and MarshalJSON covers encoding/json.

func (c Config) String() string   { return "config.Config{" + strings.Join(c.Summary(), "; ") + "}" }
func (c Config) GoString() string { return c.String() }
func (c Config) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, c.String())
}
func (c Config) LogValue() slog.Value         { return slog.StringValue(c.String()) }
func (c Config) MarshalJSON() ([]byte, error) { return json.Marshal(c.String()) }

func (p Push) String() string {
	return fmt.Sprintf("config.Push{configured: %t}", p.Configured())
}
func (p Push) GoString() string { return p.String() }
func (p Push) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, p.String())
}
func (p Push) LogValue() slog.Value         { return slog.StringValue(p.String()) }
func (p Push) MarshalJSON() ([]byte, error) { return json.Marshal(p.String()) }

func (p Providers) String() string {
	return fmt.Sprintf("config.Providers{github: %s, jira email: %s, jira token: %s, jira base url: %s, slack: %s}",
		state(p.GitHubToken != ""), state(p.JiraEmail != ""), state(p.JiraToken != ""),
		state(p.JiraBaseURL != ""), state(p.SlackToken != ""))
}
func (p Providers) GoString() string { return p.String() }
func (p Providers) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, p.String())
}
func (p Providers) LogValue() slog.Value         { return slog.StringValue(p.String()) }
func (p Providers) MarshalJSON() ([]byte, error) { return json.Marshal(p.String()) }

func (h HealthAI) String() string {
	return fmt.Sprintf("config.HealthAI{provider: %q, model: %q, vision model: %q, anthropic key: %s, openai key: %s}",
		h.Provider, h.Model, h.VisionModel, state(h.AnthropicKey != ""), state(h.OpenAIKey != ""))
}
func (h HealthAI) GoString() string { return h.String() }
func (h HealthAI) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, h.String())
}
func (h HealthAI) LogValue() slog.Value         { return slog.StringValue(h.String()) }
func (h HealthAI) MarshalJSON() ([]byte, error) { return json.Marshal(h.String()) }
