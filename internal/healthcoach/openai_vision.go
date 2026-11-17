package healthcoach

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// DefaultOpenAIVisionModel is used for photo analysis when
// HEALTH_AI_VISION_MODEL is unset (Jay, 2026-11-16).
const DefaultOpenAIVisionModel = "gpt-6.1-sol"

// maxPhotoBytes bounds what is base64-encoded into one request. imageintake
// never produces more than a few hundred KB; this is a backstop.
const maxPhotoBytes = 1 << 20

var _ FoodPhotoAnalyzer = (*OpenAI)(nil)

// WithVisionModel sets the model used for photos; empty keeps the default.
func (o *OpenAI) WithVisionModel(model string) *OpenAI {
	if m := strings.TrimSpace(model); m != "" {
		o.visionModel = m
	}
	return o
}

func (o *OpenAI) photoModel() string {
	if o.visionModel == "" {
		return DefaultOpenAIVisionModel
	}
	return o.visionModel
}

// AnalyzeFoodPhoto sends one JPEG to the Responses API as an input_image data
// URL with a strict JSON-schema output, store:false, through the same pinned
// base URL, no-redirect client and response cap as Respond. The image and the
// model's output are never logged.
//
// Request shape checked against developers.openai.com on 2026-11-16:
// guides/images-vision (input_image, image_url data URL, detail) and
// guides/structured-outputs (text.format json_schema, strict).
func (o *OpenAI) AnalyzeFoodPhoto(ctx context.Context, jpeg []byte) (FoodPhotoResult, error) {
	if len(jpeg) == 0 || len(jpeg) > maxPhotoBytes {
		return FoodPhotoResult{}, ErrUnavailable
	}
	dataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpeg)
	body := map[string]any{
		"model":        o.photoModel(),
		"instructions": foodPhotoPrompt,
		"input": []any{map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "input_text", "text": "Identify the foods in this meal photo."},
				// "high" tiles a <=1024 px image into a handful of tiles,
				// which keeps the cost near 1k input tokens.
				map[string]any{"type": "input_image", "image_url": dataURL, "detail": "high"},
			},
		}},
		"text": map[string]any{"format": map[string]any{
			"type":   "json_schema",
			"name":   "food_photo_identification",
			"strict": true,
			"schema": foodPhotoSchema(),
		}},
		"reasoning":         map[string]string{"effort": "low"},
		"max_output_tokens": 4000,
		"store":             false,
	}
	resp, err := o.post(ctx, body)
	if err != nil {
		return FoodPhotoResult{}, err
	}
	switch {
	case resp.Status == "failed":
		return FoodPhotoResult{}, ErrUnavailable
	case resp.Status == "incomplete":
		if resp.IncompleteDetails != nil && resp.IncompleteDetails.Reason == "content_filter" {
			return FoodPhotoResult{}, ErrDeclined
		}
		return FoodPhotoResult{}, ErrPhotoUnreadable
	}
	var text []string
	for _, raw := range resp.Output {
		var it openAIItem
		if json.Unmarshal(raw, &it) != nil {
			return FoodPhotoResult{}, ErrPhotoUnreadable
		}
		if it.Type != "message" {
			continue
		}
		for _, c := range it.Content {
			switch c.Type {
			case "refusal":
				return FoodPhotoResult{}, ErrDeclined
			case "output_text":
				text = append(text, c.Text)
			}
		}
	}
	return ParseFoodPhoto(strings.TrimSpace(strings.Join(text, "")))
}
