package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/pkg/types"
	"github.com/chainreactors/sdk/spray"
)

var (
	targetURL  = flag.String("u", "", "Single target URL")
	targetFile = flag.String("f", "", "File containing target URLs (one per line)")

	threads = flag.Int("threads", 50, "Number of threads")
	timeout = flag.Int("timeout", 10, "Request timeout in seconds")

	method  = flag.String("method", "GET", "HTTP method (GET/POST/HEAD)")
	headers = flag.String("headers", "", "Custom headers (format: 'Key1:Value1,Key2:Value2')")

	filterStatus = flag.String("fc", "", "Filter by status codes (comma separated)")
	matchStatus  = flag.String("mc", "", "Match status codes (comma separated)")

	jsonOut    = flag.Bool("json", false, "Output as JSON")
	quiet      = flag.Bool("q", false, "Quiet mode (only show matched URLs)")
	outputFile = flag.String("o", "", "Output file")
)

func main() {
	flag.Parse()

	if *targetURL == "" && *targetFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: spray [-u <url> | -f <file>] [options]")
		fmt.Fprintln(os.Stderr, "\nSingle URL:")
		fmt.Fprintln(os.Stderr, "  spray -u http://example.com")
		fmt.Fprintln(os.Stderr, "\nMultiple URLs from file:")
		fmt.Fprintln(os.Stderr, "  spray -f urls.txt -threads 100 -timeout 5 -mc 200,301,302")
		fmt.Fprintln(os.Stderr, "\nCustom headers:")
		fmt.Fprintln(os.Stderr, "  spray -u http://example.com -headers 'Authorization:Bearer token,X-Custom:value'")
		os.Exit(1)
	}

	var urls []string
	if *targetURL != "" {
		urls = append(urls, *targetURL)
	}
	if *targetFile != "" {
		fileUrls, err := readURLsFromFile(*targetFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		urls = append(urls, fileUrls...)
	}
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No URLs to scan")
		os.Exit(1)
	}

	sprayEngine, err := spray.NewEngine(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %v\n", err)
		os.Exit(1)
	}

	sprayCtx := spray.NewContext().
		SetThreads(*threads).
		SetTimeout(*timeout).
		SetMethod(*method)
	if *headers != "" {
		sprayCtx.SetHeaders(strings.Split(*headers, ","))
	}

	resultCh, err := sprayEngine.Execute(sprayCtx, spray.NewCheckTask(urls))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing scan: %v\n", err)
		os.Exit(1)
	}

	var filterStatusCodes, matchStatusCodes []string
	if *filterStatus != "" {
		filterStatusCodes = strings.Split(*filterStatus, ",")
	}
	if *matchStatus != "" {
		matchStatusCodes = strings.Split(*matchStatus, ",")
	}

	var outputWriter *bufio.Writer
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		outputWriter = bufio.NewWriter(f)
		defer outputWriter.Flush()
	}

	var results []map[string]interface{}
	totalCount, matchedCount := 0, 0

	for result := range resultCh {
		totalCount++
		if !result.Success() {
			continue
		}

		sprayResult, ok := types.ResultData[*types.SprayResult](result)
		if !ok || sprayResult == nil {
			continue
		}

		statusStr := fmt.Sprintf("%d", sprayResult.Status)
		if shouldFilter(statusStr, filterStatusCodes) || !shouldMatch(statusStr, matchStatusCodes) {
			continue
		}

		matchedCount++
		resultMap := map[string]interface{}{
			"url":    sprayResult.UrlString,
			"status": sprayResult.Status,
		}
		if sprayResult.Title != "" {
			resultMap["title"] = sprayResult.Title
		}
		results = append(results, resultMap)

		if !*jsonOut {
			var output string
			if *quiet {
				output = sprayResult.UrlString
			} else {
				output = fmt.Sprintf("[%d] %s", sprayResult.Status, sprayResult.UrlString)
				if sprayResult.Title != "" {
					output += fmt.Sprintf(" - %s", sprayResult.Title)
				}
			}
			fmt.Println(output)
			if outputWriter != nil {
				outputWriter.WriteString(output + "\n")
			}
		}
	}

	if *jsonOut {
		output := map[string]interface{}{
			"total_urls":    len(urls),
			"processed":     totalCount,
			"matched_count": matchedCount,
			"results":       results,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else if !*quiet {
		fmt.Printf("\nTotal: %d | Processed: %d | Matched: %d\n", len(urls), totalCount, matchedCount)
	}
}

func shouldFilter(status string, codes []string) bool {
	for _, c := range codes {
		if strings.TrimSpace(c) == status {
			return true
		}
	}
	return false
}

func shouldMatch(status string, codes []string) bool {
	if len(codes) == 0 {
		return true
	}
	for _, c := range codes {
		if strings.TrimSpace(c) == status {
			return true
		}
	}
	return false
}

func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}
