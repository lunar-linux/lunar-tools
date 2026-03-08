package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	fix := flag.Bool("fix", false, "auto-fix fixable issues")
	verbose := flag.Bool("verbose", false, "show what was fixed (use with --fix)")
	maxLineLength := flag.Int("max-line-length", 80, "maximum line length for heredoc text in DETAILS")
	pathFlag := flag.String("path", "", "path to a module directory (skips config and index lookup)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <module-name>\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "       %s --path <module-dir>\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Lint check Lunar Linux module files.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("llint %s (commit: %s, built: %s)\n", version, commit, buildDate)
		os.Exit(0)
	}

	opts := LintOptions{
		Fix:           *fix,
		Verbose:       *verbose,
		MaxLineLength: *maxLineLength,
	}

	var moduleDir string

	if *pathFlag != "" {
		// Direct path mode — no config or index needed
		info, err := os.Stat(*pathFlag)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "error: %q is not a valid directory\n", *pathFlag)
			os.Exit(2)
		}
		moduleDir = *pathFlag
	} else {
		if flag.NArg() != 1 {
			flag.Usage()
			os.Exit(2)
		}

		moduleName := flag.Arg(0)

		cfg, err := LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		moduleDir, err = ResolveModuleWithIndex(cfg.Moonbase, cfg.ModuleIndex, moduleName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
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

	for _, msg := range result.FixedMsgs {
		fmt.Println(msg)
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
