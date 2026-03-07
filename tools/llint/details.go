package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Special options that must be flush-left after the main variable block.
var specialOptions = map[string]bool{
	"PSAFE":                  true,
	"GARBAGE":                true,
	"ARCHIVE":                true,
	"KEEP_SOURCE":            true,
	"USE_WRAPPERS":           true,
	"COMPRESS_MANPAGES":      true,
	"KEEP_OBSOLETE_LIBS":     true,
	"LUNAR_RESTART_SERVICES": true,
	"LDD_CHECK":              true,
	"FUZZY":                  true,
}

// Required fields that must appear in every DETAILS file.
var requiredFields = []string{
	"MODULE", "VERSION", "SOURCE", "WEB_SITE", "ENTERED", "UPDATED", "SHORT",
}

// varAssignRe matches a variable assignment line (with optional leading whitespace).
var varAssignRe = regexp.MustCompile(`^\s*([A-Z_][A-Z_0-9]*(?:\[[0-9]+\])?)=(.*)$`)

// detailsLine classifies a single line from a DETAILS file.
type detailsLine struct {
	raw       string
	lineNum   int
	kind      lineKind
	varName   string // set for assignment lines
	varValue  string // set for assignment lines
	isSpecial bool   // true if varName is a special option
}

type lineKind int

const (
	kindBlank lineKind = iota
	kindComment
	kindAssignment
	kindHeredocStart
	kindHeredocBody
	kindHeredocEnd
	kindOther
)

// parseDetailsLines classifies all lines in a DETAILS file.
func parseDetailsLines(content string) []detailsLine {
	rawLines := strings.Split(content, "\n")
	lines := make([]detailsLine, 0, len(rawLines))
	inHeredoc := false

	for i, raw := range rawLines {
		dl := detailsLine{raw: raw, lineNum: i + 1}

		if inHeredoc {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "EOF" {
				dl.kind = kindHeredocEnd
				inHeredoc = false
			} else {
				dl.kind = kindHeredocBody
			}
			lines = append(lines, dl)
			continue
		}

		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			dl.kind = kindBlank
		} else if strings.HasPrefix(trimmed, "#") {
			dl.kind = kindComment
		} else if strings.HasPrefix(trimmed, "cat <<") || strings.HasPrefix(trimmed, "cat<<") {
			dl.kind = kindHeredocStart
			inHeredoc = true
		} else if m := varAssignRe.FindStringSubmatch(raw); m != nil {
			dl.kind = kindAssignment
			dl.varName = m[1]
			dl.varValue = m[2]
			// Strip array index for special option check (SOURCE_URL[0] → SOURCE_URL)
			baseName := dl.varName
			if idx := strings.Index(baseName, "["); idx >= 0 {
				baseName = baseName[:idx]
			}
			dl.isSpecial = specialOptions[baseName]
		} else {
			dl.kind = kindOther
		}

		lines = append(lines, dl)
	}

	return lines
}

// LintDetails checks a DETAILS file for formatting issues and optionally fixes them.
func LintDetails(filePath string, opts LintOptions) LintResult {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return LintResult{Errors: []LintError{{File: filePath, Line: 0, Message: err.Error()}}}
	}

	content := string(data)
	lines := parseDetailsLines(content)
	file := baseFileName(filePath)

	var result LintResult

	// Check required fields
	result.Errors = append(result.Errors, checkRequiredFields(file, lines)...)

	// Check alignment
	result.Errors = append(result.Errors, checkAlignment(file, lines)...)

	// Check special option placement
	result.Errors = append(result.Errors, checkSpecialOptions(file, lines)...)

	// Check heredoc line lengths
	result.Errors = append(result.Errors, checkHeredocLength(file, lines, opts.MaxLineLength)...)

	if opts.Fix && result.HasErrors() {
		fixed := fixDetails(lines, opts.MaxLineLength)
		if err := os.WriteFile(filePath, []byte(fixed), 0644); err != nil {
			result.Errors = append(result.Errors, LintError{
				File: file, Line: 0, Message: fmt.Sprintf("failed to write fix: %v", err),
			})
			return result
		}
		result.Fixed = true

		// Re-lint the fixed file to report only remaining (unfixable) errors
		fixedLines := parseDetailsLines(fixed)
		var remaining LintResult
		remaining.Fixed = true
		remaining.Errors = append(remaining.Errors, checkRequiredFields(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkAlignment(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkSpecialOptions(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkHeredocLength(file, fixedLines, opts.MaxLineLength)...)
		return remaining
	}

	return result
}

func baseFileName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return path
}

// checkRequiredFields reports missing required fields.
func checkRequiredFields(file string, lines []detailsLine) []LintError {
	found := make(map[string]bool)
	for _, dl := range lines {
		if dl.kind == kindAssignment {
			found[dl.varName] = true
		}
	}

	var errs []LintError
	for _, field := range requiredFields {
		if !found[field] {
			errs = append(errs, LintError{
				File: file, Line: 1, Message: fmt.Sprintf("missing required field: %s", field),
			})
		}
	}
	return errs
}

// checkAlignment verifies that all non-special assignment `=` signs are vertically aligned.
func checkAlignment(file string, lines []detailsLine) []LintError {
	// Find the most common `=` column to detect the intended alignment
	colCounts := make(map[int]int)
	for _, dl := range lines {
		if dl.kind != kindAssignment || dl.isSpecial {
			continue
		}
		col := strings.Index(dl.raw, "=")
		colCounts[col]++
	}

	if len(colCounts) == 0 {
		return nil
	}

	// Find the most common column (the "intended" alignment)
	expectedCol := 0
	maxCount := 0
	for col, count := range colCounts {
		if count > maxCount || (count == maxCount && col > expectedCol) {
			expectedCol = col
			maxCount = count
		}
	}

	var errs []LintError
	for _, dl := range lines {
		if dl.kind != kindAssignment || dl.isSpecial {
			continue
		}
		actualCol := strings.Index(dl.raw, "=")
		if actualCol != expectedCol {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("'=' not aligned (expected column %d, found %d)", expectedCol, actualCol),
				Fixable: true,
			})
		}
	}
	return errs
}

// fixAlignColumn calculates the `=` column for the fixer based on the longest variable name.
func fixAlignColumn(lines []detailsLine) int {
	maxLen := 0
	for _, dl := range lines {
		if dl.kind == kindAssignment && !dl.isSpecial {
			if len(dl.varName) > maxLen {
				maxLen = len(dl.varName)
			}
		}
	}
	return maxLen
}

// checkSpecialOptions reports special options that are in the wrong position or not flush-left.
func checkSpecialOptions(file string, lines []detailsLine) []LintError {
	var errs []LintError

	// Find where the main block ends (last non-special assignment before heredoc/EOF)
	lastMainAssign := -1
	heredocStart := -1
	for i, dl := range lines {
		if dl.kind == kindAssignment && !dl.isSpecial {
			lastMainAssign = i
		}
		if dl.kind == kindHeredocStart {
			heredocStart = i
			break
		}
	}

	for _, dl := range lines {
		if dl.kind != kindAssignment || !dl.isSpecial {
			continue
		}

		// Check flush-left: the line should start with the variable name directly (no whitespace)
		if strings.TrimLeft(dl.raw, " \t") != dl.raw {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("special option %s must be flush-left (no leading whitespace)", dl.varName),
				Fixable: true,
			})
		}

		// Check position: special options must be after main block, before heredoc
		idx := dl.lineNum - 1 // convert to 0-based
		if lastMainAssign >= 0 && idx <= lastMainAssign {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("special option %s must be placed after the main variable block", dl.varName),
				Fixable: true,
			})
		}
		if heredocStart >= 0 && idx > heredocStart {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("special option %s must be placed before the heredoc", dl.varName),
				Fixable: true,
			})
		}
	}

	return errs
}

// checkHeredocLength reports heredoc lines exceeding max length.
func checkHeredocLength(file string, lines []detailsLine, maxLen int) []LintError {
	if maxLen <= 0 {
		return nil
	}

	var errs []LintError
	for _, dl := range lines {
		if dl.kind == kindHeredocBody && len(dl.raw) > maxLen {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("heredoc line exceeds %d characters (%d)", maxLen, len(dl.raw)),
				Fixable: true,
			})
		}
	}
	return errs
}

// fixDetails produces a corrected version of the DETAILS file content.
func fixDetails(lines []detailsLine, maxLineLen int) string {
	alignCol := fixAlignColumn(lines)

	// Separate lines into regions
	var mainAssigns []detailsLine
	var specialAssigns []detailsLine
	var beforeMain []detailsLine   // comments/blanks before first assignment
	var betweenLines []detailsLine // non-assignment lines between assignments
	var heredocAndAfter []detailsLine

	inHeredocRegion := false
	firstAssignSeen := false

	for _, dl := range lines {
		if inHeredocRegion || dl.kind == kindHeredocStart {
			inHeredocRegion = true
			heredocAndAfter = append(heredocAndAfter, dl)
			continue
		}

		if dl.kind == kindAssignment {
			firstAssignSeen = true
			if dl.isSpecial {
				specialAssigns = append(specialAssigns, dl)
			} else {
				// Flush any between-lines before this assignment
				mainAssigns = append(mainAssigns, betweenLines...)
				betweenLines = nil
				mainAssigns = append(mainAssigns, dl)
			}
		} else if !firstAssignSeen {
			beforeMain = append(beforeMain, dl)
		} else {
			betweenLines = append(betweenLines, dl)
		}
	}

	var out strings.Builder

	// Write pre-main lines
	for _, dl := range beforeMain {
		out.WriteString(dl.raw)
		out.WriteByte('\n')
	}

	// Write main assignments with corrected alignment
	for _, dl := range mainAssigns {
		if dl.kind == kindAssignment && !dl.isSpecial {
			out.WriteString(formatAssignment(dl.varName, dl.varValue, alignCol))
		} else {
			out.WriteString(dl.raw)
		}
		out.WriteByte('\n')
	}

	// Write any remaining between-lines (blanks between main block and heredoc)
	for _, dl := range betweenLines {
		out.WriteString(dl.raw)
		out.WriteByte('\n')
	}

	// Write special options flush-left
	for _, dl := range specialAssigns {
		out.WriteString(fmt.Sprintf("%s=%s", dl.varName, dl.varValue))
		out.WriteByte('\n')
	}

	// Write heredoc with line wrapping
	for _, dl := range heredocAndAfter {
		if dl.kind == kindHeredocBody && maxLineLen > 0 && len(dl.raw) > maxLineLen {
			wrapped := wrapLine(dl.raw, maxLineLen)
			for _, wl := range wrapped {
				out.WriteString(wl)
				out.WriteByte('\n')
			}
		} else {
			out.WriteString(dl.raw)
			out.WriteByte('\n')
		}
	}

	result := out.String()
	// Remove trailing newline added by the loop if original didn't have one
	if len(lines) > 0 && lines[len(lines)-1].raw == "" {
		// Original ended with empty line — keep trailing newline
	} else {
		result = strings.TrimRight(result, "\n")
		result += "\n"
	}

	return result
}

// formatAssignment right-pads the variable name so `=` lands at alignCol.
func formatAssignment(varName, value string, alignCol int) string {
	padding := alignCol - len(varName)
	if padding < 0 {
		padding = 0
	}
	return fmt.Sprintf("%s%s=%s", strings.Repeat(" ", padding), varName, value)
}

// wrapLine wraps a line at word boundaries to fit within maxLen.
func wrapLine(line string, maxLen int) []string {
	if len(line) <= maxLen {
		return []string{line}
	}

	var result []string
	words := strings.Fields(line)
	var current strings.Builder

	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
		} else if current.Len()+1+len(word) <= maxLen {
			current.WriteByte(' ')
			current.WriteString(word)
		} else {
			result = append(result, current.String())
			current.Reset()
			current.WriteString(word)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
