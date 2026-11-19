package healthcoach

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var _ FoodPhotoAnalyzer = (*Anthropic)(nil)

// AnalyzeFoodPhoto sends one JPEG to the Messages API as a base64 image block
// with the same instructions and JSON schema as the OpenAI path, through
// structured outputs. The request carries only model, max_tokens, system,
// messages and output_config: no metadata, tools, betas or fallbacks, and
// nothing from the diary (#243), so exactly one model sees exactly one photo.
// The image and the model's output are never logged.
//
// Shape checked against the pinned anthropic-sdk-go v1.78.0 and
// platform.claude.com build-with-claude/vision and structured-outputs.
func (a *Anthropic) AnalyzeFoodPhoto(ctx context.Context, jpeg []byte) (FoodPhotoResult, error) {
	if len(jpeg) == 0 || len(jpeg) > maxPhotoBytes {
		return FoodPhotoResult{}, ErrUnavailable
	}
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     a.photoModelName(),
		MaxTokens: 4000,
		System:    []anthropic.TextBlockParam{{Text: foodPhotoPrompt}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(
			// Image before text, as the vision guide recommends.
			anthropic.NewImageBlockBase64("image/jpeg", base64.StdEncoding.EncodeToString(jpeg)),
			anthropic.NewTextBlock("Identify the foods in this meal photo."),
		)},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: foodPhotoSchema()},
		},
	})
	if err != nil {
		return FoodPhotoResult{}, mapAnthropicError(err)
	}
	// refusal and max_tokens can still carry JSON-looking text; they must not
	// reach the parser. Anything but a normal end is treated the same way.
	if resp.StopReason != anthropic.StopReasonEndTurn {
		return FoodPhotoResult{}, ErrPhotoUnreadable
	}
	var text []string
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			text = append(text, b.Text)
		}
	}
	return ParseFoodPhoto(strings.TrimSpace(strings.Join(text, "")))
}
