package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/spray"
)

func main() {
	// 测试 URL - 直接访问最终的重定向URL
	testURL := "https://nacos.lzfzkj.com/nacos/"

	fmt.Println("🔍 Testing Nacos URL redirect and fingerprint detection")
	fmt.Printf("   Target: %s\n\n", testURL)

	// 创建 Spray 引擎
	fmt.Println("🔧 Initializing Spray engine...")
	engine := spray.NewEngine(nil)
	if err := engine.Init(); err != nil {
		fmt.Printf("❌ Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Engine initialized\n")

	// 配置上下文
	fmt.Println("⚙️  Configuring spray context...")
	ctx := spray.NewContext().
		SetThreads(1).      // 单线程检测
		SetTimeout(10).     // 10秒超时
		SetMod("path").     // path 模式
		SetFinger(true)     // 启用指纹识别

	fmt.Printf("   Mode: path\n")
	fmt.Printf("   Threads: 1\n")
	fmt.Printf("   Timeout: 10s\n")
	fmt.Printf("   Finger: enabled\n\n")

	// 创建检测任务
	fmt.Println("🎯 Starting URL check...")
	task := spray.NewCheckTask([]string{testURL})

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
			fmt.Printf("❌ Request failed: %v\n", result.Error())
			continue
		}

		sprayResult := result.(*spray.Result).SprayResult()
		if sprayResult == nil {
			fmt.Println("⚠️  No spray result data")
			continue
		}

		// 输出详细信息
		fmt.Printf("\n✓ URL: %s\n", sprayResult.UrlString)
		fmt.Printf("  Status: %d\n", sprayResult.Status)
		fmt.Printf("  Length: %d bytes\n", sprayResult.BodyLength)

		if sprayResult.Title != "" {
			fmt.Printf("  Title: %s\n", sprayResult.Title)
		}

		// 重定向信息
		if sprayResult.RedirectURL != "" {
			fmt.Printf("  🔄 Redirect: %s\n", sprayResult.RedirectURL)
		}

		// 指纹信息
		if len(sprayResult.Frameworks) > 0 {
			fmt.Printf("  🔍 Fingerprints detected:\n")
			for _, framework := range sprayResult.Frameworks {
				fmt.Printf("     - %s", framework.Name)
				if framework.Version != "" {
					fmt.Printf(" (version: %s)", framework.Version)
				}
				fmt.Println()
			}
		} else {
			fmt.Println("  ⚠️  No fingerprints detected")
		}

		// 其他技术栈信息
		if len(sprayResult.Extracteds) > 0 {
			fmt.Printf("  📋 Extracted info:\n")
			for _, extracted := range sprayResult.Extracteds {
				fmt.Printf("     %s: %v\n", extracted.Name, extracted.ExtractResult)
			}
		}
	}

	// 输出统计
	fmt.Println("\n─────────────────────────────────────────────────────────────")
	fmt.Printf("✓ Check completed\n")
	fmt.Printf("   Total results: %d\n", count)
}
