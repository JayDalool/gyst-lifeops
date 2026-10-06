package healthcoach

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// DefaultAnthropicModel is used when HEALTH_AI_MODEL is unset.
const DefaultAnthropicModel = "claude-opus-5-5"

// maxSteps bounds the tool loop: a handful of reads, then an answer.
const maxSteps = 6

// Anthropic calls the Claude Messages API through the official Go SDK.
type Anthropic struct {
	client anthropic.Client
	model  string
}

// NewAnthropic builds the adapter. The key is passed explicitly and kept
// only inside the SDK client.
func NewAnthropic(apiKey, model string) *Anthropic {
	return newAnthropic(apiKey, model)
}

// newAnthropic lets tests point the client at a local server.
func newAnthropic(apiKey, model string, extra ...option.RequestOption) *Anthropic {
	if model == "" {
		model = DefaultAnthropicModel
	}
	opts := append([]option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(60 * time.Second),
		option.WithMaxRetries(1),
	}, extra...)
	return &Anthropic{model: model, client: anthropic.NewClient(opts...)}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Respond(ctx context.Context, req Request) (Response, error) {
	var out Response
	tools := make([]anthropic.BetaToolUnionParam, 0, len(req.Tools))
	byName := map[string]Tool{}
	for _, t := range req.Tools {
		props := t.Properties
		if props == nil {
			props = map[string]any{}
		}
		p := anthropic.BetaToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.BetaToolInputSchemaParam{Properties: props, Required: t.Required},
		}
		tools = append(tools, anthropic.BetaToolUnionParam{OfTool: &p})
		byName[t.Name] = t
	}
	messages := make([]anthropic.BetaMessageParam, 0, len(req.History)+maxSteps*2)
	for _, m := range req.History {
		if m.Role == "assistant" {
			messages = append(messages, anthropic.BetaMessageParam{Role: anthropic.BetaMessageParamRoleAssistant, Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock(m.Text)}})
		} else {
			messages = append(messages, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(m.Text)))
		}
	}
	for step := 0; step < maxSteps; step++ {
		resp, err := a.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
			Model:        a.model,
			MaxTokens:    4000,
			System:       []anthropic.BetaTextBlockParam{{Text: req.System}},
			Messages:     messages,
			Tools:        tools,
			OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
			// Refusals are re-served by the API's default fallback route.
			Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
			Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		})
		if err != nil {
			return Response{}, mapAnthropicError(err)
		}
		if resp.StopReason == anthropic.BetaStopReasonRefusal {
			return Response{}, ErrDeclined
		}
		messages = append(messages, resp.ToParam())
		var text []string
		var results []anthropic.BetaContentBlockParamUnion
		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.BetaTextBlock:
				text = append(text, b.Text)
			case anthropic.BetaToolUseBlock:
				tool, ok := byName[b.Name]
				if !ok {
					results = append(results, anthropic.NewBetaToolResultBlock(b.ID, `{"error":"unknown tool"}`, true))
					continue
				}
				out.ToolsUsed = append(out.ToolsUsed, b.Name)
				result, err := tool.Run(ctx, []byte(b.JSON.Input.Raw()))
				if err != nil {
					msg, _ := json.Marshal(map[string]string{"error": err.Error()})
					results = append(results, anthropic.NewBetaToolResultBlock(b.ID, string(msg), true))
					continue
				}
				results = append(results, anthropic.NewBetaToolResultBlock(b.ID, result, false))
			}
		}
		if resp.StopReason != anthropic.BetaStopReasonToolUse {
			out.Text = strings.TrimSpace(strings.Join(text, "\n\n"))
			if out.Text == "" {
				return Response{}, ErrUnavailable
			}
			return out, nil
		}
		messages = append(messages, anthropic.NewBetaUserMessage(results...))
	}
	return Response{}, ErrUnavailable
}

// mapAnthropicError keeps vendor detail (which may echo request content) out
// of the page and the logs.
func mapAnthropicError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return ErrNotConfigured
		case http.StatusTooManyRequests, 529:
			return ErrBusy
		}
	}
	return ErrUnavailable
}
