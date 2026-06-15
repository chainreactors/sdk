package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/gogo"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

var (
	cyberhubURL  = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey       = flag.String("key", "", "Cyberhub API Key")
	target       = flag.String("target", "", "Target IP or CIDR (required)")
	ports        = flag.String("ports", "80,443,8080,8443", "Ports to scan")
	threads      = flag.Int("threads", 1000, "Number of threads")
	versionLevel = flag.Int("version", 0, "Version detection level (0-3)")
	jsonOut      = flag.Bool("json", false, "Output as JSON")
)

func main() {
	flag.Parse()

	if *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: gogo -target <ip/cidr> [-ports <ports>] [options]")
		fmt.Fprintln(os.Stderr, "\nWith Cyberhub (auto-loads fingers + neutron):")
		fmt.Fprintln(os.Stderr, "  gogo -url http://127.0.0.1:8080 -key your_key -target 192.168.1.1 -ports 80,443")
		fmt.Fprintln(os.Stderr, "\nWith local data:")
		fmt.Fprintln(os.Stderr, "  gogo -target 192.168.1.0/24 -ports 80,443,8080 -threads 2000 -version 2")
		os.Exit(1)
	}

	config := gogo.NewConfig()
	if *cyberhubURL != "" {
		if *apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: -key is required when using -url")
			os.Exit(1)
		}
		config.WithProvider(cyberhub.NewProvider(*cyberhubURL, *apiKey))
	}

	engine, err := gogo.NewEngine(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %v\n", err)
		os.Exit(1)
	}

	ctx := gogo.NewContext().
		SetThreads(*threads).
		SetVersionLevel(*versionLevel)

	results, err := engine.Scan(ctx, *target, *ports)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		output, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(output))
		return
	}

	for _, r := range results {
		fws := make([]string, 0, len(r.Frameworks))
		for _, fw := range r.Frameworks {
			fws = append(fws, fw.Name)
		}
		line := fmt.Sprintf("%s:%s [%s]", r.Ip, r.Port, r.Status)
		if len(fws) > 0 {
			line += fmt.Sprintf(" %s", strings.Join(fws, ","))
		}
		if r.Title != "" {
			line += fmt.Sprintf(" (%s)", r.Title)
		}
		fmt.Println(line)
	}
	fmt.Printf("\nTotal: %d alive\n", len(results))
}
