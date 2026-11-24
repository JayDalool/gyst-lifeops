package healthcoach

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
)

// GYST-180: the photo request to each provider is pinned by a golden body
// captured from the code before AI macro estimates existed. The new estimate
// work may change only the system/instructions text and the schema subtree;
// the model, limits, image slot and the user sentence must stay as captured.
// The image base64 is replaced by a placeholder so the file holds no photo.

var updateGolden = flag.Bool("update-vision-golden", false, "rewrite the vision request goldens (review the diff by hand)")

const goldenUserText = "Identify the foods in this meal photo."

// goldenBody replaces the image data and returns an indented, key-sorted body.
func goldenBody(t *testing.T, req map[string]any, openai bool) ([]byte, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(req)
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if openai {
		content := m["input"].([]any)[0].(map[string]any)["content"].([]any)
		for _, p := range content {
			part := p.(map[string]any)
			if part["type"] == "input_image" {
				part["image_url"] = "data:image/jpeg;base64,<IMAGE_BASE64>"
			}
		}
	} else {
		content := m["messages"].([]any)[0].(map[string]any)["content"].([]any)
		for _, p := range content {
			part := p.(map[string]any)
			if part["type"] == "image" {
				part["source"].(map[string]any)["data"] = "<IMAGE_BASE64>"
			}
		}
	}
	out, _ := json.MarshalIndent(m, "", "  ")
	return append(out, '\n'), m
}

// withoutVariable drops what GYST-180 may change: system/instructions and the
// schema subtree.
func withoutVariable(m map[string]any, openai bool) map[string]any {
	raw, _ := json.Marshal(m)
	var c map[string]any
	_ = json.Unmarshal(raw, &c)
	if openai {
		delete(c, "instructions")
		delete(c["text"].(map[string]any)["format"].(map[string]any), "schema")
	} else {
		delete(c, "system")
		delete(c["output_config"].(map[string]any)["format"].(map[string]any), "schema")
	}
	return c
}

func checkGolden(t *testing.T, file string, req map[string]any, openai bool) {
	t.Helper()
	body, got := goldenBody(t, req, openai)
	path := "testdata/" + file
	if *updateGolden {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wantMap map[string]any
	if err := json.Unmarshal(want, &wantMap); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(withoutVariable(got, openai), withoutVariable(wantMap, openai)) {
		t.Fatalf("request differs from the golden outside system/instructions and schema:\n%s\nwant:\n%s", body, want)
	}
	if bytes.Contains(want, []byte("/9j/")) || bytes.Contains(want, []byte("ZmFrZS1pbWFnZQ")) {
		t.Fatal("golden holds image bytes")
	}
}

func TestAnthropicRequestGolden(t *testing.T) {
	a, f := newFake(t, textMessage("end_turn", goodVision))
	a.WithPhotoModel("claude-photo-test")
	if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "anthropic_food_photo_request.golden.json", f.requests[0], false)
	// The user text is byte-identical and is the only text block.
	content := f.requests[0]["messages"].([]any)[0].(map[string]any)["content"].([]any)
	if len(content) != 2 || content[1].(map[string]any)["text"] != goldenUserText {
		t.Fatalf("user content %v", content)
	}
}

func TestOpenAIRequestGolden(t *testing.T) {
	o, f := newOAIFake(t, jsonReply(200, visionReply(goodVision)))
	o.WithVisionModel("vision-test-model")
	if _, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "openai_food_photo_request.golden.json", f.requests[0], true)
	content := f.requests[0]["input"].([]any)[0].(map[string]any)["content"].([]any)
	if len(content) != 2 || content[0].(map[string]any)["text"] != goldenUserText ||
		!strings.HasPrefix(content[1].(map[string]any)["image_url"].(string), "data:image/jpeg;base64,") {
		t.Fatalf("user content %v", content)
	}
}
