package config

import (
	"fmt"
	"log/slog"
	"strings"
)

// A Config holds every secret as a plain string, so a stray %v, %+v, %#v,
// slog.Any or Println of the struct (or of Push, Providers, HealthAI) must not
// dump them. Each type prints only names, states and non-secret values.

func (c Config) String() string   { return "config.Config{" + strings.Join(c.Summary(), "; ") + "}" }
func (c Config) GoString() string { return c.String() }

// LogValue keeps slog.Any("cfg", cfg) redacted too.
func (c Config) LogValue() slog.Value { return slog.StringValue(c.String()) }

func (p Push) String() string {
	return fmt.Sprintf("config.Push{configured: %t}", p.Configured())
}
func (p Push) GoString() string     { return p.String() }
func (p Push) LogValue() slog.Value { return slog.StringValue(p.String()) }

func (p Providers) String() string {
	return fmt.Sprintf("config.Providers{github: %s, jira email: %s, jira token: %s, jira base url: %s, slack: %s}",
		state(p.GitHubToken != ""), state(p.JiraEmail != ""), state(p.JiraToken != ""),
		state(p.JiraBaseURL != ""), state(p.SlackToken != ""))
}
func (p Providers) GoString() string     { return p.String() }
func (p Providers) LogValue() slog.Value { return slog.StringValue(p.String()) }

func (h HealthAI) String() string {
	return fmt.Sprintf("config.HealthAI{provider: %q, model: %q, vision model: %q, anthropic key: %s, openai key: %s}",
		h.Provider, h.Model, h.VisionModel, state(h.AnthropicKey != ""), state(h.OpenAIKey != ""))
}
func (h HealthAI) GoString() string     { return h.String() }
func (h HealthAI) LogValue() slog.Value { return slog.StringValue(h.String()) }
