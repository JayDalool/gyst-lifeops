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

	"github.com/anthropics/anthropic-sdk-go/option"
)

// fakeAPI replays canned Messages API responses and records requests.
type fakeAPI struct {
	mu       sync.Mutex
	replies  []func(w http.ResponseWriter)
	requests []map[string]any
	headers  []http.Header
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	var req map[string]any
	_ = json.Unmarshal(body, &req)
	f.requests = append(f.requests, req)
	f.headers = append(f.headers, r.Header.Clone())
	reply := f.replies[0]
	if len(f.replies) > 1 {
		f.replies = f.replies[1:]
	}
	reply(w)
}

func jsonReply(status int, body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func message(stop string, content string) string {
	return `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"` + stop + `","content":` + content + `,"usage":{"input_tokens":1,"output_tokens":1}}`
}

func newFake(t *testing.T, replies ...func(http.ResponseWriter)) (*Anthropic, *fakeAPI) {
	t.Helper()
	f := &fakeAPI{replies: replies}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return newAnthropic("test-only-key", "", option.WithBaseURL(srv.URL), option.WithMaxRetries(0)), f
}

func TestAnthropicToolLoop(t *testing.T) {
	a, f := newFake(t,
		jsonReply(200, message("tool_use", `[{"type":"text","text":"Checking."},{"type":"tool_use","id":"tu_1","name":"get_today","input":{"day":"today"}},{"type":"tool_use","id":"tu_2","name":"rm_rf","input":{}}]`)),
		jsonReply(200, message("end_turn", `[{"type":"text","text":"You have 48 g of protein left."}]`)),
	)
	var gotInput string
	tools := []Tool{{Name: "get_today", Description: "today", Run: func(_ context.Context, in json.RawMessage) (string, error) {
		gotInput = string(in)
		return `{"protein_left_g":48}`, nil
	}}}
	resp, err := a.Respond(context.Background(), Request{System: "rules", History: []Message{{Role: "user", Text: "Protein left?"}}, Tools: tools})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "You have 48 g of protein left." || len(resp.ToolsUsed) != 1 || !strings.Contains(gotInput, "today") {
		t.Fatalf("resp %+v input %q", resp, gotInput)
	}
	if len(f.requests) != 2 {
		t.Fatalf("requests %d", len(f.requests))
	}
	first, second := f.requests[0], f.requests[1]
	if first["model"] != DefaultAnthropicModel || first["fallbacks"] != "default" {
		t.Fatalf("model/fallbacks: %v %v", first["model"], first["fallbacks"])
	}
	if oc, _ := first["output_config"].(map[string]any); oc["effort"] != "low" {
		t.Fatalf("effort %v", first["output_config"])
	}
	h := f.headers[0]
	if h.Get("X-Api-Key") != "test-only-key" || h.Get("Anthropic-Version") == "" || !strings.Contains(h.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Fatalf("headers %v", h)
	}
	// The second request carries both tool results: real data, and an error
	// for the tool that isn't on the allowlist.
	raw, _ := json.Marshal(second["messages"])
	for _, want := range []string{`"tool_use_id":"tu_1"`, `protein_left_g`, `"tool_use_id":"tu_2"`, `unknown tool`, `"is_error":true`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("second request missing %s", want)
		}
	}
}

func TestAnthropicErrorsAreSanitised(t *testing.T) {
	for _, tc := range []struct {
		reply func(http.ResponseWriter)
		want  error
	}{
		{jsonReply(200, `{"id":"m","type":"message","role":"assistant","model":"x","stop_reason":"refusal","content":[],"usage":{"input_tokens":1,"output_tokens":0}}`), ErrDeclined},
		{jsonReply(401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key test-only-key"}}`), ErrNotConfigured},
		{jsonReply(429, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`), ErrBusy},
		{jsonReply(500, `{"type":"error","error":{"type":"api_error","message":"echo: Jay weighs 134.5 kg"}}`), ErrUnavailable},
		{jsonReply(200, message("end_turn", `[]`)), ErrUnavailable},
	} {
		a, _ := newFake(t, tc.reply)
		_, err := a.Respond(context.Background(), Request{System: "s", History: []Message{{Role: "user", Text: "hi"}}})
		if !errors.Is(err, tc.want) {
			t.Fatalf("got %v, want %v", err, tc.want)
		}
		if strings.Contains(err.Error(), "134.5") || strings.Contains(err.Error(), "test-only-key") {
			t.Fatal("vendor detail leaked into the error")
		}
	}
}

func TestAnthropicStopsAfterMaxSteps(t *testing.T) {
	loop := jsonReply(200, message("tool_use", `[{"type":"tool_use","id":"tu","name":"get_today","input":{}}]`))
	a, f := newFake(t, loop)
	tools := []Tool{{Name: "get_today", Run: func(context.Context, json.RawMessage) (string, error) { return `{}`, nil }}}
	if _, err := a.Respond(context.Background(), Request{System: "s", History: []Message{{Role: "user", Text: "hi"}}, Tools: tools}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v", err)
	}
	if len(f.requests) != maxSteps {
		t.Fatalf("requests %d, want %d", len(f.requests), maxSteps)
	}
}

func TestActionToolAllowlist(t *testing.T) {
	var chosen string
	tool := ActionTool(&chosen)
	if _, err := tool.Run(context.Background(), json.RawMessage(`{"destination":"https://evil.example"}`)); err == nil || chosen != "" {
		t.Fatal("non-allowlisted destination accepted")
	}
	if _, err := tool.Run(context.Background(), json.RawMessage(`{"destination":"health_workout_progress"}`)); err != nil || chosen != "health_workout_progress" {
		t.Fatalf("allowlisted destination refused: %v", err)
	}
	for key, d := range Destinations {
		if !strings.HasPrefix(d.Path, "/health") || strings.Contains(d.Path, "//") {
			t.Fatalf("%s maps outside Health: %s", key, d.Path)
		}
	}
}
