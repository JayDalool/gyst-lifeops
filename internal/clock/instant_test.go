package clock

import (
	"sort"
	"testing"
	"time"
)

func TestInstantRoundTrip(t *testing.T) {
	for _, nanos := range []int{0, 500000000, 123000000, 123456789} {
		in := time.Date(2026, 9, 27, 14, 3, 0, nanos, time.UTC)
		s := FormatInstant(in)
		if len(s) != 30 {
			t.Fatalf("len(%q) = %d, want 30", s, len(s))
		}
		out, err := ParseInstant(s)
		if err != nil || !out.Equal(in) || out.Location() != time.UTC {
			t.Fatalf("round trip %q: %v %v", s, out, err)
		}
	}
}

func TestFormatInstantConvertsToUTC(t *testing.T) {
	got := FormatInstant(time.Date(2026, 9, 27, 9, 3, 0, 0, time.FixedZone("x", -5*3600)))
	if want := "2026-09-27T14:03:00.000000000Z"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseInstantOffsets(t *testing.T) {
	want, err := ParseInstant("2026-09-27T14:03:00Z")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"2026-09-27T09:03:00-05:00",
		"2026-09-27T19:33:00+05:30",
		"2026-09-27T14:03:00.000Z",
		"2026-09-27T14:03:00.000000000Z",
	} {
		got, err := ParseInstant(s)
		if err != nil || !got.Equal(want) || got.Location() != time.UTC {
			t.Errorf("%q: got %v err %v", s, got, err)
		}
	}
	for _, s := range []string{"2026-09-27T14:03:00.1Z", "2026-09-27T14:03:00.123456789Z"} {
		if _, err := ParseInstant(s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
}

func TestParseInstantRejects(t *testing.T) {
	for _, s := range []string{"2026-09-27T14:03:00", "2026-09-27 14:03:00", "2026-09-27 14:03:00Z", "2026-09-27", ""} {
		if _, err := ParseInstant(s); err == nil {
			t.Errorf("%q accepted", s)
		}
	}
}

func TestFormatInstantSortsAsText(t *testing.T) {
	base := time.Date(2026, 9, 27, 14, 3, 0, 0, time.UTC)
	ordered := []time.Time{
		time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1999, 12, 31, 23, 59, 59, 999999999, time.UTC),
		base,
		base.Add(1),
		base.Add(500 * time.Millisecond),
		base.Add(time.Second),
		base.Add(1230 * time.Millisecond),
		time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC),
	}
	strs := make([]string, len(ordered))
	for i, v := range ordered {
		strs[i] = FormatInstant(v)
	}
	if !sort.StringsAreSorted(strs) {
		t.Fatalf("not sorted: %v", strs)
	}
	for i := 1; i < len(strs); i++ {
		if strs[i-1] >= strs[i] {
			t.Fatalf("%q !< %q", strs[i-1], strs[i])
		}
	}
}

func TestInLocal(t *testing.T) {
	c := Clock{}
	cases := []struct {
		in, want string
	}{
		{"2026-01-15T18:00:00Z", "2026-01-15 12:00 CST"},
		{"2026-07-15T18:00:00Z", "2026-07-15 13:00 CDT"},
		{"2026-09-28T03:30:00Z", "2026-09-27 22:30 CDT"},
	}
	for _, tc := range cases {
		in, err := ParseInstant(tc.in)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.InLocal(in).Format("2006-01-02 15:04 MST"); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.in, got, tc.want)
		}
	}
}
