package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chainreactors/sdk/proton"
)

var (
	input        = flag.String("input", "", "File to scan (use - for stdin)")
	templatePath = flag.String("templates", "", "Path to template files or directory")
	category     = flag.String("category", "keys", "Template category (keys, spray)")
	tags         = flag.String("tags", "", "Filter by tags (comma-separated)")
)

func main() {
	flag.Parse()

	if *input == "" {
		fmt.Println("Usage: proton -input <file|-> [-templates <path>] [-category <name>]")
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

	engine, err := proton.NewEngine(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing engine: %v\n", err)
		os.Exit(1)
	}

	var data []byte
	label := *input
	if *input == "-" {
		data, err = io.ReadAll(os.Stdin)
		label = "stdin"
	} else {
		data, err = os.ReadFile(*input)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	findings := engine.ScanData(data, label)

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

	fmt.Printf("\nScan complete: %d findings\n", len(findings))
}
