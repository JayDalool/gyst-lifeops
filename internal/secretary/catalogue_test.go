package secretary

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

func okRun(context.Context, json.RawMessage) (any, error) { return 1, nil }

func TestRegisterRejects(t *testing.T) {
	allow := []string{"a", "b"}
	good := ToolSpec{Name: "a", Description: "d", Run: okRun}
	for name, tc := range map[string]struct {
		specs []ToolSpec
		want  string
	}{
		"unknown name":   {[]ToolSpec{{Name: "zzz", Description: "d", Run: okRun}}, "not on the allowlist"},
		"duplicate":      {[]ToolSpec{good, good}, "twice"},
		"empty desc":     {[]ToolSpec{{Name: "a", Run: okRun}}, "no description"},
		"nil run":        {[]ToolSpec{{Name: "a", Description: "d"}}, "no Run"},
		"unknown source": {[]ToolSpec{{Name: "a", Description: "d", Run: okRun, Sources: []string{"fax"}}}, "unknown source"},
		"empty name":     {[]ToolSpec{{Description: "d", Run: okRun}}, "not on the allowlist"},
		"case differs":   {[]ToolSpec{{Name: "A", Description: "d", Run: okRun}}, "not on the allowlist"},
	} {
		if _, err := register(allow, tc.specs); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", name, err, tc.want)
		}
	}
	got, err := register(allow, []ToolSpec{{Name: "b", Description: "d", Run: okRun}, good})
	if err != nil || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("allowlist order not applied: %+v %v", got, err)
	}
}

func newBare(t *testing.T) *Catalogue {
	t.Helper()
	cat, err := New(Deps{Service: services.New(repository.New(nil)), Health: &recordingHealth{}})
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestNewNeedsServiceAndHealth(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("empty Deps accepted")
	}
	if _, err := New(Deps{Service: services.New(repository.New(nil))}); err == nil {
		t.Fatal("missing Health accepted")
	}
	if _, err := New(Deps{Service: services.New(repository.New(nil)), Health: &recordingHealth{}, Syncers: map[string]Syncer{"fax": {}}}); err == nil {
		t.Fatal("unknown freshness source accepted")
	}
}

// TestBuilderHook is the documented way a later chunk adds a tool (doc.go):
// a builder, an allowlist name and, if it reads a synced source, a Syncer.
// The package globals are restored afterwards, so this test is not parallel.
func TestBuilderHook(t *testing.T) {
	oldBuilders, oldAllow := builders, allowlist
	t.Cleanup(func() { builders, allowlist = oldBuilders, oldAllow })
	builders = append(append([]func(*Catalogue) []ToolSpec{}, oldBuilders...), func(*Catalogue) []ToolSpec {
		return []ToolSpec{{Name: "list_events", Description: "Events for a day.", Sources: []string{SourceCalendar},
			Run: func(context.Context, json.RawMessage) (any, error) { return map[string]any{"events": []string{}}, nil }}}
	})
	allowlist = append(append([]string{}, oldAllow...), "list_events")
	synced := 0
	cat, err := New(Deps{Service: services.New(repository.New(nil)), Health: &recordingHealth{}, Syncers: map[string]Syncer{SourceCalendar: {
		Connected: func() bool { return true },
		LastSync:  func() (time.Time, error) { return time.Time{}, errors.New("unused") },
		Sync:      func(context.Context) error { synced++; return nil },
	}}})
	if err != nil {
		t.Fatal(err)
	}
	names := cat.Names()
	if names[len(names)-1] != "list_events" {
		t.Fatalf("new tool not offered last: %v", names)
	}
	for _, tool := range cat.Tools() {
		if tool.Name == "list_events" {
			out, err := tool.Run(context.Background(), nil)
			if err != nil || out != `{"events":[]}` {
				t.Fatalf("%q %v", out, err)
			}
		}
	}
	cat.Wait()
}

func TestTwoBuildsAreIdentical(t *testing.T) {
	def := func() string {
		var b strings.Builder
		for _, tool := range newBare(t).Tools() {
			j, err := json.Marshal(map[string]any{"n": tool.Name, "d": tool.Description, "p": tool.Properties, "r": tool.Required})
			if err != nil {
				t.Fatal(err)
			}
			b.Write(j)
		}
		return b.String()
	}
	if a, b := def(), def(); a != b {
		t.Fatal("tool definitions differ between two builds (a map-ordered schema?)")
	}
}
