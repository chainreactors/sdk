package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/pkg/provider"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: active_match <target_url> [cyberhub_url] [api_key]")
		os.Exit(1)
	}
	target := os.Args[1]

	config := fingers.NewConfig()
	if len(os.Args) >= 4 && os.Args[2] != "" {
		config.WithProvider(cyberhub.NewProvider(os.Args[2], os.Args[3]))
	} else {
		config.WithProvider(provider.NewEmbedProvider())
	}

	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init: %v\n", err)
		os.Exit(1)
	}

	ctx := fingers.NewContext().WithLevel(2)
	results, _ := engine.HTTPMatch(ctx, []string{target})

	for _, tr := range results {
		for _, sr := range tr.Results {
			fmt.Printf("%s %s\n", sr.Framework.Name, sr.Framework.Version)
		}
	}
}
