package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/pkg/types"
)

var (
	hubURL   = flag.String("url", "", "Cyberhub URL")
	apiKey   = flag.String("key", "", "Cyberhub API key")
	tags     = flag.String("tags", "", "Comma-separated tags filter")
	sources  = flag.String("source", "", "Comma-separated sources filter")
	severity = flag.String("severity", "", "Comma-separated severity filter")
	limit    = flag.Int("limit", 20, "Maximum records returned")
	timeout  = flag.Duration("timeout", 15*time.Second, "Request timeout")
)

func main() {
	flag.Parse()

	if *hubURL == "" || *apiKey == "" {
		testLocalFilter()
		fmt.Println("\nPass -url and -key for remote provider + filter demo")
		return
	}

	ctx := context.Background()

	// 1. Remote ExportFilter: server-side filtering before download
	fmt.Println("=== Remote ExportFilter ===")
	filter := types.NewExportFilter().
		WithTags(splitCSV(*tags)...).
		WithSources(splitCSV(*sources)...).
		WithSeverities(splitCSV(*severity)...).
		WithLimit(*limit)

	hub := cyberhub.NewProvider(*hubURL, *apiKey).
		WithTimeout(*timeout).
		WithFilter(filter)

	fgs, aliases, err := hub.Fingers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load fingers failed: %v\n", err)
		os.Exit(1)
	}
	templates, err := hub.POCs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load pocs failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Fingers: %d | Aliases: %d | POCs: %d\n", len(fgs), len(aliases), len(templates))

	for i, f := range fgs {
		if i >= 3 || f == nil {
			break
		}
		fmt.Printf("  [finger] %s [%s]\n", f.Name, f.Protocol)
	}
	for i, t := range templates {
		if i >= 3 || t == nil {
			break
		}
		fmt.Printf("  [poc] %s [%s] %s\n", t.Id, t.Info.Severity, t.Info.Name)
	}

	// 2. Local predicate filter (post-load)
	fmt.Println("\n=== Local Filter ===")
	testLocalFilter()
}

func testLocalFilter() {
	// ExportFilter construction (dry-run, no server call)
	filter := types.NewExportFilter().
		WithTags("cms", "framework").
		WithSources("github").
		WithLimit(100).
		WithUpdatedAfter(time.Now().AddDate(0, -1, 0))
	fmt.Printf("Fingers ExportFilter: Tags=%v, Sources=%v, Limit=%d\n",
		filter.Tags, filter.Sources, filter.Limit)

	// FullFingers.Filter by predicate
	fullFingers := fingers.FullFingers{}
	_ = fullFingers.Filter(func(f *fingers.FullFinger) bool {
		return f.Finger != nil && f.Finger.Protocol == "http"
	})
	fmt.Println("FullFingers.Filter: OK")

	// Templates.Filter by severity
	tpls := neutron.Templates{}
	_ = tpls.Filter(func(t *types.Template) bool {
		return t.Info.Severity == "critical" || t.Info.Severity == "high"
	})
	fmt.Println("Templates.Filter (severity): OK")
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if p := strings.TrimSpace(part); p != "" {
			result = append(result, p)
		}
	}
	return result
}
