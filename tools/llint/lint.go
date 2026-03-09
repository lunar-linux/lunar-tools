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

// WarnString formats the error as a warning message.
func (e LintError) WarnString() string {
	return fmt.Sprintf("%s:%d: warning: %s", e.File, e.Line, e.Message)
}

// LintResult collects all errors and warnings from a lint run and whether fixes were applied.
type LintResult struct {
	Errors    []LintError
	Warnings  []LintError
	Fixed     bool
	FixedMsgs []string // describes what was fixed (populated when Verbose+Fix)
}

// HasErrors returns true if any lint errors were found.
func (r LintResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if any lint warnings were found.
func (r LintResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Merge appends errors and warnings from another result.
func (r *LintResult) Merge(other LintResult) {
	r.Errors = append(r.Errors, other.Errors...)
	r.Warnings = append(r.Warnings, other.Warnings...)
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
