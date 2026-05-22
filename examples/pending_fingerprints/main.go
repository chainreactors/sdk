// Example: load 待审核 / 草稿 / 未启用 (non-active) fingerprints from a Cyberhub backend.
//
// 与 examples/pending_pocs 的区别：
//
//   - POC 导出：SDK 默认强制 status=active；要拉 pending/draft，必须显式 WithStatuses(...)。
//   - 指纹导出：SDK 不强制状态，后端默认就会返回 active + 非空 pending + inactive + deprecated；
//     但 draft 和"raw_content 为空的 pending"会被后端 shouldHideDraftOnlyFingerprints
//     规则隐掉。如果客户端要拿到这部分"空壳待审核"指纹，仍需显式
//     WithStatuses("pending") / WithStatuses("draft") 等。
//
// 用法:
//
//	pending_fingerprints -url http://127.0.0.1:8080 -key YOUR_KEY
//	pending_fingerprints -url ... -key ... -statuses active,pending,draft
//	pending_fingerprints -url ... -key ... -review pending
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

var (
	cyberhubURL = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey      = flag.String("key", "", "Cyberhub API Key")
	statuses    = flag.String("statuses", "",
		"指纹生命周期状态（逗号分隔）：active / pending / draft / inactive / deprecated；留空走后端默认（不含 draft 和空 pending）")
	review = flag.String("review", "",
		"审核流程状态：pending / approved / rejected / draft / none，留空表示不按审核状态过滤")
	preview = flag.Int("preview", 10, "最多打印多少条指纹摘要")
)

func main() {
	flag.Parse()

	if *cyberhubURL == "" || *apiKey == "" {
		fmt.Println("usage: pending_fingerprints -url <cyberhub_url> -key <api_key> [-statuses active,pending] [-review pending]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// 1. 构造 ExportFilter；不调 WithStatuses 时走后端默认语义。
	filter := cyberhub.NewExportFilter()
	if list := splitCSV(*statuses); len(list) > 0 {
		filter.WithStatuses(list...)
	}
	if *review != "" {
		filter.WithReviewStatus(*review)
	}

	// 2. 挂到 fingers.Config 上（和 POC 路径完全对称）
	config := fingers.NewConfig().WithCyberhub(*cyberhubURL, *apiKey)
	config.ExportFilter = filter

	// 3. 创建引擎，触发拉取
	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Printf("create engine failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("加载到 %d 条指纹 (statuses=%q review=%q)\n", engine.Count(), *statuses, *review)

	items := config.FullFingers.Fingers()
	limit := *preview
	if limit > len(items) {
		limit = len(items)
	}
	for i := 0; i < limit; i++ {
		f := items[i]
		fmt.Printf("  [%s] protocol=%s tags=%v\n", f.Name, f.Protocol, f.Tags)
	}
	if len(items) > limit {
		fmt.Printf("... (省略 %d 条)\n", len(items)-limit)
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
