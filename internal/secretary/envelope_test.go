package secretary

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestCleanStrips(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"C0 controls dropped", "a\x00b\x01c\x07d\x1be", "abcde"},
		{"DEL dropped", "a\x7fb", "ab"},
		{"C1 controls dropped", "a\u0080b\u009fc", "abc"},
		{"tab CR LF VT FF NEL become spaces", "a\tb\rc\nd\ve\ff\u0085g", "a b c d e f g"},
		{"soft hyphen", "co\u00ADop", "coop"},
		{"Arabic letter mark U+061C", "a\u061Cb", "ab"},
		{"Mongolian vowel separator U+180E", "a\u180Eb", "ab"},
		{"zero width and bidi marks U+200B-200F", "a\u200Bb\u200Cc\u200Dd\u200Ee\u200Ff", "abcdef"},
		{"bidi embeddings U+202A-202E", "a\u202Ab\u202Bc\u202Cd\u202De\u202Ef", "abcdef"},
		{"word joiner and invisible operators U+2060-2064", "a\u2060b\u2061c\u2062d\u2063e\u2064f", "abcdef"},
		{"bidi isolates U+2066-206F", "a\u2066b\u2067c\u2068d\u2069e\u206Ff", "abcdef"},
		{"BOM", "\xEF\xBB\xBFabc", "abc"},
		{"interlinear annotation U+FFF9-FFFB", "a\uFFF9b\uFFFAc\uFFFBd", "abcd"},
		{"tag characters", "a\U000E0001b\U000E0041c\U000E007Fd", "abcd"},
		{"a Cf outside every listed range (Arabic number sign U+0600)", "a\u0600b", "ab"},
		{"variation selectors", "a\uFE0Fb\uFE00c\U000E0100d\U000E01EFe", "abcde"},
		{"combining grapheme joiner", "a\u034Fb", "ab"},
		{"Hangul fillers", "a\u115Fb\u1160c\u3164d\uFFA0e", "abcde"},
		{"LS and PS become one space", "a\u2028b\u2029c", "a b c"},
		{"NBSP and ideographic space collapse", "a\u00A0\u00A0b\u3000c", "a b c"},
		{"whitespace collapses and trims", "  a \t\n  b  ", "a b"},
		{"invalid UTF-8 dropped", "a\xffb\xc3c", "abc"},
		{"a real U+FFFD is kept", "a\uFFFDb", "a\uFFFDb"},
		{"plain text untouched", "Dentist <2 pm> & lunch 日本語 é", "Dentist <2 pm> & lunch 日本語 é"},
	} {
		got, trunc := Clean(tc.in, 200)
		if got != tc.want || trunc {
			t.Errorf("%s: Clean(%q) = %q, %v; want %q", tc.name, tc.in, got, trunc, tc.want)
		}
	}
}

// Every code point of the Unicode Cf category goes, not only the ranges the
// plan listed, so a later Unicode version cannot add a new hidden character.
func TestCleanStripsAllOfCf(t *testing.T) {
	n := 0
	for _, r16 := range unicode.Cf.R16 {
		for r := rune(r16.Lo); r <= rune(r16.Hi); r += rune(r16.Stride) {
			n++
			if got, _ := Clean("a"+string(r)+"b", 10); got != "ab" {
				t.Fatalf("U+%04X survived: %q", r, got)
			}
		}
	}
	for _, r32 := range unicode.Cf.R32 {
		for r := rune(r32.Lo); r <= rune(r32.Hi); r += rune(r32.Stride) {
			n++
			if got, _ := Clean("a"+string(r)+"b", 10); got != "ab" {
				t.Fatalf("U+%04X survived: %q", r, got)
			}
		}
	}
	if n < 100 {
		t.Fatalf("only %d Cf code points checked", n)
	}
}

func TestCleanCaps(t *testing.T) {
	got, trunc := Clean(strings.Repeat("é", 10), 10)
	if got != strings.Repeat("é", 10) || trunc {
		t.Fatalf("exact fit: %q %v", got, trunc)
	}
	got, trunc = Clean(strings.Repeat("é", 11), 10)
	if got != strings.Repeat("é", 9)+"…" || !trunc || utf8.RuneCountInString(got) != 10 {
		t.Fatalf("cut: %q %v", got, trunc)
	}
	// Four-byte runes count as one; the cap is in runes, not bytes.
	got, _ = Clean(strings.Repeat("😀", 200), 80)
	if utf8.RuneCountInString(got) != 80 || !strings.HasSuffix(got, "…") {
		t.Fatalf("emoji cap: %d runes", utf8.RuneCountInString(got))
	}
	// A cut that lands after a space does not leave a space before the ellipsis.
	got, _ = Clean("abcd efgh", 6)
	if got != "abcd…" {
		t.Fatalf("space before ellipsis: %q", got)
	}
	if got, trunc := Clean("abc", 0); got != "" || !trunc {
		t.Fatalf("zero cap: %q %v", got, trunc)
	}
	if got, trunc := Clean("", 5); got != "" || trunc {
		t.Fatalf("empty: %q %v", got, trunc)
	}
	if TitleCap != 80 || SubjectCap != 120 || NameCap != 80 || DescriptionCap != 2048 {
		t.Fatal("exported caps changed")
	}
}

func TestWrap(t *testing.T) {
	for _, src := range []string{"email", "calendar"} {
		e, err := Wrap(src, "  Hi\u200B there  ", 80)
		if err != nil || !e.Untrusted || e.Source != src || e.Text != "Hi there" || e.Truncated {
			t.Fatalf("%s: %+v %v", src, e, err)
		}
	}
	for _, src := range []string{"", "web", "Email", "system", "task", "project"} {
		if _, err := Wrap(src, "x", 80); err != ErrUnknownSource {
			t.Errorf("source %q accepted", src)
		}
	}
	e, _ := Wrap("email", strings.Repeat("a", 500), SubjectCap)
	if !e.Truncated || utf8.RuneCountInString(e.Text) != SubjectCap {
		t.Fatalf("cap: %d %v", utf8.RuneCountInString(e.Text), e.Truncated)
	}
}

func FuzzClean(f *testing.F) {
	for _, s := range []string{"", "plain", "a\u202Eb", "\xff\xfe", "x\u2028y", "\U000E0041tag", strings.Repeat("😀", 50)} {
		f.Add(s, 10)
	}
	f.Fuzz(func(t *testing.T, s string, max int) {
		max = max%200 + 1
		if max < 1 {
			max = 1
		}
		got, _ := Clean(s, max)
		if !utf8.ValidString(got) {
			t.Fatalf("invalid UTF-8 from %q", s)
		}
		if utf8.RuneCountInString(got) > max {
			t.Fatalf("%d runes over cap %d", utf8.RuneCountInString(got), max)
		}
		for _, r := range got {
			if hidden(r) || (unicode.IsControl(r)) || (unicode.IsSpace(r) && r != ' ') {
				t.Fatalf("U+%04X left in %q", r, got)
			}
		}
		if strings.Contains(got, "  ") || strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") {
			t.Fatalf("whitespace not collapsed: %q", got)
		}
	})
}
