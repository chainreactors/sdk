package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/pkg/provider"
	"github.com/chainreactors/sdk/pkg/types"
)

var (
	cyberhubURL = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey      = flag.String("key", "", "Cyberhub API Key")
	localPath   = flag.String("path", "", "Local POC directory or file")

	target     = flag.String("target", "", "Target URL to scan")
	pocID      = flag.String("poc", "", "Specific POC ID to execute")
	listPOCs   = flag.Bool("list", false, "List all loaded POCs")
	jsonOut    = flag.Bool("json", false, "Output as JSON")
	maxPOCs    = flag.Int("max", 0, "Maximum number of POCs to execute (0 = all)")
	severities = flag.String("severity", "", "Filter by severity (comma separated: info,low,medium,high,critical)")
	tagsFlag   = flag.String("tags", "", "Filter by tags (comma separated)")
)

func main() {
	flag.Parse()

	if *cyberhubURL == "" && *localPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: neutron [-url <cyberhub_url> -key <api_key>] [-path <local_path>] -target <url>")
		fmt.Fprintln(os.Stderr, "\nLoad from Cyberhub:")
		fmt.Fprintln(os.Stderr, "  neutron -url http://127.0.0.1:8080 -key your_key -target http://example.com")
		fmt.Fprintln(os.Stderr, "\nLoad from local:")
		fmt.Fprintln(os.Stderr, "  neutron -path ./pocs -target http://example.com")
		fmt.Fprintln(os.Stderr, "\nFilter:")
		fmt.Fprintln(os.Stderr, "  neutron -url ... -target ... -severity critical -tags cve,rce")
		fmt.Fprintln(os.Stderr, "\nList POCs:")
		fmt.Fprintln(os.Stderr, "  neutron -url ... -list")
		os.Exit(1)
	}

	config := neutron.NewConfig()
	if *cyberhubURL != "" {
		if *apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: -key is required when using -url")
			os.Exit(1)
		}
		config.WithProvider(cyberhub.NewProvider(*cyberhubURL, *apiKey))
	} else if *localPath != "" {
		config.WithProvider(provider.NewFileProvider("", *localPath))
	}

	engine, err := neutron.NewEngine(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %v\n", err)
		os.Exit(1)
	}

	templates := filterTemplates(engine.Get(), *severities, *tagsFlag, *pocID)
	fmt.Fprintf(os.Stderr, "Loaded %d POC(s)\n", len(templates))

	if *listPOCs {
		listPOCsFunc(templates, *jsonOut)
		return
	}

	if *target == "" {
		fmt.Fprintln(os.Stderr, "Error: -target is required for scanning")
		os.Exit(1)
	}

	if *maxPOCs > 0 && *maxPOCs < len(templates) {
		templates = templates[:*maxPOCs]
	}
	fmt.Fprintf(os.Stderr, "Executing %d POC(s) against: %s\n\n", len(templates), *target)

	var matchedPOCs []map[string]interface{}
	matchCount := 0

	for _, t := range templates {
		result, err := t.Execute(*target, nil)
		if err != nil {
			if err == types.OpsecError {
				continue
			}
			fmt.Fprintf(os.Stderr, "Error executing %s: %v\n", t.Id, err)
			continue
		}

		if result != nil && result.Matched {
			matchCount++
			if !*jsonOut {
				fmt.Printf("[vuln] %s - %s [%s]\n", t.Id, t.Info.Name, t.Info.Severity)
			}
			matchedPOCs = append(matchedPOCs, map[string]interface{}{
				"id":       t.Id,
				"name":     t.Info.Name,
				"severity": t.Info.Severity,
				"tags":     t.GetTags(),
			})
		}
	}

	if *jsonOut {
		output := map[string]interface{}{
			"target":        *target,
			"total_pocs":    len(templates),
			"matched_count": matchCount,
			"matched_pocs":  matchedPOCs,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("\nTotal POCs: %d | Matched: %d\n", len(templates), matchCount)
	}

	if matchCount > 0 {
		os.Exit(1)
	}
}

func filterTemplates(templates []*types.Template, severities, tags, pocID string) []*types.Template {
	if severities == "" && tags == "" && pocID == "" {
		return templates
	}

	var filtered []*types.Template
	severityList := splitIfNotEmpty(severities)
	tagList := splitIfNotEmpty(tags)

	for _, t := range templates {
		if pocID != "" {
			if strings.EqualFold(t.Id, pocID) {
				filtered = append(filtered, t)
			}
			continue
		}

		if len(severityList) > 0 && !containsFold(severityList, t.Info.Severity) {
			continue
		}
		if len(tagList) > 0 && !matchesTags(t.GetTags(), tagList) {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

func listPOCsFunc(templates []*types.Template, jsonOut bool) {
	if jsonOut {
		var output []map[string]interface{}
		for _, t := range templates {
			output = append(output, map[string]interface{}{
				"id":       t.Id,
				"name":     t.Info.Name,
				"severity": t.Info.Severity,
				"tags":     t.GetTags(),
			})
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		for i, t := range templates {
			fmt.Printf("[%d] %s  %s  [%s]\n", i+1, t.Id, t.Info.Name, t.Info.Severity)
		}
	}
}

func splitIfNotEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func containsFold(list []string, s string) bool {
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), s) {
			return true
		}
	}
	return false
}

func matchesTags(templateTags, filterTags []string) bool {
	for _, ft := range filterTags {
		for _, tt := range templateTags {
			if strings.Contains(strings.ToLower(tt), strings.ToLower(strings.TrimSpace(ft))) {
				return true
			}
		}
	}
	return false
}
