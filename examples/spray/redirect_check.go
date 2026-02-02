package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/spray"
)

func main() {
	// 测试URL - 会产生302重定向
	testURLs := []string{
		"https://nacos.lzfzkj.com/nacos",  // 302 -> /nacos/
		"https://nacos.lzfzkj.com/nacos/", // 200 OK
	}

	fmt.Println("🔍 Testing 30x redirect handling in CHECK mode")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	// 创建 Spray 引擎
	fmt.Println("🔧 Initializing Spray engine...")
	engine := spray.NewEngine(nil)
	if err := engine.Init(); err != nil {
		fmt.Printf("❌ Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Engine initialized\n")

	// 配置上下文 - check模式
	fmt.Println("⚙️  Configuring spray context (CHECK mode)...")
	ctx := spray.NewContext().
		SetThreads(1).
		SetTimeout(10).
		SetMod("path").
		SetFinger(true)

	fmt.Printf("   Mode: path (check)\n")
	fmt.Printf("   Threads: 1\n")
	fmt.Printf("   Timeout: 10s\n")
	fmt.Printf("   Finger: enabled\n\n")

	// 创建检测任务
	fmt.Println("🎯 Starting URL check...")
	fmt.Printf("   Testing %d URLs\n\n", len(testURLs))
	task := spray.NewCheckTask(testURLs)

	// 执行任务
	resultCh, err := engine.Execute(ctx, task)
	if err != nil {
		fmt.Printf("❌ Error executing task: %v\n", err)
		os.Exit(1)
	}

	// 处理结果
	fmt.Println("📊 Results:")
	fmt.Println("─────────────────────────────────────────────────────────────")

	count := 0
	for result := range resultCh {
		count++

		if !result.Success() {
			fmt.Printf("\n❌ Request failed: %v\n", result.Error())
			continue
		}

		sprayResult := result.(*spray.Result).SprayResult()
		if sprayResult == nil {
			fmt.Println("\n⚠️  No spray result data")
			continue
		}

		// 输出详细信息
		fmt.Printf("\n[%d] URL: %s\n", count, sprayResult.UrlString)
		fmt.Printf("    Status: %d\n", sprayResult.Status)
		fmt.Printf("    Length: %d bytes\n", sprayResult.BodyLength)

		if sprayResult.Title != "" {
			fmt.Printf("    Title: %s\n", sprayResult.Title)
		}

		// 重定向信息
		if sprayResult.RedirectURL != "" {
			fmt.Printf("    🔄 Redirect: %s\n", sprayResult.RedirectURL)
		}

		// 指纹信息
		if len(sprayResult.Frameworks) > 0 {
			fmt.Printf("    🔍 Fingerprints: ")
			first := true
			for _, framework := range sprayResult.Frameworks {
				if !first {
					fmt.Printf(", ")
				}
				first = false
				fmt.Printf("%s", framework.Name)
			}
			fmt.Println()
		}
	}

	// 输出统计
	fmt.Println("\n─────────────────────────────────────────────────────────────")
	fmt.Printf("✓ Check completed\n")
	fmt.Printf("   Total results: %d\n", count)
}
