package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/pkg/association"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/pkg/types"
	"github.com/chainreactors/utils/httputils"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: finger_to_poc <cyberhub_url> <api_key> <target_url>")
		os.Exit(1)
	}
	hubURL, apiKey, target := os.Args[1], os.Args[2], os.Args[3]
	ctx := context.Background()

	fingerProvider := cyberhub.NewProvider(hubURL, apiKey)
	fingerProvider.WithTimeout(300 * time.Second)

	pocProvider := cyberhub.NewProvider(hubURL, apiKey)
	pocProvider.WithTimeout(300 * time.Second)
	pocProvider.WithFilter(types.NewExportFilter().WithDraft(true).WithReviewStatus(types.ReviewStatusPending))

	// 1. 指纹引擎
	engine, err := fingers.NewEngine(fingers.NewConfig().WithProvider(fingerProvider))
	if err != nil || engine.Get() == nil {
		fmt.Fprintf(os.Stderr, "init engine: %v\n", err)
		os.Exit(1)
	}

	// 2. 关联索引（指纹 provider + POC provider）
	idx, err := association.BuildFromProvider(ctx, fingerProvider, pocProvider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build index: %v\n", err)
		os.Exit(1)
	}

	// 3. 指纹识别 → 关联 POC
	resp, err := http.Get(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	frameworks, _ := engine.Get().DetectContent(httputils.ReadRaw(resp))
	for _, fw := range frameworks {
		r := idx.Lookup(association.NewQuery().WithFingers(fw.Name))
		fmt.Printf("[%s] cpe=%s  pocs=%d\n", fw.Name, fw.CPE(), len(r.Templates))
		for _, t := range r.Templates {
			fmt.Printf("  -> %s\n", t.Id)
		}
	}
}
