// match_detail 演示：在 sdk/fingers 上直接拿到指纹命中的 matcher 详情 + 命中的资源 URL。
//
// 关键点：
//  1. 调用 MatchHTTPWithDetail(resp)，SDK 会自动打开 MatchDetail。
//  2. 结果里直接读 MatchResult.MatchURL / MatcherType / MatcherValue。
//  3. match_url 取值优先级：MatchDetail.SendData 中的 "url=" > 当前请求 URL。
//
// 用法：
//
//	go run ./examples/cases/match_detail -url http://127.0.0.1:8080 -key <api_key> -target http://example.com
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/chainreactors/sdk/fingers"
)

func main() {
	cyberhubURL := flag.String("url", "", "Cyberhub URL (optional, local fingers used if empty)")
	apiKey := flag.String("key", "", "Cyberhub API Key")
	target := flag.String("target", "", "Target URL to match (required)")
	flag.Parse()
	if *target == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 1. 构建 engine
	cfg := fingers.NewConfig()
	if *cyberhubURL != "" {
		cfg.WithCyberhub(*cyberhubURL, *apiKey)
	}
	eng, err := fingers.NewEngine(cfg)
	if err != nil {
		fmt.Printf("engine init failed: %v\n", err)
		os.Exit(1)
	}

	// 2. 抓 + 匹配。MatchHTTPWithDetail 会自动打开 MatchDetail。
	resp, err := http.Get(*target)
	if err != nil {
		fmt.Printf("http get failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	results, err := eng.MatchHTTPWithDetail(resp)
	if err != nil {
		fmt.Printf("match failed: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("no fingerprints matched")
		return
	}
	for _, r := range results {
		name := ""
		if r.Framework != nil {
			name = r.Framework.Name
		}
		fmt.Printf("[%s]\n", name)
		fmt.Printf("  match_url     : %s\n", r.MatchURL)
		fmt.Printf("  matcher_type  : %s\n", r.MatcherType)
		fmt.Printf("  matcher_value : %s\n", r.MatcherValue)
		fmt.Printf("  rule_index    : %d\n", r.RuleIndex)
		if r.SendData != "" {
			fmt.Printf("  send_data     : %s\n", r.SendData)
		}
	}
}
