package main

import "testing"

func TestLintErrorString(t *testing.T) {
	e := LintError{File: "DETAILS", Line: 3, Message: "'=' not aligned"}
	got := e.String()
	want := "DETAILS:3: error: '=' not aligned"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLintResultHasErrors(t *testing.T) {
	r := LintResult{}
	if r.HasErrors() {
		t.Error("empty result should not have errors")
	}
	r.Errors = append(r.Errors, LintError{})
	if !r.HasErrors() {
		t.Error("result with errors should have errors")
	}
}

func TestLintResultMerge(t *testing.T) {
	r1 := LintResult{Errors: []LintError{{File: "A", Line: 1, Message: "a"}}}
	r2 := LintResult{Errors: []LintError{{File: "B", Line: 2, Message: "b"}}, Fixed: true}
	r1.Merge(r2)
	if len(r1.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(r1.Errors))
	}
	if !r1.Fixed {
		t.Error("merged result should have Fixed=true")
	}
}

func TestLintOptionsDefaults(t *testing.T) {
	opts := LintOptions{}
	if opts.Fix {
		t.Error("Fix should default to false")
	}
	if opts.MaxLineLength != 0 {
		t.Error("MaxLineLength zero value should be 0")
	}
}
