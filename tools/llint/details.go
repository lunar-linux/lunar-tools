package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

	// Check heredoc spacing
	result.Errors = append(result.Errors, checkHeredocSpacing(file, lines)...)

	// Check heredoc trailing newlines
	result.Errors = append(result.Errors, checkHeredocTrailing(file, lines)...)

	// Check heredoc line lengths
	result.Errors = append(result.Errors, checkHeredocLength(file, lines, opts.MaxLineLength)...)

	// Check duplicate assignments
	result.Errors = append(result.Errors, checkDuplicates(file, lines)...)

	// Check date fields
	result.Errors = append(result.Errors, checkDates(file, lines)...)

	// Check MODULE matches directory name
	result.Errors = append(result.Errors, checkModuleName(file, filePath, lines)...)

	// Check SOURCE/URL pairing
	srcErrs, srcWarns := checkSourceURLPairing(file, lines)
	result.Errors = append(result.Errors, srcErrs...)
	result.Warnings = append(result.Warnings, srcWarns...)

	if opts.Fix && result.HasErrors() {
		// Save pre-fix errors for verbose reporting
		preFix := result.Errors

		fixed := fixDetails(lines, opts.MaxLineLength)
		if err := os.WriteFile(filePath, []byte(fixed), 0644); err != nil {
			result.Errors = append(result.Errors, LintError{
				File: file, Line: 0, Message: fmt.Sprintf("failed to write fix: %v", err),
			})
			return result
		}

		// Re-lint the fixed file to report only remaining (unfixable) errors
		fixedLines := parseDetailsLines(fixed)
		var remaining LintResult
		remaining.Fixed = true
		remaining.Errors = append(remaining.Errors, checkRequiredFields(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkAlignment(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkSpecialOptions(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkHeredocSpacing(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkHeredocTrailing(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkHeredocLength(file, fixedLines, opts.MaxLineLength)...)
		remaining.Errors = append(remaining.Errors, checkDuplicates(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkDates(file, fixedLines)...)
		remaining.Errors = append(remaining.Errors, checkModuleName(file, filePath, fixedLines)...)
		fixSrcErrs, fixSrcWarns := checkSourceURLPairing(file, fixedLines)
		remaining.Errors = append(remaining.Errors, fixSrcErrs...)
		remaining.Warnings = append(remaining.Warnings, fixSrcWarns...)

		// Build verbose messages for errors that were fixed
		if opts.Verbose {
			remainSet := make(map[string]bool)
			for _, e := range remaining.Errors {
				remainSet[e.String()] = true
			}
			for _, e := range preFix {
				if !remainSet[e.String()] {
					remaining.FixedMsgs = append(remaining.FixedMsgs,
						fmt.Sprintf("%s:%d: fixed: %s", e.File, e.Line, e.Message))
				}
			}
		}

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

// checkDuplicates reports duplicate variable assignments.
// Exact duplicates (same name and value) are fixable; conflicting values are not.
func checkDuplicates(file string, lines []detailsLine) []LintError {
	type seen struct {
		value   string
		lineNum int
	}
	assignments := make(map[string][]seen)

	for _, dl := range lines {
		if dl.kind != kindAssignment {
			continue
		}
		assignments[dl.varName] = append(assignments[dl.varName], seen{value: dl.varValue, lineNum: dl.lineNum})
	}

	var errs []LintError
	for varName, occurrences := range assignments {
		if len(occurrences) < 2 {
			continue
		}
		// Check if all values are the same
		allSame := true
		for _, o := range occurrences[1:] {
			if o.value != occurrences[0].value {
				allSame = false
				break
			}
		}
		if allSame {
			// Exact duplicates — fixable, report on all but the first
			for _, o := range occurrences[1:] {
				errs = append(errs, LintError{
					File:    file,
					Line:    o.lineNum,
					Message: fmt.Sprintf("duplicate assignment: %s (same value)", varName),
					Fixable: true,
				})
			}
		} else {
			// Conflicting values — not fixable, report on all occurrences
			for _, o := range occurrences {
				errs = append(errs, LintError{
					File:    file,
					Line:    o.lineNum,
					Message: fmt.Sprintf("conflicting duplicate assignment: %s", varName),
					Fixable: false,
				})
			}
		}
	}
	return errs
}

// checkDates validates ENTERED and UPDATED fields:
// - must be valid dates in yyyymmdd format
// - must not be in the future
// - UPDATED must be >= ENTERED
func checkDates(file string, lines []detailsLine) []LintError {
	var errs []LintError
	dates := make(map[string]time.Time)
	lineNums := make(map[string]int)

	today := time.Now().Truncate(24 * time.Hour)

	for _, dl := range lines {
		if dl.kind != kindAssignment {
			continue
		}
		if dl.varName != "ENTERED" && dl.varName != "UPDATED" {
			continue
		}

		val := strings.Trim(dl.varValue, "\"'")
		t, err := time.Parse("20060102", val)
		if err != nil {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("%s has invalid date format %q (expected yyyymmdd)", dl.varName, val),
			})
			continue
		}

		if t.After(today.AddDate(0, 0, 1)) {
			errs = append(errs, LintError{
				File:    file,
				Line:    dl.lineNum,
				Message: fmt.Sprintf("%s date %s is in the future", dl.varName, val),
			})
		}

		dates[dl.varName] = t
		lineNums[dl.varName] = dl.lineNum
	}

	entered, hasEntered := dates["ENTERED"]
	updated, hasUpdated := dates["UPDATED"]
	if hasEntered && hasUpdated && updated.Before(entered) {
		errs = append(errs, LintError{
			File:    file,
			Line:    lineNums["UPDATED"],
			Message: fmt.Sprintf("UPDATED (%s) is before ENTERED (%s)",
				updated.Format("20060102"), entered.Format("20060102")),
		})
	}

	return errs
}

// checkModuleName verifies the MODULE field matches the directory name.
func checkModuleName(file string, filePath string, lines []detailsLine) []LintError {
	dirName := filepath.Base(filepath.Dir(filePath))

	for _, dl := range lines {
		if dl.kind == kindAssignment && dl.varName == "MODULE" {
			val := strings.Trim(dl.varValue, "\"'")
			if val != dirName {
				return []LintError{{
					File:    file,
					Line:    dl.lineNum,
					Message: fmt.Sprintf("MODULE value %q does not match directory name %q", val, dirName),
				}}
			}
			return nil
		}
	}
	return nil
}

// checkHeredocSpacing verifies there is exactly one blank line before `cat << EOF`.
func checkHeredocSpacing(file string, lines []detailsLine) []LintError {
	for i, dl := range lines {
		if dl.kind != kindHeredocStart {
			continue
		}
		// Count consecutive blank lines immediately before the heredoc
		blanks := 0
		for j := i - 1; j >= 0; j-- {
			if lines[j].kind == kindBlank {
				blanks++
			} else {
				break
			}
		}
		if blanks == 1 {
			return nil // correct
		}
		msg := "expected exactly one blank line before heredoc"
		if blanks == 0 {
			msg = "missing blank line before heredoc"
		} else {
			msg = fmt.Sprintf("expected 1 blank line before heredoc, found %d", blanks)
		}
		return []LintError{{
			File:    file,
			Line:    dl.lineNum,
			Message: msg,
			Fixable: true,
		}}
	}
	return nil
}

// checkHeredocTrailing verifies there are no extra blank lines after the heredoc EOF.
func checkHeredocTrailing(file string, lines []detailsLine) []LintError {
	for i, dl := range lines {
		if dl.kind != kindHeredocEnd {
			continue
		}
		// Count blank lines after EOF
		blanks := 0
		for j := i + 1; j < len(lines); j++ {
			if lines[j].kind == kindBlank {
				blanks++
			} else {
				// Non-blank content after EOF — not our concern here
				return nil
			}
		}
		if blanks > 1 {
			return []LintError{{
				File:    file,
				Line:    lines[i+1].lineNum,
				Message: fmt.Sprintf("expected 1 trailing newline after EOF, found %d blank lines", blanks),
				Fixable: true,
			}}
		}
	}
	return nil
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
	// Deduplicate exact-duplicate assignments (keep first occurrence only)
	lines = dedup(lines)

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

	// Write any non-blank between-lines (e.g. comments between main block and heredoc)
	for _, dl := range betweenLines {
		if dl.kind != kindBlank {
			out.WriteString(dl.raw)
			out.WriteByte('\n')
		}
	}

	// Write special options flush-left
	for _, dl := range specialAssigns {
		out.WriteString(fmt.Sprintf("%s=%s", dl.varName, dl.varValue))
		out.WriteByte('\n')
	}

	// Ensure exactly one blank line before heredoc
	if len(heredocAndAfter) > 0 {
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
	// Ensure file ends with exactly one trailing newline
	result = strings.TrimRight(result, "\n")
	result += "\n"

	return result
}

// dedup removes exact-duplicate assignments (same varName and varValue), keeping the first occurrence.
// Conflicting duplicates (same name, different value) are left in place — they are unfixable.
func dedup(lines []detailsLine) []detailsLine {
	type entry struct {
		values   []string
		conflict bool
	}
	seen := make(map[string]*entry)

	// First pass: detect duplicates and conflicts
	for _, dl := range lines {
		if dl.kind != kindAssignment {
			continue
		}
		if e, ok := seen[dl.varName]; ok {
			e.values = append(e.values, dl.varValue)
			if dl.varValue != e.values[0] {
				e.conflict = true
			}
		} else {
			seen[dl.varName] = &entry{values: []string{dl.varValue}}
		}
	}

	// Second pass: filter out exact duplicates (keep first), preserve conflicts
	kept := make(map[string]bool)
	result := make([]detailsLine, 0, len(lines))
	for _, dl := range lines {
		if dl.kind == kindAssignment {
			e := seen[dl.varName]
			if len(e.values) > 1 && !e.conflict {
				// Exact duplicates — keep only the first
				if kept[dl.varName] {
					continue
				}
				kept[dl.varName] = true
			}
		}
		result = append(result, dl)
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

// sourceRe matches SOURCE or SOURCE<N> variable names.
var sourceRe = regexp.MustCompile(`^SOURCE(\d*)$`)

// sourceURLRe matches SOURCE_URL, SOURCE_URL_FULL, SOURCE<N>_URL, or SOURCE<N>_URL_FULL.
var sourceURLRe = regexp.MustCompile(`^SOURCE(\d*)_(URL|URL_FULL)$`)

// sourceVFYRe matches SOURCE_VFY or SOURCE<N>_VFY.
var sourceVFYRe = regexp.MustCompile(`^SOURCE(\d*)_VFY$`)

// checkSourceURLPairing validates that every SOURCE<N> has a matching URL and vice versa.
// Returns (errors, warnings). Warnings are for missing SOURCE<N>_VFY.
func checkSourceURLPairing(file string, lines []detailsLine) ([]LintError, []LintError) {
	type sourceInfo struct {
		hasSource  bool
		sourceLine int
		hasURL     bool
		urlLine    int
		hasVFY     bool
	}

	groups := make(map[string]*sourceInfo)

	for _, dl := range lines {
		if dl.kind != kindAssignment {
			continue
		}

		// Strip array index (e.g., SOURCE_URL[0] → SOURCE_URL)
		baseName := dl.varName
		if idx := strings.Index(baseName, "["); idx >= 0 {
			baseName = baseName[:idx]
		}

		if m := sourceRe.FindStringSubmatch(baseName); m != nil {
			suffix := m[1]
			if groups[suffix] == nil {
				groups[suffix] = &sourceInfo{}
			}
			groups[suffix].hasSource = true
			groups[suffix].sourceLine = dl.lineNum
		} else if m := sourceURLRe.FindStringSubmatch(baseName); m != nil {
			suffix := m[1]
			if groups[suffix] == nil {
				groups[suffix] = &sourceInfo{}
			}
			groups[suffix].hasURL = true
			groups[suffix].urlLine = dl.lineNum
		} else if m := sourceVFYRe.FindStringSubmatch(baseName); m != nil {
			suffix := m[1]
			if groups[suffix] == nil {
				groups[suffix] = &sourceInfo{}
			}
			groups[suffix].hasVFY = true
		}
	}

	var errs, warns []LintError

	for suffix, info := range groups {
		label := "SOURCE"
		if suffix != "" {
			label = "SOURCE" + suffix
		}

		if info.hasSource && !info.hasURL {
			errs = append(errs, LintError{
				File:    file,
				Line:    info.sourceLine,
				Message: fmt.Sprintf("%s has no matching %s_URL or %s_URL_FULL", label, label, label),
			})
		}

		if info.hasURL && !info.hasSource {
			errs = append(errs, LintError{
				File:    file,
				Line:    info.urlLine,
				Message: fmt.Sprintf("%s_URL/_URL_FULL found but %s is not defined", label, label),
			})
		}

		if info.hasSource && !info.hasVFY {
			warns = append(warns, LintError{
				File:    file,
				Line:    info.sourceLine,
				Message: fmt.Sprintf("%s_VFY is not defined (recommended)", label),
			})
		}
	}

	return errs, warns
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
