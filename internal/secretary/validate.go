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

// validateCount accepts count as a JSON integer from 1 to 5.
func validateCount(input json.RawMessage) error {
	if noInput(input) {
		return nil
	}
	var in struct {
		Count *int `json:"count"`
	}
	if json.Unmarshal(input, &in) != nil {
		return errInvalidInput
	}
	if in.Count != nil && (*in.Count < 1 || *in.Count > 5) {
		return errInvalidInput
	}
	return nil
}

// validateWeeks accepts weeks as 4, 12 or 26.
func validateWeeks(input json.RawMessage) error {
	if noInput(input) {
		return nil
	}
	var in struct {
		Weeks *int `json:"weeks"`
	}
	if json.Unmarshal(input, &in) != nil {
		return errInvalidInput
	}
	if in.Weeks != nil && *in.Weeks != 4 && *in.Weeks != 12 && *in.Weeks != 26 {
		return errInvalidInput
	}
	return nil
}
