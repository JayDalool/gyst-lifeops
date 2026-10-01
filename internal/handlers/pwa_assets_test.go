package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPWAShellAssetsMatchTemplates(t *testing.T) {
	worker, err := os.ReadFile("../../web/static/service-worker.js")
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Split(string(worker), "const SHELL = ")[1]
	line = strings.Split(line, ";\n")[0]
	var assets []string
	if err = json.Unmarshal([]byte(line), &assets); err != nil {
		t.Fatal(err)
	}
	layout, err := os.ReadFile("../../web/templates/layout.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range assets {
		if !strings.HasPrefix(asset, "/static/") || strings.Contains(asset, "three.") || strings.Contains(asset, "galaxy.js") {
			t.Fatalf("private/expensive precache entry: %s", asset)
		}
		path := strings.Split(strings.TrimPrefix(asset, "/static/"), "?")[0]
		if _, err = os.Stat(filepath.Join("../../web/static", path)); err != nil {
			t.Fatalf("missing shell asset %s", asset)
		}
		if !strings.Contains(string(layout), asset) {
			t.Fatalf("stale precache version %s", asset)
		}
	}
	for _, match := range regexp.MustCompile(`(?:href|src)="(/static/(?:css/[^" ]+|js/(?:theme|app|charts|completion-toast)\.js[^" ]*))"`).FindAllStringSubmatch(string(layout), -1) {
		if !strings.Contains(line, `"`+match[1]+`"`) {
			t.Fatalf("shell file missing from precache: %s", match[1])
		}
	}
	raw, err := os.ReadFile("../../web/static/manifest.webmanifest")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["short_name"] != "GYST" || manifest["start_url"] != "/today" {
		t.Fatal("PWA identity/start URL regressed")
	}
}
