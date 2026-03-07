package main

import "fmt"

// LintError represents a single lint finding.
type LintError struct {
	File    string
	Line    int
	Message string
	Fixable bool
}

func (e LintError) String() string {
	return fmt.Sprintf("%s:%d: error: %s", e.File, e.Line, e.Message)
}

// LintResult collects all errors from a lint run and whether fixes were applied.
type LintResult struct {
	Errors    []LintError
	Fixed     bool
	FixedMsgs []string // describes what was fixed (populated when Verbose+Fix)
}

// HasErrors returns true if any lint errors were found.
func (r LintResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Merge appends errors from another result.
func (r *LintResult) Merge(other LintResult) {
	r.Errors = append(r.Errors, other.Errors...)
	r.FixedMsgs = append(r.FixedMsgs, other.FixedMsgs...)
	if other.Fixed {
		r.Fixed = true
	}
}

// LintOptions controls lint behavior.
type LintOptions struct {
	Fix           bool
	Verbose       bool
	MaxLineLength int
}
