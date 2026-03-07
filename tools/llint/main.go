package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fix := flag.Bool("fix", false, "auto-fix fixable issues")
	maxLineLength := flag.Int("max-line-length", 120, "maximum line length for heredoc text in DETAILS")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <module-name>\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Lint check Lunar Linux module files.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	moduleName := flag.Arg(0)

	opts := LintOptions{
		Fix:           *fix,
		MaxLineLength: *maxLineLength,
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	moduleDir, err := ResolveModuleWithIndex(cfg.Moonbase, cfg.ModuleIndex, moduleName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	var result LintResult

	detailsPath := filepath.Join(moduleDir, "DETAILS")
	if _, err := os.Stat(detailsPath); err == nil {
		result.Merge(LintDetails(detailsPath, opts))
	}

	dependsPath := filepath.Join(moduleDir, "DEPENDS")
	if _, err := os.Stat(dependsPath); err == nil {
		result.Merge(LintDepends(dependsPath, opts))
	}

	for _, e := range result.Errors {
		fmt.Println(e)
	}

	if result.Fixed {
		fmt.Println("Fixes applied.")
	}

	if result.HasErrors() {
		os.Exit(1)
	}
}
