package repository

import "errors"

// ErrProjectNotActive and ErrProofRequired mark completions the rules refuse.
// They are the user's to fix, not server faults, so handlers answer them with
// a readable page instead of a 500.
var (
	ErrProjectNotActive = errors.New("project is not active")
	ErrProofRequired    = errors.New("proof required")
	// ErrProjectParentCycle marks a parent change that would put a project
	// inside itself (directly or through its own sub-projects).
	ErrProjectParentCycle = errors.New("project parent would form a cycle")
)
