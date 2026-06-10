package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/proton"
)

var (
	target       = flag.String("target", "", "Target directory or file to scan")
	templatePath = flag.String("templates", "", "Path to template files or directory")
	category     = flag.String("category", "keys", "Template category (keys, spray)")
	severity     = flag.String("severity", "", "Filter by severity (comma-separated)")
	tags         = flag.String("tags", "", "Filter by tags (comma-separated)")
)

func main() {
	flag.Parse()

	if *target == "" {
		fmt.Println("Usage: proton -target <path> [-templates <path>] [-category <name>]")
		os.Exit(1)
	}

	config := proton.NewConfig()

	if *templatePath != "" {
		config.WithTemplatePaths(*templatePath)
	} else {
		config.WithCategories(strings.Split(*category, ",")...)
	}

	if *tags != "" {
		config.WithTags(strings.Split(*tags, ",")...)
	}

	engine := proton.NewEngine(config)
	if err := engine.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing engine: %v\n", err)
		os.Exit(1)
	}

	ctx := proton.NewContext()
	findings, err := engine.Scan(ctx, *target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
		os.Exit(1)
	}

	for _, f := range findings {
		sev := f.Severity
		if sev == "" {
			sev = "unknown"
		}
		fmt.Printf("[%s] %s (%s) %s\n", sev, f.TemplateID, f.TemplateName, f.FilePath)
		for name, events := range f.Matches {
			for _, e := range events {
				fmt.Printf("  match [%s] line %d: %s\n", name, e.Line, e.Value)
			}
		}
		for _, e := range f.Extracts {
			fmt.Printf("  extract line %d: %s\n", e.Line, e.Value)
		}
	}

	stats := engine.Scanner().Stats
	fmt.Printf("\nScan complete: %d files (%s), %d findings\n",
		stats.Files, stats.HumanBytes(), stats.Findings)
}
