package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})
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
	LintDetails(path, LintOptions{Fix: true, MaxLineLength: 80})

	data, _ := os.ReadFile(path)
	fixed := string(data)
	for _, line := range strings.Split(fixed, "\n") {
		if len(line) > 80 && !strings.Contains(line, "=") && line != "EOF" && !strings.HasPrefix(line, "cat") {
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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})

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
	result := LintDetails(path, LintOptions{Fix: true, MaxLineLength: 120})

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
