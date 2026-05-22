// Example: load 待审核 / 未启用 (non-active) POCs from a Cyberhub backend.
//
// 默认情况下 SDK 只会拉取 status=active 的 POC（向后兼容老用户）。
// 如需加载待审核 / 草稿 / 已禁用的规则，显式通过
// cyberhub.NewExportFilter().WithStatuses(...) / .WithReviewStatus(...) 指定。
//
// 用法:
//
//	pending_pocs -url http://127.0.0.1:8080 -key YOUR_KEY
//	pending_pocs -url ... -key ... -statuses pending,draft
//	pending_pocs -url ... -key ... -review pending
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

var (
	cyberhubURL = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey      = flag.String("key", "", "Cyberhub API Key")
	statuses    = flag.String("statuses", "active,pending,draft",
		"POC 生命周期状态（逗号分隔）：active / pending / draft / inactive / deprecated")
	review = flag.String("review", "",
		"审核流程状态：pending / approved / rejected / draft / none，留空表示不按审核状态过滤")
	preview = flag.Int("preview", 10, "最多打印多少条 POC 摘要")
)

func main() {
	flag.Parse()

	if *cyberhubURL == "" || *apiKey == "" {
		fmt.Println("usage: pending_pocs -url <cyberhub_url> -key <api_key> [-statuses active,pending] [-review pending]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// 1. 构造 ExportFilter，显式声明需要的状态
	filter := cyberhub.NewExportFilter().
		WithStatuses(splitCSV(*statuses)...)
	if *review != "" {
		filter.WithReviewStatus(*review)
	}

	// 2. 挂到 neutron.Config 上（和 examples/filter/main.go 同款用法）
	config := neutron.NewConfig().WithCyberhub(*cyberhubURL, *apiKey)
	config.ExportFilter = filter

	// 3. 创建引擎，触发拉取
	engine, err := neutron.NewEngine(config)
	if err != nil {
		fmt.Printf("create engine failed: %v\n", err)
		os.Exit(1)
	}

	tpls := engine.Get()
	fmt.Printf("加载到 %d 条 POC (statuses=%s review=%q)\n", len(tpls), *statuses, *review)

	limit := *preview
	if limit > len(tpls) {
		limit = len(tpls)
	}
	for i := 0; i < limit; i++ {
		t := tpls[i]
		fmt.Printf("  [%s] %s  severity=%s\n", t.Id, t.Info.Name, t.Info.Severity)
	}
	if len(tpls) > limit {
		fmt.Printf("... (省略 %d 条)\n", len(tpls)-limit)
	}
}

func splitCSV(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
