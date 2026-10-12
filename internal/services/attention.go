package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

// Attention is computed on read from each source's own data. Nothing here is
// stored and nothing here is editable work: an item only points at the place
// where the work lives (GYST-VN-066, 05A D5).

// AttentionSeverity orders items. The zero value is not valid.
type AttentionSeverity string

const (
	AttentionInfo        AttentionSeverity = "info"
	AttentionNeedsAction AttentionSeverity = "needs_action"
	AttentionWarning     AttentionSeverity = "warning"
	AttentionCritical    AttentionSeverity = "critical"
)

// AttentionKind says whether the user must decide something or something is
// broken or late.
type AttentionKind string

const (
	AttentionDecision AttentionKind = "decision"
	AttentionProblem  AttentionKind = "problem"
)

// AttentionItem is one thing that wants the user's attention.
type AttentionItem struct {
	Source    string
	SourceKey string
	Kind      AttentionKind
	Severity  AttentionSeverity
	// Title is plain text. Context is a short plain-text reason.
	Title   string
	Context string
	// Target is a rooted internal path ("/tasks/3") or empty.
	Target string
	// FirstSeen is the instant the item became true, in UTC.
	FirstSeen time.Time
	// Fingerprint changes when the item's state changes and not when its
	// title is renamed, so a rename does not re-fire a notification.
	Fingerprint string
}

// ID is the stable identifier of an item: the first 16 hex characters of
// sha256(Source + NUL + SourceKey).
func (i AttentionItem) ID() string {
	sum := sha256.Sum256([]byte(i.Source + "\x00" + i.SourceKey))
	return hex.EncodeToString(sum[:])[:16]
}

// AttentionFingerprint is sha256 hex of source|key|severity|state. Titles are
// deliberately not part of the state.
func AttentionFingerprint(source, key string, severity AttentionSeverity, state string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", source, key, severity, state)))
	return hex.EncodeToString(sum[:])
}

// AttentionProducer is one source of attention items. Adding a source needs
// no change to the service: pass it to ListAttention.
type AttentionProducer interface {
	Source() string
	Label() string
	Produce(ctx context.Context, now time.Time) ([]AttentionItem, error)
}

// AttentionList is the merged result. Unavailable holds the Label of every
// producer that failed: a failure never means healthy.
type AttentionList struct {
	Items       []AttentionItem
	Unavailable []string
}

// Surface returns the items that are not merely informational.
func (l AttentionList) Surface() []AttentionItem {
	var out []AttentionItem
	for _, item := range l.Items {
		if item.Severity != AttentionInfo {
			out = append(out, item)
		}
	}
	return out
}

// BadgeCount counts needs_action plus critical items.
func (l AttentionList) BadgeCount() int {
	n := 0
	for _, item := range l.Items {
		if item.Severity == AttentionNeedsAction || item.Severity == AttentionCritical {
			n++
		}
	}
	return n
}

var attentionRank = map[AttentionSeverity]int{
	AttentionCritical:    0,
	AttentionNeedsAction: 1,
	AttentionWarning:     2,
	AttentionInfo:        3,
}

func attentionRankOf(s AttentionSeverity) int {
	if r, ok := attentionRank[s]; ok {
		return r
	}
	return len(attentionRank)
}

// runProducer isolates one producer: an error or a panic is reported, never
// propagated.
func runProducer(ctx context.Context, p AttentionProducer, now time.Time) (items []AttentionItem, ok bool) {
	defer func() {
		if recover() != nil {
			items, ok = nil, false
		}
	}()
	got, err := p.Produce(ctx, now)
	if err != nil {
		return nil, false
	}
	return got, true
}

// ListAttention merges the producers' items. It is pure apart from what the
// producers read. Order: severity (critical first), FirstSeen ascending, then
// Source and SourceKey. A repeated (Source, SourceKey) keeps the first item.
func ListAttention(ctx context.Context, now time.Time, producers ...AttentionProducer) AttentionList {
	var list AttentionList
	seen := map[[2]string]bool{}
	for _, p := range producers {
		items, ok := runProducer(ctx, p, now)
		if !ok {
			list.Unavailable = append(list.Unavailable, safeLabel(p))
			continue
		}
		for _, item := range items {
			k := [2]string{item.Source, item.SourceKey}
			if seen[k] {
				continue
			}
			seen[k] = true
			item.FirstSeen = item.FirstSeen.UTC()
			list.Items = append(list.Items, item)
		}
	}
	sort.SliceStable(list.Items, func(i, j int) bool {
		a, b := list.Items[i], list.Items[j]
		if ra, rb := attentionRankOf(a.Severity), attentionRankOf(b.Severity); ra != rb {
			return ra < rb
		}
		if !a.FirstSeen.Equal(b.FirstSeen) {
			return a.FirstSeen.Before(b.FirstSeen)
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.SourceKey < b.SourceKey
	})
	return list
}

// safeLabel keeps a panicking Label from failing the caller.
func safeLabel(p AttentionProducer) (label string) {
	defer func() {
		if recover() != nil {
			label = "Unknown source"
		}
	}()
	return p.Label()
}

// Attention lists everything wanting attention right now.
func (s *Service) Attention(ctx context.Context) AttentionList {
	return ListAttention(ctx, s.Now().UTC(), s.AttentionProducers...)
}
