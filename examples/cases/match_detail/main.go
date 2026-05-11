// match_detail 演示：在 sdk/fingers 上拿到指纹命中的 matcher 详情 + 命中的资源 URL。
//
// 关键点：
//  1. SDK 不需要任何改造，只在 NewEngine() 之后翻开 MatchDetail 开关即可。
//     原因：NewEngine 内部会调用 engine.Compile()，把每条 finger 的
//     EnableMatchDetail 重置回 engine 字段的默认值 (false)。
//  2. 命中后直接读 common.Framework.MatchDetail，没有额外封装。
//  3. MatchDetail.SendData 在 active 探测增强后的链路下形如
//     "scope=... method=... url=..."；SDK 自带的被动匹配下通常为空，
//     此时 match_url 用最终请求 URL 兜底。
//
// 用法：
//   go run ./examples/cases/match_detail -url http://127.0.0.1:8080 -key <api_key> -target http://example.com
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/utils/httputils"
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

	// 2. ★ 关键：NewEngine 之后翻开 MatchDetail 开关
	if fe, _ := eng.GetFingersEngine(); fe != nil {
		fe.EnableMatchDetail()
	}

	// 3. 抓 + 匹配
	resp, err := http.Get(*target)
	if err != nil {
		fmt.Printf("http get failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	frameworks, err := eng.Get().DetectContent(httputils.ReadRaw(resp))
	if err != nil {
		fmt.Printf("match failed: %v\n", err)
		os.Exit(1)
	}

	// 4. 直接读 common.Framework.MatchDetail；match_url 用最终请求 URL 兜底
	finalURL := *target
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	if len(frameworks) == 0 {
		fmt.Println("no fingerprints matched")
		return
	}
	for _, fw := range frameworks {
		fmt.Printf("[%s]\n", fw.Name)
		d := fw.MatchDetail
		if d == nil {
			fmt.Printf("  match_url     : %s (MatchDetail empty)\n", finalURL)
			continue
		}
		matchURL := finalURL
		if u := extractURL(d.SendData); u != "" {
			matchURL = u
		}
		fmt.Printf("  match_url     : %s\n", matchURL)
		fmt.Printf("  matcher_type  : %s\n", d.MatcherType)
		fmt.Printf("  matcher_value : %s\n", d.MatcherValue)
		fmt.Printf("  rule_index    : %d\n", d.RuleIndex)
		if d.SendData != "" {
			fmt.Printf("  send_data     : %s\n", d.SendData)
		}
	}
}

// extractURL 从 "scope=... method=... url=<...>" 里取 url= 后整段。
// 词边界判断避免 value 里出现 "url=" 子串时误匹配。
func extractURL(s string) string {
	const tag = "url="
	for start := 0; start < len(s); {
		i := strings.Index(s[start:], tag)
		if i < 0 {
			return ""
		}
		i += start
		if i == 0 || s[i-1] == ' ' {
			return strings.TrimSpace(s[i+len(tag):])
		}
		start = i + len(tag)
	}
	return ""
}
