package healthcoach

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

func textMessage(stop, text string) func(http.ResponseWriter) {
	b, _ := json.Marshal(text)
	return jsonReply(200, message(stop, `[{"type":"text","text":`+string(b)+`}]`))
}

// newRetryingFake keeps the production retry count (one retry), so a test that
// must prove "no retry" can see a retry happen. Responses that are meant to be
// retried carry retry-after-ms (withRetryAfter) to keep the back-off short.
func newRetryingFake(t *testing.T, replies ...func(http.ResponseWriter)) (*Anthropic, *fakeAPI) {
	t.Helper()
	f := &fakeAPI{replies: replies}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return newAnthropic("test-only-key", "", option.WithBaseURL(srv.URL)), f
}

func withRetryAfter(reply func(http.ResponseWriter)) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("retry-after-ms", "1")
		reply(w)
	}
}

func TestAnthropicFoodPhotoRequestShape(t *testing.T) {
	a, f := newFake(t, textMessage("end_turn", goodVision))
	a.WithPhotoModel("claude-photo-test")
	if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 1 {
		t.Fatalf("requests %d", len(f.requests))
	}
	req := f.requests[0]
	// Exactly these keys: no metadata, tools, betas, fallbacks or diary data.
	var keys []string
	for k := range req {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if want := []string{"max_tokens", "messages", "model", "output_config", "system"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("request keys %v, want %v", keys, want)
	}
	if req["model"] != "claude-photo-test" {
		t.Fatalf("model %v", req["model"])
	}
	if h := f.headers[0]; h.Get("Anthropic-Beta") != "" {
		t.Fatalf("beta header sent: %v", h.Get("Anthropic-Beta"))
	}
	sys, _ := req["system"].([]any)
	if len(sys) != 1 || sys[0].(map[string]any)["text"] != foodPhotoPrompt {
		t.Fatalf("system %v", req["system"])
	}
	msgs, _ := req["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages %v", req["messages"])
	}
	content, _ := msgs[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content %v", content)
	}
	img, _ := content[0].(map[string]any)
	src, _ := img["source"].(map[string]any)
	if img["type"] != "image" || src["type"] != "base64" || src["media_type"] != "image/jpeg" ||
		src["data"] != base64.StdEncoding.EncodeToString(fakeJPEG) {
		t.Fatalf("image block %v", img)
	}
	if txt, _ := content[1].(map[string]any); txt["type"] != "text" {
		t.Fatalf("text block %v", content[1])
	}
	oc, _ := req["output_config"].(map[string]any)
	format, _ := oc["format"].(map[string]any)
	if oc["effort"] != "low" || format["type"] != "json_schema" {
		t.Fatalf("output_config %v", oc)
	}
	gotSchema, _ := json.Marshal(format["schema"])
	wantSchema, _ := json.Marshal(foodPhotoSchema())
	var g, w any
	_ = json.Unmarshal(gotSchema, &g)
	_ = json.Unmarshal(wantSchema, &w)
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("schema differs:\n%s\n%s", gotSchema, wantSchema)
	}
}

func TestAnthropicFoodPhotoDefaultModel(t *testing.T) {
	a, f := newFake(t, textMessage("end_turn", goodVision))
	if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	if f.requests[0]["model"] != DefaultAnthropicModel {
		t.Fatalf("model %v", f.requests[0]["model"])
	}
	// A blank saved photo model keeps the default.
	b, g := newFake(t, textMessage("end_turn", goodVision))
	b.WithPhotoModel("  ")
	if _, err := b.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil || g.requests[0]["model"] != DefaultAnthropicModel {
		t.Fatalf("blank photo model: %v %v", err, g.requests[0]["model"])
	}
}

func TestAnthropicFoodPhotoMatchesOpenAI(t *testing.T) {
	a, _ := newFake(t, textMessage("end_turn", goodVision))
	o, _ := newOAIFake(t, jsonReply(200, visionReply(goodVision)))
	got, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
	if err != nil {
		t.Fatal(err)
	}
	want, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || len(got.Items) != 2 {
		t.Fatalf("anthropic %+v, openai %+v", got, want)
	}
}

func TestAnthropicFoodPhotoUnreadable(t *testing.T) {
	// Nutrient fields in the reply are dropped, the items are kept.
	withNutrients := `{"items":[{"name":"test stew","role":"main","est_grams":200,"confidence":0.8,"visible":true,"notes":"","calories":999,"protein_g":77}],"uncertainties":[],"questions":[]}`
	a, _ := newFake(t, textMessage("end_turn", withNutrients))
	res, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
	if err != nil || len(res.Items) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if raw, _ := json.Marshal(res); strings.Contains(string(raw), "999") || strings.Contains(string(raw), "calories") {
		t.Fatalf("nutrient field survived: %s", raw)
	}
	for name, reply := range map[string]func(http.ResponseWriter){
		"malformed":  textMessage("end_turn", `{"items":[`),
		"refusal":    textMessage("refusal", goodVision),
		"max_tokens": textMessage("max_tokens", goodVision),
		"no text":    jsonReply(200, message("end_turn", `[]`)),
	} {
		a, _ := newFake(t, reply)
		if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); !errors.Is(err, ErrPhotoUnreadable) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAnthropicFoodPhotoOversize(t *testing.T) {
	for name, in := range map[string][]byte{"oversize": make([]byte, maxPhotoBytes+1), "empty": nil} {
		a, f := newFake(t, textMessage("end_turn", goodVision))
		if _, err := a.AnalyzeFoodPhoto(context.Background(), in); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: %v", name, err)
		}
		if len(f.requests) != 0 {
			t.Errorf("%s: %d requests", name, len(f.requests))
		}
	}
}

const (
	billing402   = `{"type":"error","error":{"type":"billing_error","message":"fictional"}}`
	spend400     = `{"type":"error","error":{"type":"invalid_request_error","message":"You have reached your specified API usage limits. You will regain access on 2026-12-01 at 00:00 UTC."}}`
	workspace400 = `{"type":"error","error":{"type":"invalid_request_error","message":"You have reached your specified workspace API usage limits. You will regain access on 2026-12-01 at 00:00 UTC."}}`
	lowCredit400 = `{"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."}}`
	other400     = `{"type":"error","error":{"type":"invalid_request_error","message":"messages.0.content.0: fictional validation problem"}}`
	cap429       = `{"type":"error","error":{"type":"rate_limit_error","message":"fictional","details":{"error_code":"enforced_spend_limit_reached"}}}`
)

// Each money response is one request, never retried, with the production
// retry count; ordinary 429, 529 and 500 still retry.
func TestAnthropicCreditNoRetry(t *testing.T) {
	retryHeader := func(status int, body string) func(http.ResponseWriter) {
		return func(w http.ResponseWriter) {
			w.Header().Set("x-should-retry", "true") // the SDK obeys this first
			jsonReply(status, body)(w)
		}
	}
	for _, tc := range []struct {
		name  string
		reply func(http.ResponseWriter)
		want  error
	}{
		{"402 billing_error", jsonReply(402, billing402), ErrCreditExhausted},
		{"402 x-should-retry true", retryHeader(402, billing402), ErrCreditExhausted},
		{"400 spend limit", jsonReply(400, spend400), ErrSpendLimit},
		{"400 workspace spend limit", jsonReply(400, workspace400), ErrSpendLimit},
		{"400 spend limit x-should-retry true", retryHeader(400, spend400), ErrSpendLimit},
		{"400 low credit", jsonReply(400, lowCredit400), ErrCreditExhausted},
		{"400 low credit x-should-retry true", retryHeader(400, lowCredit400), ErrCreditExhausted},
		{"429 tier cap x-should-retry true", retryHeader(429, cap429), ErrSpendLimit},
		{"429 tier cap", jsonReply(429, cap429), ErrSpendLimit},
		{"429 tier cap, long body", jsonReply(429, strings.Replace(cap429, `"fictional"`, `"`+strings.Repeat("x", 200<<10)+`"`, 1)), ErrSpendLimit},
	} {
		for _, via := range []string{"coach", "camera"} {
			a, f := newRetryingFake(t, tc.reply)
			var err error
			if via == "coach" {
				_, err = a.Respond(context.Background(), Request{System: "s", History: []Message{{Role: "user", Text: "hi"}}})
			} else {
				_, err = a.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("%s/%s: got %v, want %v", tc.name, via, err, tc.want)
			}
			if len(f.requests) != 1 {
				t.Errorf("%s/%s: %d requests, want 1", tc.name, via, len(f.requests))
			}
			if err != nil && (strings.Contains(err.Error(), "fictional") || strings.Contains(err.Error(), "regain")) {
				t.Errorf("%s/%s: vendor text in %q", tc.name, via, err)
			}
		}
	}
	// Controls: other 400 and 401 are one request and never relabelled.
	for _, tc := range []struct {
		name  string
		reply func(http.ResponseWriter)
		want  error
	}{
		{"other 400", jsonReply(400, other400), ErrUnavailable},
		{"400 with the credit words but another type", jsonReply(400, strings.Replace(lowCredit400, "invalid_request_error", "api_error", 1)), ErrUnavailable},
		{"401", jsonReply(401, `{"type":"error","error":{"type":"authentication_error","message":"x"}}`), ErrNotConfigured},
	} {
		a, f := newRetryingFake(t, tc.reply)
		if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
		if len(f.requests) != 1 {
			t.Errorf("%s: %d requests, want 1", tc.name, len(f.requests))
		}
	}
	// Controls: these are still retried (2 requests with one retry).
	for _, tc := range []struct {
		name  string
		reply func(http.ResponseWriter)
		want  error
	}{
		{"429 rate limit", withRetryAfter(jsonReply(429, `{"type":"error","error":{"type":"rate_limit_error","message":"slow"}}`)), ErrBusy},
		{"529", withRetryAfter(jsonReply(529, `{"type":"error","error":{"type":"overloaded_error","message":"x"}}`)), ErrBusy},
		{"500", withRetryAfter(jsonReply(500, `{"type":"error","error":{"type":"api_error","message":"x"}}`)), ErrUnavailable},
	} {
		a, f := newRetryingFake(t, tc.reply)
		if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
		if len(f.requests) != 2 {
			t.Errorf("%s control: %d requests, want 2", tc.name, len(f.requests))
		}
	}
}

type closeTracker struct {
	io.Reader
	closed atomic.Bool
}

func (c *closeTracker) Close() error { c.closed.Store(true); return nil }

// The middleware reads a 429 body to classify it. It must hand the SDK the
// whole body (not a truncated copy) and close the original.
func TestNoRetryOnCreditRestoresBody(t *testing.T) {
	long := strings.Replace(`{"error":{"type":"rate_limit_error","message":"PAD"}}`, "PAD", strings.Repeat("y", 300<<10), 1)
	orig := &closeTracker{Reader: strings.NewReader(long)}
	res, err := noRetryOnCredit(httptest.NewRequest("POST", "/", nil), func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: http.Header{}, Body: orig}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(res.Body)
	if string(got) != long {
		t.Fatalf("body truncated: %d of %d bytes", len(got), len(long))
	}
	// Closing the restored body closes the original connection's body.
	_ = res.Body.Close()
	if !orig.closed.Load() {
		t.Fatal("original body not closed")
	}
	if res.Header.Get("x-should-retry") != "" {
		t.Fatal("an ordinary 429 must stay retryable")
	}
}

type panicBody struct{ closed atomic.Bool }

func (p *panicBody) Read([]byte) (int, error) { panic("2xx body must not be read") }
func (p *panicBody) Close() error             { p.closed.Store(true); return nil }

func TestNoRetryOnCreditLeavesOtherResponsesAlone(t *testing.T) {
	for _, status := range []int{200, 401, 500, 529} {
		body := &panicBody{}
		res, err := noRetryOnCredit(httptest.NewRequest("POST", "/", nil), func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Header: http.Header{}, Body: body}, nil
		})
		if err != nil || res.Body != io.ReadCloser(body) || res.Header.Get("x-should-retry") != "" {
			t.Errorf("%d touched: %v", status, err)
		}
	}
}

// A body that fails part-way must reach the SDK as an error, not as the bytes
// read so far, and the original must still be closed.
func TestNoRetryOnCreditKeepsReadErrors(t *testing.T) {
	boom := errors.New("connection dropped")
	orig := &closeTracker{Reader: io.MultiReader(strings.NewReader(`{"error":{"type":"rate_limit_error"`), failingReader{boom})}
	res, _ := noRetryOnCredit(httptest.NewRequest("POST", "/", nil), func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: http.Header{}, Body: orig}, nil
	})
	if _, err := io.ReadAll(res.Body); !errors.Is(err, boom) {
		t.Fatalf("read error lost: %v", err)
	}
	_ = res.Body.Close()
	if !orig.closed.Load() || res.Header.Get("x-should-retry") != "" {
		t.Fatal("original not closed, or a failed read was classified")
	}
}

// A body larger than the probe cap is not classified but is still passed on
// whole, so an ordinary oversize 429 keeps its normal path.
func TestNoRetryOnCreditHugeBodyPassesWhole(t *testing.T) {
	long := `{"error":{"type":"rate_limit_error","message":"` + strings.Repeat("z", maxErrorProbe+500) + `"}}`
	res, _ := noRetryOnCredit(httptest.NewRequest("POST", "/", nil), func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: http.Header{}, Body: &closeTracker{Reader: strings.NewReader(long)}}, nil
	})
	got, _ := io.ReadAll(res.Body)
	if string(got) != long {
		t.Fatalf("got %d of %d bytes", len(got), len(long))
	}
}

// Pinning the base URL pins only the first request: a redirect must never be
// followed, so the photo and the key header can't be replayed elsewhere.
func TestAnthropicDoesNotFollowRedirects(t *testing.T) {
	for _, status := range []int{307, 308} {
		var hits atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
		t.Cleanup(target.Close)
		origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+"/v1/messages", status)
		}))
		t.Cleanup(origin.Close)
		a := newAnthropic("test-only-key", "", option.WithBaseURL(origin.URL))
		if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%d camera: %v", status, err)
		}
		if _, err := a.Respond(context.Background(), Request{System: "s", History: []Message{{Role: "user", Text: "hi"}}}); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%d coach: %v", status, err)
		}
		if hits.Load() != 0 {
			t.Errorf("%d: redirect target contacted %d times", status, hits.Load())
		}
	}
}

// The library's own environment must add no token, header or key; the key
// handed to NewAnthropic (Settings or server environment, same constructor)
// is the one sent.
func TestAnthropicIgnoresLibraryEnvironment(t *testing.T) {
	// The library reads each of these only when the ones before it are unset,
	// so each is checked on its own.
	for name, env := range map[string]map[string]string{
		"api key":        {"ANTHROPIC_API_KEY": "env-key-fictional"},
		"auth token":     {"ANTHROPIC_AUTH_TOKEN": "env-token-fictional"},
		"custom headers": {"ANTHROPIC_CUSTOM_HEADERS": "X-Env-Header: sneaky"},
		"profile":        {"ANTHROPIC_PROFILE": "nonexistent"},
	} {
		for _, key := range []string{"settings-key-fictional", "server-env-key-fictional"} {
			t.Run(name+"/"+key, func(t *testing.T) {
				for _, k := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_CUSTOM_HEADERS", "ANTHROPIC_PROFILE"} {
					t.Setenv(k, "")
				}
				t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:9/")
				for k, v := range env {
					t.Setenv(k, v)
				}
				f := &fakeAPI{replies: []func(http.ResponseWriter){textMessage("end_turn", goodVision)}}
				srv := httptest.NewServer(f)
				defer srv.Close()
				a := newAnthropic(key, "", option.WithBaseURL(srv.URL))
				if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil || len(f.headers) != 1 {
					t.Fatalf("%v (%d requests)", err, len(f.headers))
				}
				h := f.headers[0]
				if h.Get("X-Api-Key") != key || h.Get("Authorization") != "" || h.Get("X-Env-Header") != "" {
					t.Errorf("headers %v", h)
				}
			})
		}
	}
}

// Vendor text from the money responses never reaches the returned error.
func TestAnthropicCreditErrorsAreSanitised(t *testing.T) {
	for _, body := range []string{lowCredit400, spend400, billing402} {
		status := 400
		if body == billing402 {
			status = 402
		}
		a, _ := newFake(t, jsonReply(status, strings.Replace(body, `"message":"`, `"message":"echo test-only-key 134.5 kg `, 1)))
		_, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
		if err == nil || strings.Contains(err.Error(), "134.5") || strings.Contains(err.Error(), "test-only-key") {
			t.Fatalf("leak or no error: %v", err)
		}
	}
}

func TestAnthropicFoodPhotoLogsNothing(t *testing.T) {
	var slogBuf, logBuf bytes.Buffer
	oldSlog, oldWriter, oldFlags := slog.Default(), log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewTextHandler(&slogBuf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	log.SetOutput(&logBuf)
	t.Cleanup(func() { slog.SetDefault(oldSlog); log.SetOutput(oldWriter); log.SetFlags(oldFlags) })

	for _, reply := range []func(http.ResponseWriter){
		textMessage("end_turn", goodVision),
		textMessage("refusal", goodVision),
		jsonReply(402, `{"type":"error","error":{"type":"billing_error","message":"vendor-detail-fictional"}}`),
		jsonReply(400, strings.Replace(spend400, "You have", "vendor-detail-fictional You have", 1)),
		jsonReply(400, strings.Replace(lowCredit400, "Your credit", "vendor-detail-fictional Your credit", 1)),
	} {
		a, _ := newFake(t, reply)
		_, _ = a.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
	}
	all := slogBuf.String() + logBuf.String()
	for _, secret := range []string{base64.StdEncoding.EncodeToString(fakeJPEG), "test-only-key", "grilled chicken", "oil amount unknown", "vendor-detail-fictional"} {
		if strings.Contains(all, secret) {
			t.Errorf("log output contains %q", secret)
		}
	}
}
