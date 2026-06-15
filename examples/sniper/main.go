package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chainreactors/sdk/client"
	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/utils/httputils"
)

var (
	cyberhubURL = flag.String("url", "", "Cyberhub URL (required)")
	apiKey      = flag.String("key", "", "Cyberhub API Key (required)")
	target      = flag.String("target", "", "Target URL (required)")
)

func main() {
	flag.Parse()

	if *cyberhubURL == "" || *apiKey == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: sniper -url <cyberhub_url> -key <api_key> -target <url>")
		fmt.Fprintln(os.Stderr, "\nWorkflow: fingerprint -> lookup associated POCs -> precise attack")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  sniper -url http://127.0.0.1:8080 -key your_key -target http://192.168.1.1:8080")
		os.Exit(1)
	}

	provider := cyberhub.NewProvider(*cyberhubURL, *apiKey)
	c := client.New(
		client.WithProvider(provider),
		client.WithIndex(nil),
	)
	defer c.Close()

	// Step 1: Fingerprint
	fmt.Printf("[*] Step 1: Fingerprinting %s\n", *target)

	fingersEngine, err := c.Fingers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "init fingers failed: %v\n", err)
		os.Exit(1)
	}

	lib := fingersEngine.Get()
	if lib == nil {
		fmt.Fprintln(os.Stderr, "fingers engine is nil")
		os.Exit(1)
	}

	resp, err := http.Get(*target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch target failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	frameworks, err := lib.DetectContent(httputils.ReadRaw(resp))
	if err != nil {
		fmt.Fprintf(os.Stderr, "detect failed: %v\n", err)
		os.Exit(1)
	}

	if len(frameworks) == 0 {
		fmt.Println("[-] No fingerprints matched, nothing to do")
		return
	}

	names := make([]string, 0, len(frameworks))
	for _, fw := range frameworks {
		names = append(names, fw.Name)
		fmt.Printf("    [finger] %s", fw.Name)
		if fw.Version != "" {
			fmt.Printf(" %s", fw.Version)
		}
		fmt.Println()
	}

	// Step 2: Association lookup
	fmt.Printf("\n[*] Step 2: Looking up associated POCs for %d fingerprint(s)\n", len(names))

	result, err := c.LookupByFinger(names...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lookup failed: %v\n", err)
		os.Exit(1)
	}

	if len(result.Templates) == 0 {
		fmt.Println("[-] No associated POCs found")
		return
	}

	for _, t := range result.Templates {
		fmt.Printf("    [poc] %s [%s]\n", t.Id, t.Info.Severity)
	}

	// Step 3: Precise attack
	fmt.Printf("\n[*] Step 3: Executing %d POC(s) against %s\n", len(result.Templates), *target)

	neutronEngine, err := c.Neutron()
	if err != nil {
		fmt.Fprintf(os.Stderr, "init neutron failed: %v\n", err)
		os.Exit(1)
	}

	task := neutron.NewExecuteTask(*target)
	task.Templates = result.Templates

	resultCh, err := neutronEngine.Execute(neutron.NewContext(), task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute failed: %v\n", err)
		os.Exit(1)
	}

	vulnCount := 0
	for r := range resultCh {
		execResult, ok := r.(*neutron.ExecuteResult)
		if !ok || !execResult.Matched() {
			continue
		}
		vulnCount++
		fmt.Printf("    [vuln] %s [%s]\n", execResult.Template().Id, execResult.Template().Info.Severity)
	}

	// Summary
	fmt.Printf("\n[*] Done: %d fingerprint(s), %d POC(s), %d vulnerability(ies) confirmed\n",
		len(frameworks), len(result.Templates), vulnCount)
}
