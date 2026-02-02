package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/fingers"
)

func main() {
	fmt.Println("🧪 Testing Fingers Active Match APIs")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	// 创建 Fingers 引擎
	fmt.Println("🔧 Initializing Fingers engine...")
	config := fingers.NewConfig()
	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Printf("❌ Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Engine initialized (%d fingerprints loaded)\n\n", engine.Count())

	// 测试1: HTTPMatch - HTTP主动探测
	fmt.Println("📋 Test 1: HTTPMatch (HTTP Active Probing)")
	fmt.Println("   Target: http://httpbin.org")
	fmt.Println("   Level: 1 (Basic)")
	fmt.Println("   Timeout: 10s\n")

	results, err := engine.HTTPMatch("http://httpbin.org", 1, 10)
	if err != nil {
		fmt.Printf("❌ HTTPMatch failed: %v\n", err)
	} else {
		fmt.Printf("✓ HTTPMatch completed: %d results\n", len(results))
		for i, result := range results {
			if result.Framework != nil {
				fmt.Printf("  [%d] %s (from: %s)\n", i+1, result.Framework.Name, result.Framework.From)
			}
		}
	}

	fmt.Println()

	// 测试2: ServiceMatch - 通用服务主动探测
	fmt.Println("📋 Test 2: ServiceMatch (Service Active Probing)")
	fmt.Println("   Target: httpbin.org:80")
	fmt.Println("   Level: 1 (Basic)")
	fmt.Println("   Timeout: 10s\n")

	results, err = engine.ServiceMatch("httpbin.org:80", 1, 10)
	if err != nil {
		fmt.Printf("❌ ServiceMatch failed: %v\n", err)
	} else {
		fmt.Printf("✓ ServiceMatch completed: %d results\n", len(results))
		for i, result := range results {
			if result.Framework != nil {
				fmt.Printf("  [%d] %s (from: %s)\n", i+1, result.Framework.Name, result.Framework.From)
			}
		}
	}

	fmt.Println("\n✓ All tests completed")
}
