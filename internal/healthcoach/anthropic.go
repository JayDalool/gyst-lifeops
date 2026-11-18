package healthcoach

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// anthropicBaseURL is unexported and constant in a normal build so an
// environment variable can't redirect health data and the key elsewhere
// (#243). Only anthropic_browsercheck.go, compiled with the browsercheck tag,
// can change it, and only to a loopback address.
var anthropicBaseURL = "https://api.anthropic.com/"

// DefaultAnthropicModel is used when HEALTH_AI_MODEL is unset.
const DefaultAnthropicModel = "claude-opus-5-5"

// maxSteps bounds the tool loop: a handful of reads, then an answer.
const maxSteps = 6

// Anthropic calls the Claude Messages API through the official Go SDK.
type Anthropic struct {
	client     anthropic.Client
	model      string
	photoModel string
}

// WithPhotoModel sets the model used for meal photos; empty keeps the default.
func (a *Anthropic) WithPhotoModel(model string) *Anthropic {
	if m := strings.TrimSpace(model); m != "" {
		a.photoModel = m
	}
	return a
}

// PhotoModel reports the model used for meal photos, so wiring tests can prove
// the saved or environment photo model reaches the adapter.
func (a *Anthropic) PhotoModel() string { return a.photoModelName() }

func (a *Anthropic) photoModelName() string {
	if a.photoModel != "" {
		return a.photoModel
	}
	return DefaultAnthropicModel
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
		// The library's own environment reads (ANTHROPIC_AUTH_TOKEN, profile
		// files, ANTHROPIC_CUSTOM_HEADERS) must not add a token or headers to
		// requests that carry meal photos (#245 security review). The key is
		// always the explicit one below.
		option.WithoutEnvironmentDefaults(),
		// A redirect would replay the photo and the x-api-key header to another
		// host, which pinning the first URL does not prevent (docs/RULES.md
		// §1.6). The 3xx comes back as an ordinary error instead.
		option.WithHTTPClient(&http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}),
		// Pinned so an environment variable (ANTHROPIC_BASE_URL) can't redirect
		// health data and the key elsewhere. Tests override it explicitly.
		option.WithBaseURL(anthropicBaseURL),
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(60 * time.Second),
		option.WithMaxRetries(1),
		option.WithMiddleware(noRetryOnCredit),
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

// creditFailure classifies the documented "money" responses from the status
// and body alone. It returns nil for every other response.
//   - 402 billing_error: the prepaid credit is exhausted.
//   - 400 invalid_request_error "You have reached your specified ... API usage
//     limits": the monthly spend limit Jay set in the Console.
//   - 429 with error.details.error_code enforced_spend_limit_reached: the tier
//     spend cap. An ordinary 429 is a rate limit and stays retryable.
//   - 400 invalid_request_error "Your credit balance is too low to access the
//     Anthropic API...": the prepaid balance is used up. Matched on status,
//     error type and message prefix together so no other 400 is relabelled.
//     The wording comes from the code and security reviewers of #245, quoting
//     Anthropic's API guidance; it is not on the errors page read 2026-11-18.
//
// Sources: platform.claude.com/docs/en/api/errors and api/rate-limits,
// read 2026-11-18. Prepaid-balance exhaustion has no separate documented shape.
func creditFailure(status int, body []byte) error {
	var env struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Details struct {
				ErrorCode string `json:"error_code"`
			} `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &env) // a non-JSON body just matches nothing below
	switch {
	case status == http.StatusPaymentRequired || env.Error.Type == string(shared.ErrorTypeBillingError):
		return ErrCreditExhausted
	case status == http.StatusTooManyRequests && env.Error.Details.ErrorCode == "enforced_spend_limit_reached":
		return ErrSpendLimit
	case status == http.StatusBadRequest && env.Error.Type == "invalid_request_error" &&
		strings.HasPrefix(env.Error.Message, "You have reached your specified"):
		return ErrSpendLimit
	case status == http.StatusBadRequest && env.Error.Type == "invalid_request_error" &&
		strings.HasPrefix(env.Error.Message, "Your credit balance is too low"):
		return ErrCreditExhausted
	}
	return nil
}

// maxErrorProbe bounds how much of an error body is read to classify it;
// real error bodies are a few hundred bytes. The body is always handed on whole.
const maxErrorProbe = 1 << 20

type failingReader struct{ err error }

func (f failingReader) Read([]byte) (int, error) { return 0, f.err }

type readCloser struct {
	io.Reader
	io.Closer
}

// noRetryOnCredit stops the SDK retrying a response that more waiting cannot
// fix. The SDK obeys x-should-retry before it looks at the status, so the
// header is set here, ahead of that decision, for every recognised 400, 402
// and 429. The body is restored for the SDK exactly as the server sent it: the
// bytes read, then the rest of the original, which is still the one closed. A
// read that fails part-way keeps failing, so the SDK sees the error, not a
// partial body.
func noRetryOnCredit(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	res, err := next(req)
	if err != nil || res == nil {
		return res, err
	}
	switch res.StatusCode {
	case http.StatusBadRequest, http.StatusPaymentRequired, http.StatusTooManyRequests:
	default:
		return res, nil
	}
	if res.Body == nil {
		return res, nil
	}
	b, readErr := io.ReadAll(io.LimitReader(res.Body, maxErrorProbe))
	var rest io.Reader = res.Body
	if readErr != nil {
		rest = failingReader{readErr}
	}
	res.Body = readCloser{Reader: io.MultiReader(bytes.NewReader(b), rest), Closer: res.Body}
	if readErr == nil && (res.StatusCode == http.StatusPaymentRequired || creditFailure(res.StatusCode, b) != nil) {
		res.Header.Set("x-should-retry", "false")
	}
	return res, nil
}

// mapAnthropicError keeps vendor detail (which may echo request content) out
// of the page and the logs.
func mapAnthropicError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		if known := creditFailure(apiErr.StatusCode, []byte(apiErr.RawJSON())); known != nil {
			return known
		}
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return ErrNotConfigured
		case http.StatusTooManyRequests, 529:
			return ErrBusy
		}
	}
	return ErrUnavailable
}
