package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/spray"
)

func main() {
	// 测试URL - 使用http协议
	testURL := "http://nacos.lzfzkj.com/nacos"

	fmt.Println("🔍 Testing Nacos with HTTP protocol and active fingerprint")
	fmt.Printf("   Target: %s\n\n", testURL)

	// 创建 Spray 引擎
	fmt.Println("🔧 Initializing Spray engine...")
	engine := spray.NewEngine(nil)
	if err := engine.Init(); err != nil {
		fmt.Printf("❌ Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Engine initialized\n")

	// 配置上下文 - 启用主动指纹识别
	fmt.Println("⚙️  Configuring spray context...")
	ctx := spray.NewContext().
		SetThreads(1).
		SetTimeout(10).
		SetMod("path").
		SetFinger(true).        // 启用指纹识别
		SetActivePlugin(true)   // 启用主动指纹识别

	fmt.Printf("   Mode: path\n")
	fmt.Printf("   Threads: 1\n")
	fmt.Printf("   Timeout: 10s\n")
	fmt.Printf("   Finger: enabled\n")
	fmt.Printf("   Active Fingerprint: enabled\n\n")

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
	fmt.Println("📊 Results (all results including invalid):")
	fmt.Println("─────────────────────────────────────────────────────────────")

	count := 0
	validCount := 0
	for result := range resultCh {
		count++

		sprayResult := result.(*spray.Result).SprayResult()
		if sprayResult == nil {
			fmt.Printf("\n[%d] ⚠️  No spray result data, Error: %v\n", count, result.Error())
			continue
		}

		// 输出所有结果的详细信息
		if result.Success() {
			validCount++
			fmt.Printf("\n[%d] ✅ VALID - URL: %s\n", count, sprayResult.UrlString)
		} else {
			fmt.Printf("\n[%d] ❌ INVALID - URL: %s\n", count, sprayResult.UrlString)
		}

		fmt.Printf("    Status: %d\n", sprayResult.Status)
		fmt.Printf("    Length: %d bytes\n", sprayResult.BodyLength)
		fmt.Printf("    Source: %s\n", sprayResult.Source.Name())
		fmt.Printf("    IsValid: %v\n", sprayResult.IsValid)
		fmt.Printf("    IsFuzzy: %v\n", sprayResult.IsFuzzy)

		if sprayResult.Title != "" {
			fmt.Printf("    Title: %s\n", sprayResult.Title)
		}

		if sprayResult.Reason != "" {
			fmt.Printf("    Reason: %s\n", sprayResult.Reason)
		}

		if sprayResult.ErrString != "" {
			fmt.Printf("    Error: %s\n", sprayResult.ErrString)
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
	fmt.Printf("   Valid results: %d\n", validCount)
	fmt.Printf("   Invalid results: %d\n", count-validCount)
}
