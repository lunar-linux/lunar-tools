package main

import (
	"fmt"
	"os"
	"strings"
)

// Allowed function calls in DEPENDS files.
var allowedFunctions = []string{
	"depends",
	"optional_depends",
	"optional_depends_requires",
	"optional_depends_one_of",
}

// Disallowed bash keywords and constructs.
var disallowedKeywords = []string{
	"if", "then", "else", "elif", "fi",
	"case", "esac",
	"for", "do", "done",
	"while", "until",
	"function",
}

// LintDepends checks a DEPENDS file for disallowed constructs.
func LintDepends(filePath string, opts LintOptions) LintResult {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return LintResult{Errors: []LintError{{File: filePath, Line: 0, Message: err.Error()}}}
	}

	file := baseFileName(filePath)
	rawLines := strings.Split(string(data), "\n")

	// Join continuation lines (lines ending with \) into logical lines,
	// tracking the starting line number of each logical line.
	type logicalLine struct {
		text    string
		lineNum int
	}

	var logical []logicalLine
	var current strings.Builder
	startLine := 0

	for i, line := range rawLines {
		if current.Len() == 0 {
			startLine = i + 1
		}

		trimmed := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmed, "\\") {
			// Continuation: strip the backslash and append
			current.WriteString(strings.TrimSuffix(trimmed, "\\"))
			current.WriteByte(' ')
		} else {
			current.WriteString(line)
			logical = append(logical, logicalLine{text: current.String(), lineNum: startLine})
			current.Reset()
		}
	}
	// Handle unterminated continuation
	if current.Len() > 0 {
		logical = append(logical, logicalLine{text: current.String(), lineNum: startLine})
	}

	var result LintResult

	for _, ll := range logical {
		trimmed := strings.TrimSpace(ll.text)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fn := matchedFunction(trimmed)
		if fn != "" {
			// depends and optional_depends_requires take plain module
			// names — command substitutions are never valid there.
			// optional_depends and optional_depends_one_of accept quoted
			// flag arguments where $(…) is legitimate (e.g. pkg-config).
			hasQuotedArgs := fn == "optional_depends" || fn == "optional_depends_one_of"
			if hasQuotedArgs {
				if hasUnquotedSubstitution(trimmed) {
					result.Errors = append(result.Errors, LintError{
						File:    file,
						Line:    ll.lineNum,
						Message: "disallowed bash logic: command substitution outside quoted string",
						Fixable: false,
					})
				}
			} else {
				if strings.Contains(trimmed, "$(") || strings.Contains(trimmed, "`") {
					result.Errors = append(result.Errors, LintError{
						File:    file,
						Line:    ll.lineNum,
						Message: "disallowed bash logic: command substitution in function call",
						Fixable: false,
					})
				}
			}
			continue
		}

		// Determine what kind of disallowed construct this is
		msg := identifyViolation(trimmed)
		result.Errors = append(result.Errors, LintError{
			File:    file,
			Line:    ll.lineNum,
			Message: msg,
			Fixable: false,
		})
	}

	return result
}

// matchedFunction returns the name of the allowed function call that the line
// starts with, or "" if the line does not match any allowed function.
func matchedFunction(line string) string {
	for _, fn := range allowedFunctions {
		if line == fn || strings.HasPrefix(line, fn+" ") || strings.HasPrefix(line, fn+"\t") {
			return fn
		}
	}
	return ""
}

// hasUnquotedSubstitution reports whether line contains $( or ` outside of
// double-quoted strings.  Single quotes are not tracked because DEPENDS
// arguments use double quotes exclusively.
func hasUnquotedSubstitution(line string) bool {
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case '$':
			if !inQuote && i+1 < len(line) && line[i+1] == '(' {
				return true
			}
		case '`':
			if !inQuote {
				return true
			}
		}
	}
	return false
}

// identifyViolation produces a descriptive error message for a disallowed line.
func identifyViolation(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "disallowed: empty statement"
	}
	// Check for bash keywords
	firstWord := fields[0]
	// Strip leading punctuation like [ or [[
	cleanWord := strings.TrimLeft(firstWord, "[")
	if cleanWord == "" {
		// Line starts with [ or [[
		if strings.HasPrefix(line, "[[") {
			return "disallowed bash logic: '[[' test expression"
		}
		return "disallowed bash logic: '[' test expression"
	}

	for _, kw := range disallowedKeywords {
		if firstWord == kw {
			return fmt.Sprintf("disallowed bash logic: '%s'", kw)
		}
	}

	// Check for command substitution
	if strings.Contains(line, "$(") || strings.Contains(line, "`") {
		return "disallowed bash logic: command substitution"
	}

	// Check for variable assignment
	if strings.Contains(firstWord, "=") && !strings.HasPrefix(firstWord, "#") {
		return fmt.Sprintf("disallowed: variable assignment '%s'", firstWord)
	}

	// Check for test brackets
	if strings.HasPrefix(line, "[") {
		return "disallowed bash logic: test expression"
	}

	return fmt.Sprintf("disallowed: '%s' (only depends, optional_depends, optional_depends_requires, optional_depends_one_of allowed)", firstWord)
}
