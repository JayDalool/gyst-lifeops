package healthcoach

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// oaiFake replays canned Responses API replies and records requests.
type oaiFake struct {
	mu       sync.Mutex
	replies  []func(http.ResponseWriter)
	requests []map[string]any
	headers  []http.Header
	paths    []string
}

func (f *oaiFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	var req map[string]any
	_ = json.Unmarshal(body, &req)
	f.requests = append(f.requests, req)
	f.headers = append(f.headers, r.Header.Clone())
	f.paths = append(f.paths, r.URL.Path)
	reply := f.replies[0]
	if len(f.replies) > 1 {
		f.replies = f.replies[1:]
	}
	reply(w)
}

func newOAIFake(t *testing.T, replies ...func(http.ResponseWriter)) (*OpenAI, *oaiFake) {
	t.Helper()
	f := &oaiFake{replies: replies}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return newOpenAI("test-only-key", "", srv.URL+"/v1", srv.Client()), f
}

const oaiMsg = `{"status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"Looks steady."}]}]}`

func oaiCalls(calls ...string) string {
	return `{"status":"completed","output":[{"type":"reasoning","id":"rs_1","encrypted_content":"abc"},` + strings.Join(calls, ",") + `]}`
}

func oaiCall(id, name, args string) string {
	a, _ := json.Marshal(args)
	return `{"type":"function_call","id":"fc_` + id + `","call_id":"` + id + `","name":"` + name + `","arguments":` + string(a) + `}`
}

func oaiReq() Request {
	return Request{System: "rules", History: []Message{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "hello"}, {Role: "user", Text: "weight?"}}}
}

func TestOpenAIRequestShape(t *testing.T) {
	o, f := newOAIFake(t, jsonReply(200, oaiMsg))
	req := oaiReq()
	req.Tools = []Tool{{Name: "get_today", Description: "today", Run: func(context.Context, json.RawMessage) (string, error) { return "{}", nil }}}
	resp, err := o.Respond(context.Background(), req)
	if err != nil || resp.Text != "Looks steady." {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	r := f.requests[0]
	if f.paths[0] != "/v1/responses" || r["model"] != DefaultOpenAIModel || r["instructions"] != "rules" {
		t.Fatalf("bad request %v %v", f.paths[0], r)
	}
	if r["store"] != false || r["max_output_tokens"] != float64(4000) {
		t.Fatalf("store/max: %v", r)
	}
	if r["reasoning"].(map[string]any)["effort"] != "low" || r["include"].([]any)[0] != "reasoning.encrypted_content" {
		t.Fatalf("reasoning/include: %v", r)
	}
	tool := r["tools"].([]any)[0].(map[string]any)
	if tool["type"] != "function" || tool["strict"] != false {
		t.Fatalf("tool: %v", tool)
	}
	if _, has := tool["parameters"].(map[string]any)["required"]; has {
		t.Fatal("required must be omitted when empty")
	}
	if f.headers[0].Get("Authorization") != "Bearer test-only-key" || f.headers[0].Get("x-api-key") != "" {
		t.Fatal("bad auth headers")
	}
	in := r["input"].([]any)
	if len(in) != 3 || in[0].(map[string]any)["role"] != "user" || in[1].(map[string]any)["role"] != "assistant" || in[2].(map[string]any)["content"] != "weight?" {
		t.Fatalf("history order: %v", in)
	}
}

func TestOpenAIToolLoop(t *testing.T) {
	var got string
	o, f := newOAIFake(t,
		jsonReply(200, oaiCalls(oaiCall("c1", "get_today", `{"d":"x"}`), oaiCall("c2", "rm_rf", `{}`))),
		jsonReply(200, oaiMsg))
	req := oaiReq()
	req.Tools = []Tool{{Name: "get_today", Run: func(_ context.Context, in json.RawMessage) (string, error) {
		got = string(in)
		return `{"kg":99.9}`, nil
	}}}
	resp, err := o.Respond(context.Background(), req)
	if err != nil || resp.Text != "Looks steady." {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if len(resp.ToolsUsed) != 1 || resp.ToolsUsed[0] != "get_today" || got != `{"d":"x"}` || len(f.requests) != 2 {
		t.Fatalf("used=%v got=%s reqs=%d", resp.ToolsUsed, got, len(f.requests))
	}
	in := f.requests[1]["input"].([]any)
	// 3 history + reasoning + 2 function_call + 2 outputs
	if len(in) != 8 {
		t.Fatalf("second input len %d: %v", len(in), in)
	}
	o1, o2 := in[6].(map[string]any), in[7].(map[string]any)
	if o1["type"] != "function_call_output" || o1["call_id"] != "c1" || o1["output"] != `{"kg":99.9}` {
		t.Fatalf("o1 %v", o1)
	}
	if o2["call_id"] != "c2" || o2["output"] != `{"error":"unknown tool"}` {
		t.Fatalf("o2 %v", o2)
	}
	if in[4].(map[string]any)["type"] != "function_call" {
		t.Fatal("function_call items not echoed")
	}
}

func TestOpenAIToolErrorAndEmptyArguments(t *testing.T) {
	var got string
	o, f := newOAIFake(t, jsonReply(200, oaiCalls(oaiCall("c1", "t", ""))), jsonReply(200, oaiMsg))
	req := oaiReq()
	req.Tools = []Tool{{Name: "t", Run: func(_ context.Context, in json.RawMessage) (string, error) {
		got = string(in)
		return "", errors.New("invalid input")
	}}}
	if _, err := o.Respond(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got != "{}" {
		t.Fatalf("args %q", got)
	}
	in := f.requests[1]["input"].([]any)
	if in[len(in)-1].(map[string]any)["output"] != `{"error":"invalid input"}` {
		t.Fatalf("output %v", in[len(in)-1])
	}
}

func TestOpenAIStopsAfterMaxSteps(t *testing.T) {
	o, f := newOAIFake(t, jsonReply(200, oaiCalls(oaiCall("c", "t", "{}"))))
	req := oaiReq()
	req.Tools = []Tool{{Name: "t", Run: func(context.Context, json.RawMessage) (string, error) { return "{}", nil }}}
	if _, err := o.Respond(context.Background(), req); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v", err)
	}
	if len(f.requests) != maxSteps {
		t.Fatalf("requests %d", len(f.requests))
	}
}

func TestOpenAIErrorsAreSanitised(t *testing.T) {
	echo := func(code string) string {
		return `{"error":{"message":"you weigh 99.9 kg (fictional) test-only-key","type":"x","code":"` + code + `"}}`
	}
	cases := []struct {
		name  string
		reply func(http.ResponseWriter)
		want  error
	}{
		{"401", jsonReply(401, echo("invalid_api_key")), ErrNotConfigured},
		{"403", jsonReply(403, echo("x")), ErrNotConfigured},
		{"429 rate", jsonReply(429, echo("rate_limit_exceeded")), ErrBusy},
		{"429 quota", jsonReply(429, echo("insufficient_quota")), ErrNotConfigured},
		{"429 org spend", jsonReply(429, echo("organization_spend_limit_exceeded")), ErrNotConfigured},
		{"429 project spend", jsonReply(429, echo("project_spend_limit_exceeded")), ErrNotConfigured},
		{"429 org usage", jsonReply(429, echo("organization_usage_limit_exceeded")), ErrNotConfigured},
		{"429 credit", jsonReply(429, echo("credit_balance_exhausted")), ErrNotConfigured},
		{"429 slow_down", jsonReply(429, echo("slow_down")), ErrBusy},
		{"500", jsonReply(500, echo("x")), ErrUnavailable},
		{"400", jsonReply(400, echo("x")), ErrUnavailable},
		{"malformed", jsonReply(200, `{not json 99.9 kg`), ErrUnavailable},
		{"failed", jsonReply(200, `{"status":"failed","error":{"message":"99.9 kg (fictional)"},"output":[]}`), ErrUnavailable},
		{"empty", jsonReply(200, `{"status":"completed","output":[]}`), ErrUnavailable},
		{"refusal", jsonReply(200, `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"99.9 kg (fictional)"}]}]}`), ErrDeclined},
		{"content_filter", jsonReply(200, `{"status":"incomplete","incomplete_details":{"reason":"content_filter"},"output":[]}`), ErrDeclined},
		{"max tokens", jsonReply(200, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`), ErrUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, _ := newOAIFake(t, c.reply)
			_, err := o.Respond(context.Background(), oaiReq())
			if err != c.want {
				t.Fatalf("got %v want %v", err, c.want)
			}
			if s := err.Error(); strings.Contains(s, "99.9") || strings.Contains(s, "test-only-key") {
				t.Fatalf("leaked: %s", s)
			}
		})
	}
}

type oaiCapture struct{ urls []string }

func (c *oaiCapture) RoundTrip(r *http.Request) (*http.Response, error) {
	c.urls = append(c.urls, r.URL.String())
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(oaiMsg)), Request: r}, nil
}

func TestOpenAIBaseURLIsPinned(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "http://attacker.invalid")
	ct := &oaiCapture{}
	o := NewOpenAI("test-only-key", "")
	o.client.Transport = ct
	if _, err := o.Respond(context.Background(), oaiReq()); err != nil {
		t.Fatal(err)
	}
	if len(ct.urls) != 1 || !strings.HasPrefix(ct.urls[0], "https://api.openai.com/v1/responses") {
		t.Fatalf("urls %v", ct.urls)
	}
}

func TestOpenAIDoesNotFollowRedirects(t *testing.T) {
	var hits int
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer other.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer first.Close()
	o := NewOpenAI("test-only-key", "")
	o.baseURL = first.URL + "/v1"
	if _, err := o.Respond(context.Background(), oaiReq()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v", err)
	}
	if hits != 0 {
		t.Fatalf("redirect target got %d requests", hits)
	}
}

func TestOpenAIRedirect301Refused(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://attacker.invalid/", http.StatusMovedPermanently)
	}))
	defer first.Close()
	o := NewOpenAI("test-only-key", "")
	o.baseURL = first.URL + "/v1"
	if _, err := o.Respond(context.Background(), oaiReq()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v", err)
	}
}

func TestOpenAICallerCancel(t *testing.T) {
	o, _ := newOAIFake(t, jsonReply(200, oaiMsg))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := o.Respond(ctx, oaiReq()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v", err)
	}
}

func TestOpenAIBodyCap(t *testing.T) {
	big := `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"` + strings.Repeat("a", maxOpenAIBody) + `"}]}]}`
	o, _ := newOAIFake(t, jsonReply(200, big))
	if _, err := o.Respond(context.Background(), oaiReq()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v", err)
	}
}

func TestOpenAIModelOverrideAndName(t *testing.T) {
	o := newOpenAI("test-only-key", "custom-model", "http://x", nil)
	if o.Name() != "openai" || o.model != "custom-model" {
		t.Fatal("name/model")
	}
}
