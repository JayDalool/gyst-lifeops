package secretary

import (
	"bytes"
	"encoding/json"
)

// Validators for the Hevy-backed tools on the Secretary's path (GYST-164, c1).
// They run before the freshness check (spec.go), so malformed input starts no
// sync and writes nothing. The Coach does not use them: its Run clamps instead.
//
// Each one decodes with the same encoding/json rules the tool's Run uses, into
// the same shape, so an accepted value is exactly the value Run reads: no input
// is accepted and then silently replaced by a default. The default applies only
// to an absent or null field. The error is the fixed errInvalidInput and never
// echoes the input.

// noInput reports whether input is empty or null, which every tool reads as
// "no input" (see decode).
func noInput(input json.RawMessage) bool {
	t := bytes.TrimSpace(input)
	return len(t) == 0 || bytes.Equal(t, []byte("null"))
}

// validateObject accepts no input or a JSON object; unknown keys are ignored,
// as decode does.
func validateObject(input json.RawMessage) error {
	if noInput(input) {
		return nil
	}
	var in struct{}
	if json.Unmarshal(input, &in) != nil {
		return errInvalidInput
	}
	return nil
}

// The count and weeks validators decode each field both as Run does (a plain
// int: the last duplicate key wins and a null leaves the earlier value alone)
// and as a pointer (null clears it). A pointer alone would read
// {"count":99,"count":null} as omitted while Run keeps 99 (GYST-164), so a
// stale copy could be refreshed on input the tool then clamps. The two readings
// must agree: if the pointer is set, its value must be in range; if it is nil
// (omitted, or null last), Run's value must still be zero, so Run uses its
// default for exactly the inputs that were validated as omitted.

// validateCount accepts count as a JSON integer from 1 to 5.
func validateCount(input json.RawMessage) error {
	if noInput(input) {
		return nil
	}
	var plain struct {
		Count int `json:"count"`
	}
	var ptr struct {
		Count *int `json:"count"`
	}
	if json.Unmarshal(input, &plain) != nil || json.Unmarshal(input, &ptr) != nil {
		return errInvalidInput
	}
	if ptr.Count == nil {
		if plain.Count != 0 {
			return errInvalidInput
		}
		return nil
	}
	if *ptr.Count < 1 || *ptr.Count > 5 {
		return errInvalidInput
	}
	return nil
}

// validateWeeks accepts weeks as 4, 12 or 26.
func validateWeeks(input json.RawMessage) error {
	if noInput(input) {
		return nil
	}
	var plain struct {
		Weeks int `json:"weeks"`
	}
	var ptr struct {
		Weeks *int `json:"weeks"`
	}
	if json.Unmarshal(input, &plain) != nil || json.Unmarshal(input, &ptr) != nil {
		return errInvalidInput
	}
	if ptr.Weeks == nil {
		if plain.Weeks != 0 {
			return errInvalidInput
		}
		return nil
	}
	if w := *ptr.Weeks; w != 4 && w != 12 && w != 26 {
		return errInvalidInput
	}
	return nil
}
