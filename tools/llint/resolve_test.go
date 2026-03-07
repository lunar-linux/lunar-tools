package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupMoonbase(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	moonbase := filepath.Join(base, "var", "lib", "lunar", "moonbase")

	// Create a regular section with a module
	os.MkdirAll(filepath.Join(moonbase, "devel", "git"), 0755)

	// Create a zlocal section with an override module
	os.MkdirAll(filepath.Join(moonbase, "zlocal", "mymod"), 0755)

	// Create module.index
	indexDir := filepath.Join(base, "var", "state", "lunar")
	os.MkdirAll(indexDir, 0755)
	os.WriteFile(filepath.Join(indexDir, "module.index"), []byte(
		"git:devel\nflac:audio\ncurl:net\n",
	), 0644)

	// Create the audio/flac dir too
	os.MkdirAll(filepath.Join(moonbase, "audio", "flac"), 0755)

	return moonbase
}

func TestResolveModuleZlocalPriority(t *testing.T) {
	moonbase := setupMoonbase(t)

	dir, err := ResolveModule(moonbase, "mymod")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
	if filepath.Base(filepath.Dir(dir)) != "zlocal" {
		t.Errorf("expected zlocal section, got %q", dir)
	}
}

func TestResolveModuleIndexFallback(t *testing.T) {
	moonbase := setupMoonbase(t)

	dir, err := ResolveModule(moonbase, "git")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "git" {
		t.Errorf("expected git dir, got %q", dir)
	}
}

func TestResolveModuleIndexOtherSection(t *testing.T) {
	moonbase := setupMoonbase(t)

	dir, err := ResolveModule(moonbase, "flac")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(dir)) != "audio" {
		t.Errorf("expected audio section, got %q", dir)
	}
}

func TestResolveModuleNotFound(t *testing.T) {
	moonbase := setupMoonbase(t)

	_, err := ResolveModule(moonbase, "nonexistent")
	if err == nil {
		t.Error("expected error for missing module")
	}
}

func TestResolveModuleZlocalOverridesIndex(t *testing.T) {
	moonbase := setupMoonbase(t)

	// Add a module that exists in both zlocal and index
	os.MkdirAll(filepath.Join(moonbase, "zlocal", "git"), 0755)

	dir, err := ResolveModule(moonbase, "git")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(dir)) != "zlocal" {
		t.Errorf("zlocal should take priority, got %q", dir)
	}
}
