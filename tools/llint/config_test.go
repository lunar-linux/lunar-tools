package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestParseConfigBasic(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "config")
	writeFile(t, global, `
# comment
              MOONBASE=/var/lib/lunar/moonbase
          MODULE_INDEX=/var/state/lunar/module.index
export            PATH=/sbin:/bin
`)

	cfg, err := LoadConfigFrom(global, filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Moonbase != "/var/lib/lunar/moonbase" {
		t.Errorf("Moonbase = %q, want /var/lib/lunar/moonbase", cfg.Moonbase)
	}
	if cfg.ModuleIndex != "/var/state/lunar/module.index" {
		t.Errorf("ModuleIndex = %q, want /var/state/lunar/module.index", cfg.ModuleIndex)
	}
}

func TestParseConfigLocalOverride(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "config")
	local := filepath.Join(dir, "local")

	writeFile(t, global, `MOONBASE=/var/lib/lunar/moonbase
MODULE_INDEX=/var/state/lunar/module.index`)
	writeFile(t, local, `MOONBASE=/custom/moonbase`)

	cfg, err := LoadConfigFrom(global, local)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Moonbase != "/custom/moonbase" {
		t.Errorf("Moonbase = %q, want /custom/moonbase", cfg.Moonbase)
	}
}

func TestParseConfigQuotedValues(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "config")
	writeFile(t, global, `MOONBASE="/var/lib/lunar/moonbase"
MODULE_INDEX='/var/state/lunar/module.index'`)

	cfg, err := LoadConfigFrom(global, filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Moonbase != "/var/lib/lunar/moonbase" {
		t.Errorf("Moonbase = %q, want /var/lib/lunar/moonbase", cfg.Moonbase)
	}
	if cfg.ModuleIndex != "/var/state/lunar/module.index" {
		t.Errorf("ModuleIndex = %q", cfg.ModuleIndex)
	}
}

func TestParseConfigMissingMoonbase(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "config")
	writeFile(t, global, `FOO=bar`)

	_, err := LoadConfigFrom(global, filepath.Join(dir, "nonexistent"))
	if err == nil {
		t.Error("expected error for missing MOONBASE")
	}
}

func TestParseConfigMissingGlobal(t *testing.T) {
	_, err := LoadConfigFrom("/nonexistent/config", "/nonexistent/local")
	if err == nil {
		t.Error("expected error for missing global config")
	}
}

func TestParseConfigSkipsArrays(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "config")
	writeFile(t, global, `MOONBASE=/mb
MODULE_INDEX=/mi
MOONBASE_URL[0]=http://example.com
MOONBASE_URL[1]=http://other.com`)

	cfg, err := LoadConfigFrom(global, filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Moonbase != "/mb" {
		t.Errorf("Moonbase = %q", cfg.Moonbase)
	}
}

func TestParseConfigBashDefaultSyntax(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "config")
	// ${VAR:-default} is taken literally — no expansion
	writeFile(t, global, `MOONBASE=/mb
MODULE_INDEX=/mi
ARCHIVE=${ARCHIVE:-on}`)

	cfg, err := LoadConfigFrom(global, filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Moonbase != "/mb" {
		t.Errorf("Moonbase = %q", cfg.Moonbase)
	}
}
