// cmd/verify_specs/main.go
// Spec verification CLI: reads specs/, validates spec→code alignment, outputs report.
//
// Usage:
//   verify_specs [--workspace ROOT] [--format json|summary] [--check TYPE] [--level LVLS]
//
// Examples:
//   verify_specs                           # verify current directory
//   verify_specs --workspace /path/to/prj  # verify specific workspace
//   verify_specs --format json             # machine-readable output
//   verify_specs --check file_existence    # only file existence checks
//   verify_specs --level 5_slice           # only L5 specs
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"vibex-workbench/pkg/verify"
)

var (
	flagWorkspace    = flag.String("workspace", ".", "workspace root directory containing specs/")
	flagFormat       = flag.String("format", "summary", "output format: summary (default), json, short")
	flagChecks       = flag.String("check", "", "comma-separated checks to run: file_existence,parent_chain,completeness,behaviors,go_struct,svelte_props (default: all)")
	flagLevels       = flag.String("level", "", "comma-separated spec levels to check: 1_concept,2_skeleton,3_module,4_feature,5_slice (default: all)")
	flagSpecNames    = flag.String("spec", "", "comma-separated spec names to check (default: all)")
	flagShowPass     = flag.Bool("show-pass", false, "show passing checks in output (default: false)")
	flagHelp         = flag.Bool("help", false, "show help")
)

func main() {
	flag.Parse()
	if *flagHelp {
		flag.Usage()
		os.Exit(0)
	}

	workspace := *flagWorkspace
	if workspace == "" {
		workspace = "."
	}

	// Load specs
	loader := verify.NewLoader(workspace)
	specs, err := loader.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading specs: %v\n", err)
		os.Exit(1)
	}

	if len(specs) == 0 {
		fmt.Fprintf(os.Stderr, "No spec files found in %s/specs/\n", workspace)
		os.Exit(1)
	}

	// Build verifier options
	opts := verify.DefaultVerifierOptions()

	if *flagChecks != "" {
		opts.CheckFileExistence = false
		opts.CheckParentChain = false
		opts.CheckCompleteness = false
		opts.CheckBehaviors = false
		opts.CheckGoStructFields = false
		opts.CheckSvelteProps = false
		for _, c := range strings.Split(*flagChecks, ",") {
			switch strings.TrimSpace(c) {
			case "file_existence":
				opts.CheckFileExistence = true
			case "parent_chain":
				opts.CheckParentChain = true
			case "completeness":
				opts.CheckCompleteness = true
			case "behaviors":
				opts.CheckBehaviors = true
			case "go_struct":
				opts.CheckGoStructFields = true
			case "svelte_props":
				opts.CheckSvelteProps = true
			default:
				fmt.Fprintf(os.Stderr, "Unknown check type: %q\n", c)
				os.Exit(1)
			}
		}
	}

	if *flagLevels != "" {
		opts.OnlySpecLevels = nil
		for _, l := range strings.Split(*flagLevels, ",") {
			lvl := strings.TrimSpace(l)
			if lvl != "" {
				opts.OnlySpecLevels = append(opts.OnlySpecLevels, lvl)
			}
		}
	}

	verifier := verify.NewVerifier(workspace, specs).WithOptions(opts)
	report := verifier.Run()

	// Filter by spec names if requested
	if *flagSpecNames != "" {
		allowed := make(map[string]bool)
		for _, name := range strings.Split(*flagSpecNames, ",") {
			allowed[strings.TrimSpace(name)] = true
		}
		var filtered []verify.Result
		for _, r := range report.Results {
			if allowed[r.SpecName] {
				filtered = append(filtered, r)
			}
		}
		report.Results = filtered
	}

	// Filter out passes if not requested
	if !*flagShowPass {
		var filtered []verify.Result
		for _, r := range report.Results {
			if r.Status != "pass" {
				filtered = append(filtered, r)
			}
		}
		report.Results = filtered
	}

	// Output
	switch *flagFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
			os.Exit(1)
		}
	case "short":
		printShort(report)
	default:
		printSummary(report)
	}

	// Exit code: non-zero if there are failures
	if report.FailCount > 0 {
		os.Exit(1)
	}
}

func printSummary(r *verify.Report) {
	fmt.Printf("📋 Spec Verification Report\n")
	fmt.Printf("   Workspace: %s\n", r.WorkspaceRoot)
	fmt.Printf("   %s\n\n", r.Summary)

	if len(r.Results) == 0 {
		fmt.Println("✅ All checks passed!")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, res := range r.Results {
		icon := "✅"
		switch res.Severity {
		case "error":
			icon = "❌"
		case "warning":
			icon = "⚠️"
		case "info":
			icon = "ℹ️"
		}
		loc := ""
		if res.FilePath != "" {
			loc = fmt.Sprintf("  [%s]", res.FilePath)
		}
		fmt.Fprintf(tw, "%s %s/%s | %s | %s%s\n", icon, res.SpecLevel, res.SpecName, res.CheckType, res.Message, loc)
		if res.Suggestion != "" && res.Severity != "info" {
			fmt.Fprintf(tw, "  💡 %s\n", res.Suggestion)
		}
	}
	tw.Flush()
}

func printShort(r *verify.Report) {
	if r.FailCount == 0 && r.WarnCount == 0 {
		fmt.Println("OK")
		return
	}
	for _, res := range r.Results {
		if res.Status == "fail" || res.Status == "warn" {
			loc := ""
			if res.FilePath != "" {
				loc = fmt.Sprintf(" (%s)", res.FilePath)
			}
			sym := "WARN"
			if res.Severity == "error" {
				sym = "FAIL"
			}
			fmt.Printf("[%s] %s/%s%s: %s\n", sym, res.SpecName, res.CheckType, loc, res.Message)
		}
	}
}
