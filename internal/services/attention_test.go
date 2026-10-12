package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProducer struct {
	source, label string
	items         []AttentionItem
	err           error
	panics        bool
}

func (f fakeProducer) Source() string { return f.source }
func (f fakeProducer) Label() string  { return f.label }
func (f fakeProducer) Produce(context.Context, time.Time) ([]AttentionItem, error) {
	if f.panics {
		panic("boom")
	}
	return f.items, f.err
}

var attentionEpoch = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func fakeItem(source, key string, sev AttentionSeverity, hours int) AttentionItem {
	return AttentionItem{Source: source, SourceKey: key, Severity: sev, Kind: AttentionProblem,
		Title: "Item " + key, FirstSeen: attentionEpoch.Add(time.Duration(hours) * time.Hour)}
}

func keysOf(l AttentionList) []string {
	var out []string
	for _, i := range l.Items {
		out = append(out, i.Source+"/"+i.SourceKey)
	}
	return out
}

func TestListAttentionOrdering(t *testing.T) {
	p := fakeProducer{source: "fake", label: "Fake", items: []AttentionItem{
		fakeItem("fake", "info", AttentionInfo, 0),
		fakeItem("fake", "warn-new", AttentionWarning, 5),
		fakeItem("fake", "warn-old", AttentionWarning, 1),
		fakeItem("fake", "crit", AttentionCritical, 9),
		fakeItem("fake", "needs", AttentionNeedsAction, 3),
		fakeItem("fake", "tie-b", AttentionWarning, 7),
		fakeItem("fake", "tie-a", AttentionWarning, 7),
	}}
	got := keysOf(ListAttention(context.Background(), attentionEpoch, p))
	want := []string{"fake/crit", "fake/needs", "fake/warn-old", "fake/warn-new", "fake/tie-a", "fake/tie-b", "fake/info"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestListAttentionFakeProducerNeedsNoServiceChange(t *testing.T) {
	list := ListAttention(context.Background(), attentionEpoch,
		fakeProducer{source: "test.fake", label: "Fake", items: []AttentionItem{fakeItem("test.fake", "1", AttentionNeedsAction, 0)}})
	if len(list.Items) != 1 || list.Items[0].Source != "test.fake" || len(list.Unavailable) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestListAttentionProducerErrorAndPanic(t *testing.T) {
	list := ListAttention(context.Background(), attentionEpoch,
		fakeProducer{source: "bad", label: "Erroring", err: errors.New("down")},
		fakeProducer{source: "panic", label: "Panicking", panics: true},
		fakeProducer{source: "ok", label: "Fine", items: []AttentionItem{fakeItem("ok", "1", AttentionWarning, 0)}})
	if len(list.Items) != 1 || list.Items[0].Source != "ok" {
		t.Fatalf("items = %#v", list.Items)
	}
	if len(list.Unavailable) != 2 || list.Unavailable[0] != "Erroring" || list.Unavailable[1] != "Panicking" {
		t.Fatalf("unavailable = %v", list.Unavailable)
	}
}

func TestListAttentionDedupesSourceKey(t *testing.T) {
	first := fakeItem("fake", "1", AttentionWarning, 0)
	first.Title = "first"
	second := fakeItem("fake", "1", AttentionCritical, 0)
	second.Title = "second"
	list := ListAttention(context.Background(), attentionEpoch,
		fakeProducer{source: "fake", label: "Fake", items: []AttentionItem{first, second}})
	if len(list.Items) != 1 || list.Items[0].Title != "first" {
		t.Fatalf("items = %#v", list.Items)
	}
}

func TestAttentionListSurfaceAndBadgeCount(t *testing.T) {
	list := AttentionList{Items: []AttentionItem{
		fakeItem("f", "1", AttentionInfo, 0), fakeItem("f", "2", AttentionWarning, 0),
		fakeItem("f", "3", AttentionNeedsAction, 0), fakeItem("f", "4", AttentionCritical, 0),
	}}
	if got := len(list.Surface()); got != 3 {
		t.Fatalf("surface = %d", got)
	}
	if got := list.BadgeCount(); got != 2 {
		t.Fatalf("badge = %d", got)
	}
}

func TestAttentionItemIDAndFingerprint(t *testing.T) {
	a := AttentionItem{Source: "s", SourceKey: "k"}
	if len(a.ID()) != 16 || a.ID() != (AttentionItem{Source: "s", SourceKey: "k", Title: "renamed"}).ID() {
		t.Fatalf("id = %q", a.ID())
	}
	if a.ID() == (AttentionItem{Source: "sk", SourceKey: ""}).ID() {
		t.Fatal("id must separate source and key")
	}
	base := AttentionFingerprint("s", "k", AttentionWarning, "2026-10-01")
	if base != AttentionFingerprint("s", "k", AttentionWarning, "2026-10-01") {
		t.Fatal("fingerprint not stable")
	}
	if base == AttentionFingerprint("s", "k", AttentionWarning, "2026-10-02") ||
		base == AttentionFingerprint("s", "k", AttentionCritical, "2026-10-01") {
		t.Fatal("fingerprint must change with state or severity")
	}
}
