package main

import (
	"fmt"
	"os"

	"github.com/chainreactors/sdk/spray"
)

func main() {
	fmt.Println("🧪 Testing SetHost method")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	// 创建 Spray 引擎
	fmt.Println("🔧 Initializing Spray engine...")
	engine := spray.NewEngine(nil)
	if err := engine.Init(); err != nil {
		fmt.Printf("❌ Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Engine initialized\n")

	// 测试1: 使用SetHost方法
	fmt.Println("📋 Test 1: Using SetHost method")
	fmt.Println("   Target: http://httpbin.org")
	fmt.Println("   Custom Host: example.com\n")

	ctx := spray.NewContext().
		SetThreads(1).
		SetTimeout(10).
		SetHost("example.com")

	task := spray.NewCheckTask([]string{"http://httpbin.org/headers"})
	resultCh, err := engine.Execute(ctx, task)
	if err != nil {
		fmt.Printf("❌ Error executing task: %v\n", err)
		os.Exit(1)
	}

	for result := range resultCh {
		if !result.Success() {
			fmt.Printf("❌ Request failed: %v\n", result.Error())
			continue
		}

		sprayResult := result.(*spray.Result).SprayResult()
		if sprayResult == nil {
			fmt.Println("⚠️  No spray result data")
			continue
		}

		fmt.Printf("✓ Status: %d\n", sprayResult.Status)
		fmt.Printf("  Length: %d bytes\n", sprayResult.BodyLength)
		if sprayResult.Title != "" {
			fmt.Printf("  Title: %s\n", sprayResult.Title)
		}
	}

	fmt.Println("\n✓ Test completed successfully")
}
