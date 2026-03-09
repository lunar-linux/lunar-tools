package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMaxLineLength = 80

func writeTempDetails(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	modDir := filepath.Join(dir, "testmod")
	os.MkdirAll(modDir, 0755)
	path := filepath.Join(modDir, "DETAILS")
	os.WriteFile(path, []byte(content), 0644)
	return path
}

func TestDetailsCorrectlyAligned(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
This is a test module.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "not aligned") {
			t.Errorf("unexpected alignment error: %s", e)
		}
	}
}

func TestDetailsMisaligned(t *testing.T) {
	content := `MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	foundAlignErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "not aligned") {
			foundAlignErr = true
			break
		}
	}
	if !foundAlignErr {
		t.Error("expected alignment error for MODULE line")
	}
}

func TestDetailsSpecialOptionInMainBlock(t *testing.T) {
	// PSAFE is mixed in with main assignments — should be flagged
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
           PSAFE=no
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	foundSpecialErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "special option") && strings.Contains(e.Message, "PSAFE") {
			foundSpecialErr = true
			break
		}
	}
	if !foundSpecialErr {
		t.Error("expected error for PSAFE in main block")
	}
}

func TestDetailsSpecialOptionNotFlushLeft(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
  PSAFE=no
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	foundFlushErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "flush-left") {
			foundFlushErr = true
			break
		}
	}
	if !foundFlushErr {
		t.Error("expected flush-left error for indented PSAFE")
	}
}

func TestDetailsCorrectSpecialOption(t *testing.T) {
	// PSAFE flush-left after main block — no errors expected for special options
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
PSAFE=no
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "special option") || strings.Contains(e.Message, "flush-left") {
			t.Errorf("unexpected special option error: %s", e)
		}
	}
}

func TestDetailsHeredocTooLong(t *testing.T) {
	longLine := strings.Repeat("word ", 30) // 150 chars
	content := `MODULE=testmod
VERSION=1.0
SOURCE=$MODULE-$VERSION.tar.gz
SOURCE_VFY=sha256:abc123
WEB_SITE=http://example.com
ENTERED=20200101
UPDATED=20200101
SHORT="A test module"
cat << EOF
` + longLine + `
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	foundLenErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "exceeds") {
			foundLenErr = true
			break
		}
	}
	if !foundLenErr {
		t.Error("expected heredoc line length error")
	}
}

func TestDetailsMissingRequiredFields(t *testing.T) {
	content := `MODULE=testmod
VERSION=1.0
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	missing := map[string]bool{"SOURCE": false, "WEB_SITE": false, "ENTERED": false, "UPDATED": false, "SHORT": false}
	for _, e := range result.Errors {
		for field := range missing {
			if strings.Contains(e.Message, field) {
				missing[field] = true
			}
		}
	}
	for field, found := range missing {
		if !found {
			t.Errorf("expected missing field error for %s", field)
		}
	}
}

func TestDetailsFixAlignment(t *testing.T) {
	content := `MODULE=testmod
VERSION=1.0
SOURCE=$MODULE-$VERSION.tar.gz
SOURCE_URL=http://example.com
SOURCE_VFY=sha256:abc123
WEB_SITE=http://example.com
ENTERED=20200101
UPDATED=20200101
SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.Fixed {
		t.Error("expected Fixed=true")
	}

	// Re-read and verify alignment
	data, _ := os.ReadFile(path)
	fixed := string(data)
	fixedLines := parseDetailsLines(fixed)

	alignCol := fixAlignColumn(fixedLines)
	for _, dl := range fixedLines {
		if dl.kind == kindAssignment && !dl.isSpecial {
			actualCol := strings.Index(dl.raw, "=")
			if actualCol != alignCol {
				t.Errorf("line %d: expected '=' at column %d, got %d: %s", dl.lineNum, alignCol, actualCol, dl.raw)
			}
		}
	}
}

func TestDetailsFixSpecialOptionRelocation(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
           PSAFE=no
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})
	if !result.Fixed {
		t.Fatal("expected Fixed=true")
	}

	data, _ := os.ReadFile(path)
	fixed := string(data)

	// PSAFE should be flush-left and after the main block (before cat << EOF)
	fixedLines := strings.Split(fixed, "\n")
	catIdx := -1
	psafeIdx := -1
	for i, line := range fixedLines {
		if strings.HasPrefix(strings.TrimSpace(line), "cat <<") {
			catIdx = i
		}
		if strings.HasPrefix(line, "PSAFE=") {
			psafeIdx = i
		}
	}
	if psafeIdx < 0 {
		t.Fatal("PSAFE= not found in fixed output")
	}
	if catIdx < 0 {
		t.Fatal("cat << EOF not found in fixed output")
	}
	if psafeIdx >= catIdx {
		t.Errorf("PSAFE (line %d) should be before heredoc (line %d)", psafeIdx, catIdx)
	}
	if fixedLines[psafeIdx] != "PSAFE=no" {
		t.Errorf("PSAFE line should be flush-left, got %q", fixedLines[psafeIdx])
	}
}

func TestDetailsFixHeredocWrap(t *testing.T) {
	longLine := strings.Repeat("word ", 30) // 150 chars
	content := `MODULE=testmod
VERSION=1.0
SOURCE=$MODULE-$VERSION.tar.gz
SOURCE_VFY=sha256:abc123
WEB_SITE=http://example.com
ENTERED=20200101
UPDATED=20200101
SHORT="A test module"
cat << EOF
` + longLine + `
EOF
`
	path := writeTempDetails(t, content)
	LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	data, _ := os.ReadFile(path)
	fixed := string(data)
	for _, line := range strings.Split(fixed, "\n") {
		if len(line) > testMaxLineLength && !strings.Contains(line, "=") && line != "EOF" && !strings.HasPrefix(line, "cat") {
			t.Errorf("heredoc line still too long after fix: %d chars: %s", len(line), line)
		}
	}
}

func TestDetailsSourceURLArrays(t *testing.T) {
	content := `            MODULE=testmod
           VERSION=1.0
            SOURCE=$MODULE-$VERSION.tar.gz
   SOURCE_URL_FULL=http://example.com/v$VERSION.tar.gz
        SOURCE_VFY=sha256:abc123
          WEB_SITE=http://example.com
           ENTERED=20200101
           UPDATED=20200101
             SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "not aligned") {
			t.Errorf("unexpected alignment error: %s", e)
		}
	}
}

func TestDetailsCommentsPreserved(t *testing.T) {
	content := `# This is a comment
          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# This is a comment") {
		t.Error("comment was not preserved after fix")
	}
}

func TestDetailsMultiSource(t *testing.T) {
	content := `            MODULE=cargo-c
           VERSION=0.10.20
            SOURCE=$MODULE-$VERSION.tar.gz
           SOURCE2=Cargo.lock
   SOURCE_URL_FULL=https://github.com/lu-zero/cargo-c/archive/refs/tags/v$VERSION.tar.gz
  SOURCE2_URL_FULL=https://github.com/lu-zero/cargo-c/releases/download/v$VERSION/Cargo.lock
        SOURCE_VFY=sha256:abc123
       SOURCE2_VFY=sha256:def456
          WEB_SITE=https://github.com/lu-zero/cargo-c/
           ENTERED=20230105
           UPDATED=20260202
             SHORT="build and install C-ABI libraries"
cat << EOF
Cargo applet to build and install C-ABI compatible dynamic and static libraries.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "not aligned") {
			t.Errorf("unexpected alignment error in multi-source file: %s", e)
		}
	}
}

func TestDetailsFixClearsFixableErrors(t *testing.T) {
	// Misaligned file — all errors are fixable
	content := `MODULE=testmod
VERSION=1.0
SOURCE=$MODULE-$VERSION.tar.gz
SOURCE_URL=http://example.com
SOURCE_VFY=sha256:abc123
WEB_SITE=http://example.com
ENTERED=20200101
UPDATED=20200101
SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.Fixed {
		t.Error("expected Fixed=true")
	}
	// After fix, no errors should remain (all were fixable alignment issues)
	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected remaining error after fix: %s", e)
		}
	}
}

func TestDetailsFixKeepsUnfixableErrors(t *testing.T) {
	// Missing required fields (unfixable) + misaligned (fixable)
	content := `MODULE=testmod
VERSION=1.0
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.Fixed {
		t.Error("expected Fixed=true")
	}
	// Missing fields should still be reported
	if !result.HasErrors() {
		t.Error("expected remaining errors for missing required fields")
	}
	for _, e := range result.Errors {
		if !strings.Contains(e.Message, "missing required field") {
			t.Errorf("expected only missing-field errors after fix, got: %s", e)
		}
	}
}

func TestDetailsExactDuplicate(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	foundDup := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "duplicate assignment") {
			foundDup = true
			if !e.Fixable {
				t.Error("exact duplicate should be fixable")
			}
		}
	}
	if !foundDup {
		t.Error("expected duplicate assignment error for SOURCE")
	}
}

func TestDetailsConflictingDuplicate(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
PSAFE=no
PSAFE=yes
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	conflictCount := 0
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "conflicting duplicate") {
			conflictCount++
			if e.Fixable {
				t.Error("conflicting duplicate should NOT be fixable")
			}
		}
	}
	if conflictCount != 2 {
		t.Errorf("expected 2 conflicting duplicate errors, got %d", conflictCount)
	}
}

func TestDetailsFixExactDuplicate(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.Fixed {
		t.Error("expected Fixed=true")
	}
	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected remaining error after fix: %s", e)
		}
	}

	// Verify the duplicate was removed
	data, _ := os.ReadFile(path)
	count := strings.Count(string(data), "SOURCE=$MODULE-$VERSION.tar.gz")
	if count != 1 {
		t.Errorf("expected SOURCE to appear once after dedup, found %d times", count)
	}
}

func TestDetailsFixConflictingDuplicateStillFails(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
PSAFE=no
PSAFE=yes
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.HasErrors() {
		t.Error("expected errors to remain for conflicting duplicates")
	}
	foundConflict := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "conflicting duplicate") {
			foundConflict = true
		}
	}
	if !foundConflict {
		t.Error("expected conflicting duplicate error to remain after fix")
	}
}

func TestDetailsHeredocSpacingMissing(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "blank line before heredoc") {
			found = true
			if !e.Fixable {
				t.Error("heredoc spacing should be fixable")
			}
		}
	}
	if !found {
		t.Error("expected missing blank line before heredoc error")
	}
}

func TestDetailsHeredocSpacingTooMany(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"



cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "blank line before heredoc") {
			found = true
		}
	}
	if !found {
		t.Error("expected too many blank lines before heredoc error")
	}
}

func TestDetailsHeredocSpacingCorrect(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "blank line before heredoc") {
			t.Errorf("unexpected heredoc spacing error: %s", e)
		}
	}
}

func TestDetailsFixHeredocSpacing(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"
cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.Fixed {
		t.Error("expected Fixed=true")
	}

	// Re-lint should find no spacing errors
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "blank line before heredoc") {
			t.Errorf("heredoc spacing error should be fixed: %s", e)
		}
	}

	// Verify the file has exactly one blank line before cat << EOF
	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "cat <<") {
			if i == 0 || lines[i-1] != "" {
				t.Error("expected blank line immediately before cat << EOF")
			}
			if i >= 2 && lines[i-2] == "" {
				t.Error("expected only ONE blank line before cat << EOF")
			}
			break
		}
	}
}

func TestDetailsTrailingNewlinesAfterEOF(t *testing.T) {
	content := "          MODULE=testmod\n         VERSION=1.0\n          SOURCE=$MODULE-$VERSION.tar.gz\n      SOURCE_VFY=sha256:abc123\n        WEB_SITE=http://example.com\n         ENTERED=20200101\n         UPDATED=20200101\n           SHORT=\"A test module\"\n\ncat << EOF\nTest.\nEOF\n\n\n\n"
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "trailing newline after EOF") {
			found = true
			if !e.Fixable {
				t.Error("trailing newlines after EOF should be fixable")
			}
		}
	}
	if !found {
		t.Error("expected trailing newline error after EOF")
	}
}

func TestDetailsTrailingNewlinesCorrect(t *testing.T) {
	content := "          MODULE=testmod\n         VERSION=1.0\n          SOURCE=$MODULE-$VERSION.tar.gz\n      SOURCE_VFY=sha256:abc123\n        WEB_SITE=http://example.com\n         ENTERED=20200101\n         UPDATED=20200101\n           SHORT=\"A test module\"\n\ncat << EOF\nTest.\nEOF\n"
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "trailing newline after EOF") {
			t.Errorf("unexpected trailing newline error: %s", e)
		}
	}
}

func TestDetailsFixTrailingNewlines(t *testing.T) {
	content := "          MODULE=testmod\n         VERSION=1.0\n          SOURCE=$MODULE-$VERSION.tar.gz\n      SOURCE_VFY=sha256:abc123\n        WEB_SITE=http://example.com\n         ENTERED=20200101\n         UPDATED=20200101\n           SHORT=\"A test module\"\n\ncat << EOF\nTest.\nEOF\n\n\n\n"
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: testMaxLineLength})

	if !result.Fixed {
		t.Error("expected Fixed=true")
	}
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "trailing newline after EOF") {
			t.Errorf("trailing newline error should be fixed: %s", e)
		}
	}

	// Verify file ends with exactly "EOF\n"
	data, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(data), "EOF\n") {
		t.Errorf("expected file to end with EOF\\n, got: %q", string(data)[len(data)-10:])
	}
	if strings.HasSuffix(string(data), "EOF\n\n") {
		t.Error("file has extra trailing newline after EOF")
	}
}

func TestDetailsValidDates(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200601
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "ENTERED") || strings.Contains(e.Message, "UPDATED") {
			t.Errorf("unexpected date error: %s", e)
		}
	}
}

func TestDetailsInvalidDateFormat(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=2020-01-01
         UPDATED=20200601
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "invalid date format") {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid date format error for ENTERED")
	}
}

func TestDatesFutureDate(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20991231
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "in the future") {
			found = true
		}
	}
	if !found {
		t.Error("expected future date error for UPDATED")
	}
}

func TestDatesUpdatedBeforeEntered(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200601
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "UPDATED") && strings.Contains(e.Message, "before ENTERED") {
			found = true
		}
	}
	if !found {
		t.Error("expected UPDATED before ENTERED error")
	}
}

func TestDetailsModuleNameMatchesDir(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "does not match directory") {
			t.Errorf("unexpected MODULE name error: %s", e)
		}
	}
}

func TestDetailsModuleNameMismatch(t *testing.T) {
	content := `          MODULE=wrong-name
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "does not match directory") {
			found = true
		}
	}
	if !found {
		t.Error("expected MODULE name mismatch error")
	}
}

func TestSourceURLPairingValid(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE") && strings.Contains(e.Message, "URL") {
			t.Errorf("unexpected SOURCE/URL pairing error: %s", e)
		}
	}
}

func TestSourceURLFullPairingValid(t *testing.T) {
	content := `            MODULE=testmod
           VERSION=1.0
            SOURCE=$MODULE-$VERSION.tar.gz
   SOURCE_URL_FULL=http://example.com/v1.0.tar.gz
        SOURCE_VFY=sha256:abc123
          WEB_SITE=http://example.com
           ENTERED=20200101
           UPDATED=20200101
             SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE") && strings.Contains(e.Message, "URL") {
			t.Errorf("unexpected SOURCE/URL pairing error: %s", e)
		}
	}
}

func TestSourceURLPairingMissingURL(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE has no matching SOURCE_URL or SOURCE_URL_FULL") {
			found = true
		}
	}
	if !found {
		t.Error("expected error for SOURCE without URL")
	}
}

func TestSourceURLPairingOrphanedURL(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
      SOURCE_VFY=sha256:abc123
  SOURCE2_URL_FULL=http://example.com/extra.tar.gz
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE2_URL/_URL_FULL found but SOURCE2 is not defined") {
			found = true
		}
	}
	if !found {
		t.Error("expected error for orphaned SOURCE2_URL_FULL")
	}
}

func TestSourceURLPairingMultiSourceValid(t *testing.T) {
	content := `            MODULE=testmod
           VERSION=1.0
            SOURCE=$MODULE-$VERSION.tar.gz
           SOURCE2=extra.tar.gz
   SOURCE_URL_FULL=http://example.com/v1.0.tar.gz
  SOURCE2_URL_FULL=http://example.com/extra.tar.gz
        SOURCE_VFY=sha256:abc123
       SOURCE2_VFY=sha256:def456
          WEB_SITE=http://example.com
           ENTERED=20200101
           UPDATED=20200101
             SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE") && strings.Contains(e.Message, "URL") {
			t.Errorf("unexpected SOURCE/URL pairing error: %s", e)
		}
	}
}

func TestSourceURLPairingMultiSourceMissingURL(t *testing.T) {
	content := `            MODULE=testmod
           VERSION=1.0
            SOURCE=$MODULE-$VERSION.tar.gz
           SOURCE2=extra.tar.gz
   SOURCE_URL_FULL=http://example.com/v1.0.tar.gz
        SOURCE_VFY=sha256:abc123
          WEB_SITE=http://example.com
           ENTERED=20200101
           UPDATED=20200101
             SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE2 has no matching SOURCE2_URL or SOURCE2_URL_FULL") {
			found = true
		}
	}
	if !found {
		t.Error("expected error for SOURCE2 without URL")
	}
}

func TestSourceURLPairingMultiSourceOrphanedURL(t *testing.T) {
	content := `            MODULE=testmod
           VERSION=1.0
            SOURCE=$MODULE-$VERSION.tar.gz
   SOURCE_URL_FULL=http://example.com/v1.0.tar.gz
  SOURCE3_URL_FULL=http://example.com/orphan.tar.gz
        SOURCE_VFY=sha256:abc123
          WEB_SITE=http://example.com
           ENTERED=20200101
           UPDATED=20200101
             SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE3_URL/_URL_FULL found but SOURCE3 is not defined") {
			found = true
		}
	}
	if !found {
		t.Error("expected error for orphaned SOURCE3_URL_FULL")
	}
}

func TestSourceVFYWarning(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	// Should be a warning, not an error
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "SOURCE_VFY") {
			t.Errorf("SOURCE_VFY should be a warning, not an error: %s", e)
		}
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "SOURCE_VFY is not defined") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for missing SOURCE_VFY")
	}
}

func TestSourceVFYWarningMultiSource(t *testing.T) {
	content := `            MODULE=testmod
           VERSION=1.0
            SOURCE=$MODULE-$VERSION.tar.gz
           SOURCE2=extra.tar.gz
   SOURCE_URL_FULL=http://example.com/v1.0.tar.gz
  SOURCE2_URL_FULL=http://example.com/extra.tar.gz
        SOURCE_VFY=sha256:abc123
          WEB_SITE=http://example.com
           ENTERED=20200101
           UPDATED=20200101
             SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "SOURCE2_VFY is not defined") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for missing SOURCE2_VFY")
	}
	// SOURCE_VFY is present, so no warning for it
	for _, w := range result.Warnings {
		if w.Message == "SOURCE_VFY is not defined (recommended)" {
			t.Error("unexpected warning for SOURCE_VFY which is present")
		}
	}
}

func TestSourceVFYNoWarningWhenPresent(t *testing.T) {
	content := `          MODULE=testmod
         VERSION=1.0
          SOURCE=$MODULE-$VERSION.tar.gz
      SOURCE_URL=http://example.com
      SOURCE_VFY=sha256:abc123
        WEB_SITE=http://example.com
         ENTERED=20200101
         UPDATED=20200101
           SHORT="A test module"

cat << EOF
Test.
EOF
`
	path := writeTempDetails(t, content)
	result := LintDetails(path, LintOptions{MaxLineLength: testMaxLineLength})

	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "SOURCE_VFY") {
			t.Errorf("unexpected warning for SOURCE_VFY when it is present: %s", w)
		}
	}
}
