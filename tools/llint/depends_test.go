package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempDepends(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	modDir := filepath.Join(dir, "testmod")
	os.MkdirAll(modDir, 0755)
	path := filepath.Join(modDir, "DEPENDS")
	os.WriteFile(path, []byte(content), 0644)
	return path
}

func TestDependsCleanFile(t *testing.T) {
	content := `depends python
depends zlib

optional_depends curl "--with-curl" "" "for http(s) transports" y
optional_depends expat "--with-expat" "" "for WebDAV" y
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestDependsAllFourFunctions(t *testing.T) {
	content := `depends python
optional_depends curl "--with-curl" "" "for http" y
optional_depends_requires curl curl-dev
optional_depends_one_of "TLS implementation" openssl "--with-openssl" "" gnutls "--with-gnutls" ""
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestDependsContinuationLines(t *testing.T) {
	content := `depends python

optional_depends gnutls \
                 "--with-gnutls" \
                 "--without-gnutls" \
                 "for TLS support" y
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestDependsIfLogic(t *testing.T) {
	content := `depends python

if [ "` + "`" + `get_module_config DOCS` + "`" + `" = "y" ]; then
  depends xmlto
fi
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if !result.HasErrors() {
		t.Error("expected errors for if/fi logic")
	}

	foundIf := false
	foundFi := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "'if'") {
			foundIf = true
		}
		if strings.Contains(e.Message, "'fi'") {
			foundFi = true
		}
	}
	if !foundIf {
		t.Error("expected error for 'if'")
	}
	if !foundFi {
		t.Error("expected error for 'fi'")
	}
}

func TestDependsCaseLogic(t *testing.T) {
	content := `case $FOO in
  bar) depends baz ;;
esac
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if !result.HasErrors() {
		t.Error("expected errors for case/esac")
	}
}

func TestDependsVariableAssignment(t *testing.T) {
	content := `FOO=bar
depends python
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	foundAssign := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "variable assignment") {
			foundAssign = true
		}
	}
	if !foundAssign {
		t.Error("expected variable assignment error")
	}
}

func TestDependsCommandSubstitution(t *testing.T) {
	content := `depends $(get_module python)
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if !result.HasErrors() {
		t.Error("expected error for command substitution")
	}
}

func TestDependsCommentsIgnored(t *testing.T) {
	content := `# This is a comment
depends python
# Another comment
optional_depends curl "" "" "for curl"
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestDependsBlankLinesIgnored(t *testing.T) {
	content := `depends python

depends zlib


optional_depends curl "" "" "for curl"
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e)
		}
	}
}

func TestDependsMixedValidInvalid(t *testing.T) {
	content := `depends python
if true; then
  depends bad
fi
optional_depends curl "" "" "ok"
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	// Should flag "if true; then", "depends bad" is inside if so it's OK as a logical line,
	// actually "  depends bad" after joining is just "depends bad" which is valid.
	// But "if true; then" and "fi" are invalid.
	errorCount := 0
	for _, e := range result.Errors {
		errorCount++
		_ = e
	}
	if errorCount < 2 {
		t.Errorf("expected at least 2 errors (if and fi), got %d", errorCount)
	}
}

func TestDependsCorrectLineNumbers(t *testing.T) {
	content := `depends python
depends zlib
if foo; then
  depends bar
fi
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	for _, e := range result.Errors {
		if strings.Contains(e.Message, "'if'") && e.Line != 3 {
			t.Errorf("expected 'if' error on line 3, got line %d", e.Line)
		}
		if strings.Contains(e.Message, "'fi'") && e.Line != 5 {
			t.Errorf("expected 'fi' error on line 5, got line %d", e.Line)
		}
	}
}

func TestDependsNonFixable(t *testing.T) {
	content := `if true; then
  depends foo
fi
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	for _, e := range result.Errors {
		if e.Fixable {
			t.Errorf("DEPENDS errors should not be fixable: %s", e)
		}
	}
}

func TestDependsPercentAlias(t *testing.T) {
	// %OSSL is a valid argument to depends
	content := `depends %OSSL
depends zlib
`
	path := writeTempDepends(t, content)
	result := LintDepends(path, LintOptions{})

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e)
		}
	}
}
