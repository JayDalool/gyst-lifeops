package app

import "testing"

func TestHealthCoachProviderSelection(t *testing.T) {
	cases := []struct {
		name string
		cfg  HealthAIConfig
		want string // "" means disabled
	}{
		{"openai", HealthAIConfig{Provider: "openai", OpenAIKey: "test-only-key"}, "openai"},
		{"openai spaced", HealthAIConfig{Provider: " OpenAI ", OpenAIKey: "test-only-key"}, "openai"},
		{"openai other key", HealthAIConfig{Provider: "openai", AnthropicKey: "test-only-key"}, ""},
		{"anthropic other key", HealthAIConfig{Provider: "anthropic", OpenAIKey: "test-only-key"}, ""},
		{"anthropic", HealthAIConfig{Provider: "anthropic", AnthropicKey: "test-only-key"}, "anthropic"},
		{"unknown", HealthAIConfig{Provider: "other", OpenAIKey: "k", AnthropicKey: "k"}, ""},
		{"empty", HealthAIConfig{OpenAIKey: "k"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := healthCoachProvider(c.cfg)
			if c.want == "" {
				if p != nil {
					t.Fatalf("want disabled, got %s", p.Name())
				}
				return
			}
			if p == nil || p.Name() != c.want {
				t.Fatalf("want %s, got %v", c.want, p)
			}
		})
	}
}
