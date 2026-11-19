package app

import (
	"testing"

	"lifeops-tracker/internal/healthcoach"
)

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
			// #243: both providers can read a meal photo.
			if _, ok := p.(healthcoach.FoodPhotoAnalyzer); !ok {
				t.Fatalf("%s is not a FoodPhotoAnalyzer", c.want)
			}
		})
	}
}

// The environment photo model (HEALTH_AI_VISION_MODEL) must reach Anthropic;
// without it the default applies.
func TestHealthCoachProviderAnthropicPhotoModel(t *testing.T) {
	for _, tc := range []struct{ vision, want string }{
		{"claude-photo-test", "claude-photo-test"},
		{"", healthcoach.DefaultAnthropicModel},
	} {
		p := healthCoachProvider(HealthAIConfig{Provider: "anthropic", AnthropicKey: "test-only-key", VisionModel: tc.vision})
		a, ok := p.(*healthcoach.Anthropic)
		if !ok || a.PhotoModel() != tc.want {
			t.Errorf("vision %q: got %v, want photo model %q", tc.vision, p, tc.want)
		}
	}
}
