package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveModule finds a module directory by name.
// It checks zlocal sections first (zlocal overrides), then falls back to module.index.
// moduleIndex is optional — if empty, the standard path is computed from moonbase.
func ResolveModule(moonbase, moduleName string) (string, error) {
	return ResolveModuleWithIndex(moonbase, "", moduleName)
}

// ResolveModuleWithIndex is like ResolveModule but accepts an explicit module.index path.
func ResolveModuleWithIndex(moonbase, moduleIndex, moduleName string) (string, error) {
	// Check zlocal sections first
	if dir, err := findInZlocal(moonbase, moduleName); err == nil {
		return dir, nil
	}

	// Fall back to module.index
	return findInModuleIndex(moonbase, moduleIndex, moduleName)
}

// findInZlocal scans all zlocal* directories in moonbase for the module.
func findInZlocal(moonbase, moduleName string) (string, error) {
	entries, err := os.ReadDir(moonbase)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "zlocal") {
			continue
		}
		modDir := filepath.Join(moonbase, entry.Name(), moduleName)
		if info, err := os.Stat(modDir); err == nil && info.IsDir() {
			return modDir, nil
		}
	}

	return "", fmt.Errorf("module %q not found in zlocal", moduleName)
}

// findInModuleIndex looks up the module in the module.index file.
// Moonbase is typically /var/lib/lunar/moonbase, index is /var/state/lunar/module.index.
// They share /var as a common ancestor.
func findInModuleIndex(moonbase, moduleIndex, moduleName string) (string, error) {
	indexPath := moduleIndex

	if indexPath == "" {
		// Go up from moonbase (/var/lib/lunar/moonbase) to /var, then into state/lunar/
		varDir := filepath.Dir(filepath.Dir(filepath.Dir(moonbase)))
		indexPath = filepath.Join(varDir, "state", "lunar", "module.index")

		// If standard location doesn't exist, try alongside moonbase's parent
		if _, err := os.Stat(indexPath); err != nil {
			indexPath = filepath.Join(filepath.Dir(moonbase), "module.index")
		}
	}

	f, err := os.Open(indexPath)
	if err != nil {
		return "", fmt.Errorf("cannot open module.index: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == moduleName {
			modDir := filepath.Join(moonbase, parts[1], moduleName)
			if info, err := os.Stat(modDir); err == nil && info.IsDir() {
				return modDir, nil
			}
			return "", fmt.Errorf("module %q found in index (section %s) but directory does not exist", moduleName, parts[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("module %q not found", moduleName)
}
