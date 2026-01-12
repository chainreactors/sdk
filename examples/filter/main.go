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
		// 只测试 API 编译
		testExportFilter()
		testLocalFilter()
		fmt.Println("所有 Filter API 测试通过（无真实数据）")
		return
	}

	// 真实数据测试
	ctx := context.Background()
	testWithRealData(ctx, *url, *key)
}

// testExportFilter 测试远程筛选 API
func testExportFilter() {
	// 指纹筛选
	filter := cyberhub.NewExportFilter().
		WithTags("cms", "framework").
		WithSources("github").
		WithLimit(100).
		WithUpdatedAfter(time.Now().AddDate(0, -1, 0))

	config := fingers.NewConfig().
		WithCyberhub("http://127.0.0.1:8080", "test_key")
	config.ExportFilter = filter

	fmt.Printf("Fingers ExportFilter: Tags=%v, Sources=%v, Limit=%d\n",
		config.ExportFilter.Tags,
		config.ExportFilter.Sources,
		config.ExportFilter.Limit)

	// POC 筛选
	pocFilter := cyberhub.NewExportFilter().
		WithTags("cve", "rce").
		WithSources("nuclei")

	nConfig := neutron.NewConfig().
		WithCyberhub("http://127.0.0.1:8080", "test_key")
	nConfig.ExportFilter = pocFilter

	fmt.Printf("Neutron ExportFilter: Tags=%v, Sources=%v\n",
		nConfig.ExportFilter.Tags,
		nConfig.ExportFilter.Sources)
}

// testLocalFilter 测试本地筛选 API
func testLocalFilter() {
	fConfig := fingers.NewConfig()
	fConfig.WithFilter(func(f *fingers.FullFinger) bool {
		return f.Finger != nil && f.Finger.Protocol == "http"
	})
	fmt.Println("Fingers WithFilter: OK")

	nConfig := neutron.NewConfig()
	nConfig.WithFilter(func(t *templates.Template) bool {
		severity := t.Info.Severity
		return severity == "critical" || severity == "high"
	})
	fmt.Println("Neutron WithFilter: OK")

	tpls := neutron.Templates{}
	filtered := tpls.Filter(func(t *templates.Template) bool {
		for _, tag := range t.GetTags() {
			if tag == "rce" {
				return true
			}
		}
		return false
	})
	fmt.Printf("Templates.Filter: OK (filtered %d)\n", filtered.Len())
}

// testWithRealData 测试真实数据加载
func testWithRealData(ctx context.Context, url, key string) {
	fmt.Println("=== 测试 Fingers 远程筛选 ===")

	// 测试1: 无筛选，加载全部
	config1 := fingers.NewConfig().WithCyberhub(url, key)
	engine1, err := fingers.NewEngine(config1)
	if err != nil {
		fmt.Printf("❌ Fingers 加载失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 无筛选: %d 个指纹\n", engine1.Count())

	// 测试2: 按标签筛选
	filter2 := cyberhub.NewExportFilter().WithTags("cms")
	config2 := fingers.NewConfig().WithCyberhub(url, key)
	config2.ExportFilter = filter2
	engine2, err := fingers.NewEngine(config2)
	if err != nil {
		fmt.Printf("❌ Fingers 筛选失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 按标签 cms 筛选: %d 个指纹\n", engine2.Count())

	// 测试3: 限制数量
	filter3 := cyberhub.NewExportFilter().WithLimit(10)
	config3 := fingers.NewConfig().WithCyberhub(url, key)
	config3.ExportFilter = filter3
	engine3, err := fingers.NewEngine(config3)
	if err != nil {
		fmt.Printf("❌ Fingers 限制失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 限制 10 个: %d 个指纹\n", engine3.Count())

	fmt.Println("\n=== 测试 Neutron 远程筛选 ===")

	// 测试4: 无筛选
	nConfig1 := neutron.NewConfig().WithCyberhub(url, key)
	nEngine1, err := neutron.NewEngine(nConfig1)
	if err != nil {
		fmt.Printf("❌ Neutron 加载失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 无筛选: %d 个 POC\n", len(nEngine1.Get()))

	// 测试5: 按标签筛选
	nFilter2 := cyberhub.NewExportFilter().WithTags("rce")
	nConfig2 := neutron.NewConfig().WithCyberhub(url, key)
	nConfig2.ExportFilter = nFilter2
	nEngine2, err := neutron.NewEngine(nConfig2)
	if err != nil {
		fmt.Printf("❌ Neutron 筛选失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 按标签 rce 筛选: %d 个 POC\n", len(nEngine2.Get()))

	// 测试6: 限制数量
	nFilter3 := cyberhub.NewExportFilter().WithLimit(20)
	nConfig3 := neutron.NewConfig().WithCyberhub(url, key)
	nConfig3.ExportFilter = nFilter3
	nEngine3, err := neutron.NewEngine(nConfig3)
	if err != nil {
		fmt.Printf("❌ Neutron 限制失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 限制 20 个: %d 个 POC\n", len(nEngine3.Get()))

	fmt.Println("\n=== 测试本地筛选 ===")

	// 测试7: 加载后再筛选
	allTemplates := (neutron.Templates{}).Merge(nEngine1.Get())
	highSeverity := allTemplates.Filter(func(t *templates.Template) bool {
		return t.Info.Severity == "critical" || t.Info.Severity == "high"
	})
	fmt.Printf("✅ 从 %d 个 POC 中筛选 critical/high: %d 个\n", allTemplates.Len(), highSeverity.Len())

	fmt.Println("\n所有测试完成!")
}
