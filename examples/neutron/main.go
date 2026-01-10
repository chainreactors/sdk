package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/neutron/protocols"
	neutronTemplates "github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/sdk/neutron"
)

var (
	// Cyberhub 配置
	cyberhubURL = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey      = flag.String("key", "", "Cyberhub API Key")
	source      = flag.String("source", "", "Filter by source (optional)")

	// 本地配置
	localPath = flag.String("path", "", "Local POC directory or file")

	// 扫描选项
	target     = flag.String("target", "", "Target URL to scan")
	pocID      = flag.String("poc", "", "Specific POC ID to execute (optional)")
	listPOCs   = flag.Bool("list", false, "List all loaded POCs")
	jsonOut    = flag.Bool("json", false, "Output as JSON")
	timeout    = flag.Int("timeout", 10, "Request timeout in seconds")
	maxPOCs    = flag.Int("max", 0, "Maximum number of POCs to execute (0 = all)")
	severities = flag.String("severity", "", "Filter by severity (comma separated: info,low,medium,high,critical)")
	tags       = flag.String("tags", "", "Filter by tags (comma separated)")
)

func main() {
	flag.Parse()

	// 验证参数
	if *cyberhubURL == "" && *localPath == "" {
		fmt.Println("Usage: neutron [-url <cyberhub_url> -key <api_key>] [-path <local_path>] -target <url>")
		fmt.Println("\nLoad from Cyberhub:")
		fmt.Println("  neutron -url http://127.0.0.1:8080 -key your_key -target http://example.com")
		fmt.Println("\nLoad from local:")
		fmt.Println("  neutron -path ./pocs -target http://example.com")
		fmt.Println("\nFilter options:")
		fmt.Println("  neutron -url ... -target ... -source github -severity critical -tags cve,rce")
		fmt.Println("\nList POCs:")
		fmt.Println("  neutron -url ... -list")
		fmt.Println("\nExecute specific POC:")
		fmt.Println("  neutron -url ... -target ... -poc CVE-2021-12345")
		os.Exit(1)
	}

	ctx := context.Background()

	// 1. 加载 Neutron POCs
	var engine *neutron.Engine
	var err error

	config := neutron.NewConfig()

	if *cyberhubURL != "" {
		if *apiKey == "" {
			fmt.Println("Error: -key is required when using -url")
			os.Exit(1)
		}
		config.WithCyberhub(*cyberhubURL, *apiKey)
		if *source != "" {
			config.SetSources(*source)
		}
		fmt.Printf("Loading POCs from Cyberhub (%s)...\n", *cyberhubURL)
	} else if *localPath != "" {
		config.WithLocalFile(*localPath)
		fmt.Printf("Loading POCs from local (%s)...\n", *localPath)
	}

	engine, err = neutron.NewEngine(config)
	if err != nil {
		fmt.Printf("Error creating engine: %v\n", err)
		os.Exit(1)
	}

	if err := config.Load(ctx); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	templates, err := engine.Load(ctx)
	if err != nil {
		fmt.Printf("Error loading POCs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Loaded and compiled %d POC(s)\n\n", len(templates))

	// 2. 过滤 POCs
	filteredTemplates := filterTemplates(templates, *severities, *tags, *pocID)
	if len(filteredTemplates) < len(templates) {
		fmt.Printf("📋 After filtering: %d POC(s)\n\n", len(filteredTemplates))
	}

	// 3. 列出 POCs
	if *listPOCs {
		listPOCsFunc(filteredTemplates, *jsonOut)
		os.Exit(0)
	}

	// 4. 验证 target
	if *target == "" {
		fmt.Println("Error: -target is required for scanning")
		os.Exit(1)
	}

	// 5. 执行扫描
	if *maxPOCs > 0 && *maxPOCs < len(filteredTemplates) {
		filteredTemplates = filteredTemplates[:*maxPOCs]
		fmt.Printf("🔍 Executing first %d POC(s) against: %s\n\n", *maxPOCs, *target)
	} else {
		fmt.Printf("🔍 Executing %d POC(s) against: %s\n\n", len(filteredTemplates), *target)
	}

	matchedPOCs := []map[string]interface{}{}
	matchCount := 0

	for i, t := range filteredTemplates {
		if !*jsonOut && i%10 == 0 && i > 0 {
			fmt.Printf("  Progress: %d/%d\n", i, len(filteredTemplates))
		}

		result, err := t.Execute(*target, nil)
		if err != nil {
			if err == protocols.OpsecError {
				continue // Skip opsec POCs silently
			}
			if !*jsonOut {
				fmt.Printf("⚠️  Error executing %s: %v\n", t.Id, err)
			}
			continue
		}

		if result != nil && result.Matched {
			matchCount++
			if !*jsonOut {
				fmt.Printf("🎯 [%d] %s - %s (Severity: %s)\n", matchCount, t.Id, t.Info.Name, t.Info.Severity)
			}
			matchedPOCs = append(matchedPOCs, map[string]interface{}{
				"id":          t.Id,
				"name":        t.Info.Name,
				"severity":    t.Info.Severity,
				"description": t.Info.Description,
				"tags":        t.GetTags(),
			})
		}
	}

	// 6. 输出结果
	fmt.Println("\n========================================")
	if *jsonOut {
		output := map[string]interface{}{
			"target":        *target,
			"total_pocs":    len(filteredTemplates),
			"matched_count": matchCount,
			"matched_pocs":  matchedPOCs,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("📊 Results:\n")
		fmt.Printf("  Total POCs: %d\n", len(filteredTemplates))
		fmt.Printf("  Matched: %d\n", matchCount)
		fmt.Println("========================================")
	}

	if matchCount > 0 {
		os.Exit(1) // Exit with error if vulnerabilities found
	}
}

func filterTemplates(templates []*neutronTemplates.Template, severities, tags, pocID string) []*neutronTemplates.Template {
	if severities == "" && tags == "" && pocID == "" {
		return templates
	}

	var filtered []*neutronTemplates.Template

	// Parse filters
	severityList := []string{}
	if severities != "" {
		severityList = strings.Split(severities, ",")
	}

	tagList := []string{}
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}

	for _, t := range templates {
		// Filter by POC ID
		if pocID != "" {
			if strings.ToLower(t.Id) == strings.ToLower(pocID) {
				filtered = append(filtered, t)
			}
			continue
		}

		// Filter by severity
		if len(severityList) > 0 {
			matched := false
			for _, sev := range severityList {
				if strings.ToLower(t.Info.Severity) == strings.ToLower(strings.TrimSpace(sev)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Filter by tags
		if len(tagList) > 0 {
			matched := false
			templateTags := t.GetTags()
			for _, filterTag := range tagList {
				for _, tTag := range templateTags {
					if strings.Contains(strings.ToLower(tTag), strings.ToLower(strings.TrimSpace(filterTag))) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		filtered = append(filtered, t)
	}

	return filtered
}

func listPOCsFunc(templates []*neutronTemplates.Template, jsonOut bool) {
	if jsonOut {
		output := []map[string]interface{}{}
		for _, t := range templates {
			output = append(output, map[string]interface{}{
				"id":          t.Id,
				"name":        t.Info.Name,
				"severity":    t.Info.Severity,
				"description": t.Info.Description,
				"tags":        t.GetTags(),
			})
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("📋 Available POCs (%d total):\n\n", len(templates))
		for i, t := range templates {
			fmt.Printf("[%d] %s\n", i+1, t.Id)
			fmt.Printf("    Name: %s\n", t.Info.Name)
			fmt.Printf("    Severity: %s\n", t.Info.Severity)
			if len(t.GetTags()) > 0 {
				fmt.Printf("    Tags: %s\n", strings.Join(t.GetTags(), ", "))
			}
			fmt.Println()
		}
	}
}
