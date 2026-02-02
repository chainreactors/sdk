package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/spray"
)

func main() {
	// 测试URL - 基础URL
	baseURL := "https://nacos.lzfzkj.com"

	// 测试字典 - 包含会产生302重定向的路径
	wordlist := []string{
		"nacos",   // 会302重定向到 /nacos/
		"nacos/",  // 直接访问最终URL
	}

	fmt.Println("🔍 Testing 30x redirect handling in BRUTE mode")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	// 创建 Spray 引擎
	fmt.Println("🔧 Initializing Spray engine...")
	engine := spray.NewEngine(nil)
	if err := engine.Init(); err != nil {
		fmt.Printf("❌ Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Engine initialized\n")

	// 配置上下文 - brute模式
	fmt.Println("⚙️  Configuring spray context (BRUTE mode)...")
	ctx := spray.NewContext().
		SetThreads(1).
		SetTimeout(10).
		SetMod("path").
		SetFinger(true)

	fmt.Printf("   Mode: path (brute)\n")
	fmt.Printf("   Base URL: %s\n", baseURL)
	fmt.Printf("   Threads: 1\n")
	fmt.Printf("   Timeout: 10s\n")
	fmt.Printf("   Finger: enabled\n\n")

	// 创建暴力破解任务
	fmt.Println("🎯 Starting brute force...")
	fmt.Printf("   Testing %d paths\n\n", len(wordlist))
	task := spray.NewBruteTask(baseURL, wordlist)

	// 执行任务
	resultCh, err := engine.Execute(ctx, task)
	if err != nil {
		fmt.Printf("❌ Error executing task: %v\n", err)
		os.Exit(1)
	}

	// 处理结果
	fmt.Println("📊 Results (including all results for debugging):")
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

		// 输出所有结果的详细信息（包括无效的）
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
	fmt.Printf("✓ Brute force completed\n")
	fmt.Printf("   Total results: %d\n", count)
	fmt.Printf("   Valid results: %d\n", validCount)
	fmt.Printf("   Invalid results: %d\n", count-validCount)
}
