package secretary

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Untrusted text (an event title, a mail subject, a project name) is written
// by someone else and can try to instruct the model. Every such string leaves
// this package inside an Envelope that says so, after Clean has removed the
// characters that hide text from a human reader (09 B.8, brain section 10).

// Caps, in runes, for untrusted fields. Later chunks use these so a cap is
// decided once.
const (
	TitleCap       = 80
	SubjectCap     = 120
	NameCap        = 80
	DescriptionCap = 2048
)

// Envelope is how untrusted text reaches the model.
type Envelope struct {
	Untrusted bool   `json:"untrusted"`
	Source    string `json:"source"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
}

// ErrUnknownSource is returned for a source other than email or calendar.
var ErrUnknownSource = errors.New("unknown source")

// Wrap cleans s, caps it at max runes and wraps it. Only "email" and
// "calendar" are sources of untrusted text; anything else is a bug in the
// caller and is rejected, never wrapped under a made-up label.
func Wrap(source, s string, max int) (Envelope, error) {
	if source != "email" && source != "calendar" {
		return Envelope{}, ErrUnknownSource
	}
	text, truncated := Clean(s, max)
	return Envelope{Untrusted: true, Source: source, Text: text, Truncated: truncated}, nil
}

// hidden reports characters that are invisible or reorder text but are not
// whitespace: every Cf (format) code point, which covers the soft hyphen,
// U+061C, U+180E, zero-width and bidi controls, the word joiner and
// invisible operators, BOM, interlinear annotation marks and tag characters;
// plus variation selectors, the combining grapheme joiner, and the Hangul
// fillers, which render as blanks. unicode.Cf is used as a whole category so a
// code point added by a later Unicode version is covered without a list.
func hidden(r rune) bool {
	switch {
	case unicode.Is(unicode.Cf, r):
		return true
	case r >= 0xFE00 && r <= 0xFE0F, r >= 0xE0100 && r <= 0xE01EF: // variation selectors
		return true
	case r == 0x180E, r == 0x034F, r == 0x115F, r == 0x1160, r == 0x3164, r == 0xFFA0:
		return true
	}
	return false
}

// Clean removes control and hidden characters, turns every kind of whitespace
// (tab, CR, LF, VT, FF, NEL, U+2028, U+2029, NBSP, ...) into one space, trims,
// drops invalid UTF-8, and caps the result at max runes. When it cuts, the
// last rune is an ellipsis and truncated is true. A max below 1 gives "".
func Clean(s string, max int) (text string, truncated bool) {
	var b strings.Builder
	space := true // true at the start so leading whitespace is dropped
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r == utf8.RuneError && size == 1 {
			continue // invalid byte
		}
		switch {
		case hidden(r):
			continue
		case unicode.IsSpace(r) || r == '\v' || r == '\f' || r == 0x85:
			if !space {
				b.WriteByte(' ')
				space = true
			}
		case unicode.Is(unicode.Cc, r):
			continue
		default:
			b.WriteRune(r)
			space = false
		}
	}
	text = strings.TrimRight(b.String(), " ")
	if max < 1 {
		return "", text != ""
	}
	if utf8.RuneCountInString(text) <= max {
		return text, false
	}
	runes := []rune(text)[:max-1]
	return strings.TrimRight(string(runes), " ") + "…", true
}
