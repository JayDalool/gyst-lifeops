package healthcoach

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultOpenAIModel is used when HEALTH_AI_MODEL is unset: OpenAI's flagship
// model, the same tier as the Anthropic default. Checked against
// https://developers.openai.com/api/docs/models/gpt-5.6-sol on 2026-10-11.
const DefaultOpenAIModel = "gpt-5.6-sol"

// openAIBaseURL is a constant on purpose: no environment variable
// (OPENAI_BASE_URL) can redirect health data and the key elsewhere. Tests use
// newOpenAI with their own URL.
const openAIBaseURL = "https://api.openai.com/v1"

// maxOpenAIBody caps how much of a response is read.
const maxOpenAIBody = 2 << 20

// OpenAI calls the Responses API over plain net/http. The key lives only in
// this struct and the Authorization header; the adapter never logs.
type OpenAI struct {
	key     string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAI builds the adapter against the pinned OpenAI base URL.
func NewOpenAI(apiKey, model string) *OpenAI {
	return newOpenAI(apiKey, model, openAIBaseURL, &http.Client{
		Timeout: 60 * time.Second,
		// A redirect would re-send the key and health data elsewhere.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	})
}

func newOpenAI(apiKey, model, baseURL string, client *http.Client) *OpenAI {
	if model == "" {
		model = DefaultOpenAIModel
	}
	return &OpenAI{key: apiKey, model: model, baseURL: baseURL, client: client}
}

func (o *OpenAI) Name() string { return "openai" }

type openAIMessageIn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIFunctionOut struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type openAITool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Strict      bool           `json:"strict"`
}

type openAIResponse struct {
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []json.RawMessage `json:"output"`
}

type openAIItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (o *OpenAI) Respond(ctx context.Context, req Request) (Response, error) {
	var out Response
	tools := make([]openAITool, 0, len(req.Tools))
	byName := map[string]Tool{}
	for _, t := range req.Tools {
		params := map[string]any{"type": "object", "properties": map[string]any{}}
		if t.Properties != nil {
			params["properties"] = t.Properties
		}
		if len(t.Required) > 0 {
			params["required"] = t.Required
		}
		tools = append(tools, openAITool{Type: "function", Name: t.Name, Description: t.Description, Parameters: params})
		byName[t.Name] = t
	}
	input := make([]any, 0, len(req.History)+maxSteps*3)
	for _, m := range req.History {
		role := "user"
		if m.Role == "assistant" {
			role = "assistant"
		}
		input = append(input, openAIMessageIn{Role: role, Content: m.Text})
	}
	for step := 0; step < maxSteps; step++ {
		body := map[string]any{
			"model":             o.model,
			"instructions":      req.System,
			"input":             input,
			"reasoning":         map[string]string{"effort": "low"},
			"max_output_tokens": 4000,
			"store":             false,
			"include":           []string{"reasoning.encrypted_content"},
		}
		if len(tools) > 0 {
			body["tools"] = tools
		}
		resp, err := o.post(ctx, body)
		if err != nil {
			return Response{}, err
		}
		switch {
		case resp.Status == "failed":
			return Response{}, ErrUnavailable
		case resp.Status == "incomplete":
			if resp.IncompleteDetails != nil && resp.IncompleteDetails.Reason == "content_filter" {
				return Response{}, ErrDeclined
			}
			return Response{}, ErrUnavailable
		}
		var text []string
		var results []any
		called := false
		for _, raw := range resp.Output {
			var it openAIItem
			if json.Unmarshal(raw, &it) != nil {
				return Response{}, ErrUnavailable
			}
			switch it.Type {
			case "message":
				for _, c := range it.Content {
					switch c.Type {
					case "refusal":
						return Response{}, ErrDeclined
					case "output_text":
						text = append(text, c.Text)
					}
				}
			case "function_call":
				called = true
				tool, ok := byName[it.Name]
				if !ok {
					results = append(results, openAIFunctionOut{"function_call_output", it.CallID, `{"error":"unknown tool"}`})
					continue
				}
				out.ToolsUsed = append(out.ToolsUsed, it.Name)
				args := strings.TrimSpace(it.Arguments)
				if args == "" {
					args = "{}"
				}
				result, err := tool.Run(ctx, json.RawMessage(args))
				if err != nil {
					msg, _ := json.Marshal(map[string]string{"error": err.Error()})
					result = string(msg)
				}
				results = append(results, openAIFunctionOut{"function_call_output", it.CallID, result})
			}
		}
		if !called {
			out.Text = strings.TrimSpace(strings.Join(text, "\n\n"))
			if out.Text == "" {
				return Response{}, ErrUnavailable
			}
			return out, nil
		}
		// Stateless loop (store:false): echo every output item back verbatim,
		// reasoning items included, then the tool results.
		for _, raw := range resp.Output {
			input = append(input, raw)
		}
		input = append(input, results...)
	}
	return Response{}, ErrUnavailable
}

// post sends one request and maps every failure to a fixed error. Vendor
// message text and bodies never leave this function.
func (o *OpenAI) post(ctx context.Context, body map[string]any) (openAIResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return openAIResponse{}, ErrUnavailable
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return openAIResponse{}, ErrUnavailable
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.key)
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := o.client.Do(httpReq)
	if err != nil {
		return openAIResponse{}, ErrUnavailable
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxOpenAIBody+1))
	if err != nil || len(raw) > maxOpenAIBody {
		return openAIResponse{}, ErrUnavailable
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return openAIResponse{}, mapOpenAIError(res.StatusCode, raw)
	}
	var resp openAIResponse
	if json.Unmarshal(raw, &resp) != nil {
		return openAIResponse{}, ErrUnavailable
	}
	return resp, nil
}

// mapOpenAIError reads only error.code from the body.
func mapOpenAIError(status int, body []byte) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrNotConfigured
	case http.StatusTooManyRequests:
		var e struct {
			Error struct {
				Code string `json:"code"`
				Type string `json:"type"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		switch e.Error.Code {
		case "insufficient_quota", "credit_balance_exhausted",
			"organization_spend_limit_exceeded", "project_spend_limit_exceeded",
			"organization_usage_limit_exceeded":
			return ErrNotConfigured
		}
		if e.Error.Type == "insufficient_quota" {
			return ErrNotConfigured
		}
		return ErrBusy
	}
	return ErrUnavailable
}
