package app

import "testing"

// GYST-169: the environment wins over Settings whenever it names a provider
// or holds a provider key.
func TestHealthAIConfigEnvManaged(t *testing.T) {
	cases := []struct {
		name string
		cfg  HealthAIConfig
		want bool
	}{
		{"empty", HealthAIConfig{}, false},
		{"model only", HealthAIConfig{Model: "x"}, false},
		{"blank provider", HealthAIConfig{Provider: "  "}, false},
		{"provider", HealthAIConfig{Provider: "openai"}, true},
		{"anthropic key only", HealthAIConfig{AnthropicKey: "k"}, true},
		{"openai key only", HealthAIConfig{OpenAIKey: "k"}, true},
	}
	for _, c := range cases {
		if got := c.cfg.EnvManaged(); got != c.want {
			t.Errorf("%s: EnvManaged = %v, want %v", c.name, got, c.want)
		}
	}
}
