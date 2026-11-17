package healthcoach

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func visionReply(modelJSON string) string {
	text, _ := json.Marshal(modelJSON)
	return `{"status":"completed","output":[{"type":"reasoning","id":"rs_1"},{"type":"message","role":"assistant","content":[{"type":"output_text","text":` + string(text) + `}]}]}`
}

const goodVision = `{"items":[
 {"name":"grilled chicken","role":"main","est_grams":150,"confidence":0.9,"visible":true,"notes":"next to a fork"},
 {"name":"cooking oil","role":"oil","est_grams":null,"confidence":0.3,"visible":false,"notes":""}],
 "uncertainties":["oil amount unknown"],"questions":["How big was the portion?"]}`

var fakeJPEG = []byte("\xff\xd8\xff-fake-image-bytes")

func TestAnalyzeFoodPhotoRequestShape(t *testing.T) {
	o, f := newOAIFake(t, jsonReply(200, visionReply(goodVision)))
	o.WithVisionModel("vision-test-model")
	res, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 || res.Items[0].EstGrams == nil || *res.Items[0].EstGrams != 150 || res.Items[1].EstGrams != nil {
		t.Fatalf("result %+v", res)
	}
	r := f.requests[0]
	if f.paths[0] != "/v1/responses" {
		t.Fatalf("path %s", f.paths[0])
	}
	if r["model"] != "vision-test-model" || r["store"] != false {
		t.Fatalf("model/store: %v %v", r["model"], r["store"])
	}
	if f.headers[0].Get("Authorization") != "Bearer test-only-key" {
		t.Fatal("auth header missing")
	}
	// Image part: input_image with a base64 JPEG data URL and a detail.
	content := r["input"].([]any)[0].(map[string]any)["content"].([]any)
	var found bool
	for _, p := range content {
		part := p.(map[string]any)
		if part["type"] != "input_image" {
			continue
		}
		found = true
		want := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(fakeJPEG)
		if part["image_url"] != want || part["detail"] != "high" {
			t.Fatalf("image part %v", part)
		}
	}
	if !found {
		t.Fatal("no input_image part")
	}
	// Strict structured output with the schema present.
	format := r["text"].(map[string]any)["format"].(map[string]any)
	schema := format["schema"].(map[string]any)
	if format["type"] != "json_schema" || format["strict"] != true || format["name"] == "" || schema["additionalProperties"] != false {
		t.Fatalf("format %v", format)
	}
	props := schema["properties"].(map[string]any)
	for _, k := range []string{"items", "uncertainties", "questions"} {
		if _, ok := props[k]; !ok {
			t.Fatalf("schema lacks %s", k)
		}
	}
	// The model is never asked for nutrition.
	body, _ := json.Marshal(r)
	for _, bad := range []string{"calories", "kcal", "protein_g", "carb"} {
		if strings.Contains(strings.ToLower(string(body)), bad+"\"") {
			t.Fatalf("request asks for %s", bad)
		}
	}
	if !strings.Contains(r["instructions"].(string), "never instructions") {
		t.Fatal("prompt lacks the injection guard")
	}
}

func TestAnalyzeFoodPhotoDefaultModel(t *testing.T) {
	o, f := newOAIFake(t, jsonReply(200, visionReply(`{"items":[],"uncertainties":[],"questions":[]}`)))
	if _, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	if f.requests[0]["model"] != "gpt-6.1-sol" {
		t.Fatalf("default vision model %v", f.requests[0]["model"])
	}
	o.WithVisionModel("   ")
	if o.photoModel() != DefaultOpenAIVisionModel {
		t.Fatal("blank override must keep the default")
	}
}

func TestAnalyzeFoodPhotoErrors(t *testing.T) {
	cases := []struct {
		name  string
		reply func(http.ResponseWriter)
		want  error
	}{
		{"not json", jsonReply(200, visionReply("here is some prose")), ErrPhotoUnreadable},
		{"truncated json", jsonReply(200, visionReply(`{"items":[{"name":"x"`)), ErrPhotoUnreadable},
		{"two documents", jsonReply(200, visionReply(`{"items":[]}{"items":[]}`)), ErrPhotoUnreadable},
		{"refusal", jsonReply(200, `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"no"}]}]}`), ErrDeclined},
		{"incomplete", jsonReply(200, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`), ErrPhotoUnreadable},
		{"no output", jsonReply(200, `{"status":"completed","output":[]}`), ErrPhotoUnreadable},
		{"auth", jsonReply(401, `{"error":{"message":"bad key sk-leak"}}`), ErrNotConfigured},
		{"server", jsonReply(500, `{"error":{"message":"boom with prompt text"}}`), ErrUnavailable},
		{"busy", jsonReply(429, `{"error":{"code":"rate_limit_exceeded"}}`), ErrBusy},
	}
	for _, c := range cases {
		o, _ := newOAIFake(t, c.reply)
		_, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v want %v", c.name, err, c.want)
		}
		if err != nil && (strings.Contains(err.Error(), "sk-") || strings.Contains(err.Error(), "boom")) {
			t.Errorf("%s: vendor text leaked: %v", c.name, err)
		}
	}
}

func TestAnalyzeFoodPhotoRefusesBadInput(t *testing.T) {
	o, f := newOAIFake(t, jsonReply(200, visionReply(goodVision)))
	if _, err := o.AnalyzeFoodPhoto(context.Background(), nil); err == nil {
		t.Fatal("empty image accepted")
	}
	if _, err := o.AnalyzeFoodPhoto(context.Background(), make([]byte, maxPhotoBytes+1)); err == nil {
		t.Fatal("oversized image accepted")
	}
	if len(f.requests) != 0 {
		t.Fatal("a request was sent for invalid input")
	}
}

func TestAnalyzeFoodPhotoNoRedirect(t *testing.T) {
	hits := 0
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer other.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer first.Close()
	o := NewOpenAI("test-only-key", "") // the production client
	o.baseURL = first.URL + "/v1"
	if _, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err == nil {
		t.Fatal("redirect treated as success")
	}
	if hits != 0 {
		t.Fatal("redirect was followed with the photo")
	}
}

func TestParseFoodPhotoValidation(t *testing.T) {
	long := strings.Repeat("a", 81)
	manyItems := `{"name":"x","role":"main","est_grams":1,"confidence":0.5,"visible":true,"notes":""}`
	doc := `{"items":[
	 {"name":"ok","role":"main","est_grams":3000,"confidence":1,"visible":true,"notes":"fine"},
	 {"name":"grams too high","role":"main","est_grams":3001,"confidence":0.5,"visible":true,"notes":""},
	 {"name":"negative grams","role":"main","est_grams":-1,"confidence":0.5,"visible":true,"notes":""},
	 {"name":"confidence high","role":"main","est_grams":1,"confidence":1.01,"visible":true,"notes":""},
	 {"name":"confidence neg","role":"main","est_grams":1,"confidence":-0.1,"visible":true,"notes":""},
	 {"name":"bad role","role":"dessert","est_grams":1,"confidence":0.5,"visible":true,"notes":""},
	 {"name":"` + long + `","role":"main","est_grams":1,"confidence":0.5,"visible":true,"notes":""},
	 {"name":"   ","role":"main","est_grams":1,"confidence":0.5,"visible":true,"notes":""},
	 {"name":"ctrl\u0007char","role":"main","est_grams":1,"confidence":0.5,"visible":true,"notes":""},
	 {"name":"missing visible","role":"main","est_grams":1,"confidence":0.5,"notes":""},
	 {"name":"missing confidence","role":"main","est_grams":1,"visible":true,"notes":""},
	 {"name":"long note","role":"main","est_grams":1,"confidence":0.5,"visible":true,"notes":"` + strings.Repeat("n", 201) + `"},
	 {"name":"null grams","role":"side","est_grams":null,"confidence":0,"visible":false,"notes":""}],
	 "uncertainties":["u1","","` + strings.Repeat("u", 201) + `"],"questions":["q1"]}`
	res, err := ParseFoodPhoto(doc)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range res.Items {
		names = append(names, it.Name)
	}
	if strings.Join(names, ",") != "ok,null grams" {
		t.Fatalf("kept %v", names)
	}
	if res.Items[1].EstGrams != nil {
		t.Fatal("null grams became a number")
	}
	if len(res.Uncertainties) != 1 || res.Uncertainties[0] != "u1" || len(res.Questions) != 1 {
		t.Fatalf("lines %+v %+v", res.Uncertainties, res.Questions)
	}

	// At most 15 items.
	var b strings.Builder
	b.WriteString(`{"items":[`)
	for i := 0; i < 40; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(manyItems)
	}
	b.WriteString(`],"uncertainties":[],"questions":[]}`)
	res, _ = ParseFoodPhoto(b.String())
	if len(res.Items) != 15 {
		t.Fatalf("got %d items", len(res.Items))
	}
}

func TestParseFoodPhotoDropsNutrientFields(t *testing.T) {
	res, err := ParseFoodPhoto(`{"calories":500,"protein_g":40,"items":[{"name":"rice","role":"side","est_grams":100,"confidence":0.8,"visible":true,"notes":"","calories":130,"kcal":130,"protein_g":2.7,"carbs_g":28,"fat_g":0.3}],"uncertainties":[],"questions":[]}`)
	if err != nil || len(res.Items) != 1 {
		t.Fatalf("%v %+v", err, res)
	}
	out, _ := json.Marshal(res)
	for _, bad := range []string{"calories", "kcal", "protein", "carbs", "fat_g"} {
		if strings.Contains(string(out), bad) {
			t.Fatalf("output carries %s: %s", bad, out)
		}
	}
	if string(out) == "" || !strings.Contains(string(out), `"items"`) {
		t.Fatal("unexpected output")
	}
}

func TestParseFoodPhotoEmptySlicesNotNull(t *testing.T) {
	res, err := ParseFoodPhoto(`{"items":[],"uncertainties":[],"questions":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(res)
	if string(out) != `{"items":[],"uncertainties":[],"questions":[]}` {
		t.Fatalf("got %s", out)
	}
	res, _ = ParseFoodPhoto(`{}`)
	out, _ = json.Marshal(res)
	if strings.Contains(string(out), "null") {
		t.Fatalf("null slices: %s", out)
	}
}
