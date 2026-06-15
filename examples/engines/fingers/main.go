package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/utils/httputils"
)

var (
	cyberhubURL = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey      = flag.String("key", "", "Cyberhub API Key")
	target      = flag.String("target", "", "Target URL to match (required)")
	jsonOut     = flag.Bool("json", false, "Output as JSON")
)

func main() {
	flag.Parse()

	if *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: fingers -target <url> [-url <cyberhub_url> -key <api_key>] [-json]")
		fmt.Fprintln(os.Stderr, "\nWith Cyberhub:")
		fmt.Fprintln(os.Stderr, "  fingers -url http://127.0.0.1:8080 -key your_key -target http://example.com")
		fmt.Fprintln(os.Stderr, "\nWith local data:")
		fmt.Fprintln(os.Stderr, "  fingers -target http://example.com")
		os.Exit(1)
	}

	config := fingers.NewConfig()
	if *cyberhubURL != "" {
		if *apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: -key is required when using -url")
			os.Exit(1)
		}
		config.WithProvider(cyberhub.NewProvider(*cyberhubURL, *apiKey))
	}

	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %v\n", err)
		os.Exit(1)
	}

	lib := engine.Get()
	if lib == nil {
		fmt.Fprintln(os.Stderr, "Error: fingers engine is nil")
		os.Exit(1)
	}

	resp, err := http.Get(*target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching target: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	frameworks, err := lib.DetectContent(httputils.ReadRaw(resp))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting fingerprints: %v\n", err)
		os.Exit(1)
	}

	if len(frameworks) == 0 {
		fmt.Println("No fingerprints matched")
		return
	}

	if *jsonOut {
		output, _ := json.MarshalIndent(frameworks, "", "  ")
		fmt.Println(string(output))
		return
	}

	fmt.Printf("Matched %d fingerprint(s):\n", len(frameworks))
	idx := 1
	for _, fw := range frameworks {
		line := fmt.Sprintf("[%d] %s", idx, fw.Name)
		if fw.Version != "" {
			line += fmt.Sprintf(" (%s)", fw.Version)
		}
		if cpe := fw.CPE(); cpe != "" {
			line += fmt.Sprintf(" [%s]", cpe)
		}
		fmt.Println(line)
		idx++
	}

	if lib.Aliases == nil {
		return
	}
	fmt.Println("\nAssociated POCs:")
	for _, fw := range frameworks {
		if a, ok := lib.Aliases.FindFramework(fw); ok && len(a.Pocs) > 0 {
			fmt.Printf("  %s -> %v\n", fw.Name, a.Pocs)
		}
	}
}
