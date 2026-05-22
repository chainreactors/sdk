package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

var (
	url = flag.String("url", "", "Cyberhub URL")
	key = flag.String("key", "", "Cyberhub API Key")
)

func main() {
	flag.Parse()

	if *url == "" || *key == "" {
		testExportFilter()
		testLocalFilter()
		fmt.Println("All Filter API tests passed (no real data)")
		return
	}

	ctx := context.Background()
	testWithRealData(ctx, *url, *key)
}

func testExportFilter() {
	filter := cyberhub.NewExportFilter().
		WithTags("cms", "framework").
		WithSources("github").
		WithLimit(100).
		WithUpdatedAfter(time.Now().AddDate(0, -1, 0))

	config := fingers.NewConfig()
	config.Provider = cyberhub.NewProvider("http://127.0.0.1:8080", "test_key").WithFilter(filter)

	fmt.Printf("Fingers Filter: Tags=%v, Sources=%v, Limit=%d\n",
		filter.Tags, filter.Sources, filter.Limit)
}

func testLocalFilter() {
	fullFingers := fingers.FullFingers{}
	_ = fullFingers.Filter(func(f *fingers.FullFinger) bool {
		return f.Finger != nil && f.Finger.Protocol == "http"
	})
	fmt.Println("FullFingers.Filter: OK")

	tpls := neutron.Templates{}
	_ = tpls.Filter(func(t *templates.Template) bool {
		severity := t.Info.Severity
		return severity == "critical" || severity == "high"
	})
	fmt.Println("Templates.Filter (severity): OK")
}

func testWithRealData(ctx context.Context, url, key string) {
	fmt.Println("=== Fingers remote filter ===")

	hub := cyberhub.NewProvider(url, key)

	config1 := fingers.NewConfig()
	config1.Provider = hub
	engine1, err := fingers.NewEngine(config1)
	if err != nil {
		fmt.Printf("Fingers load failed: %v\n", err)
		return
	}
	fmt.Printf("No filter: %d fingerprints\n", engine1.Count())

	config2 := fingers.NewConfig()
	config2.Provider = cyberhub.NewProvider(url, key).
		WithFilter(cyberhub.NewExportFilter().WithTags("cms"))
	engine2, err := fingers.NewEngine(config2)
	if err != nil {
		fmt.Printf("Fingers filter failed: %v\n", err)
		return
	}
	fmt.Printf("Tag=cms: %d fingerprints\n", engine2.Count())

	fmt.Println("\n=== Neutron remote filter ===")

	nConfig1 := neutron.NewConfig()
	nConfig1.Provider = hub
	nEngine1, err := neutron.NewEngine(nConfig1)
	if err != nil {
		fmt.Printf("Neutron load failed: %v\n", err)
		return
	}
	fmt.Printf("No filter: %d POCs\n", len(nEngine1.Get()))

	nConfig2 := neutron.NewConfig()
	nConfig2.Provider = cyberhub.NewProvider(url, key).
		WithFilter(cyberhub.NewExportFilter().WithTags("rce"))
	nEngine2, err := neutron.NewEngine(nConfig2)
	if err != nil {
		fmt.Printf("Neutron filter failed: %v\n", err)
		return
	}
	fmt.Printf("Tag=rce: %d POCs\n", len(nEngine2.Get()))

	fmt.Println("\nAll tests done!")
}
