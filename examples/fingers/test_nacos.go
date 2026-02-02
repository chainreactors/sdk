package main

import (
	"context"
	"fmt"
	"os"

	"github.com/chainreactors/sdk/fingers"
)

func main() {
	fmt.Println("🧪 Testing Nacos Fingerprint Detection")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	// 创建临时的 nacos 指纹文件（注意：必须是数组格式）
	nacosFingerprint := `- name: nacos
  protocol: http
  focus: true
  tag:
  - nacos
  send_data: /nacos/
  rule:
  - regexps:
      body:
      - <title>Nacos</title>
  - favicon:
      mmh3:
          - "13942501"
    send_data: /nacos/console-ui/public/img/nacos-logo.png
`

	// 写入临时文件
	tmpFile := "nacos_finger.yaml"
	if err := os.WriteFile(tmpFile, []byte(nacosFingerprint), 0644); err != nil {
		fmt.Printf("❌ Failed to write fingerprint file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	fmt.Println("📝 Created temporary nacos fingerprint file")

	// 验证文件内容
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		fmt.Printf("❌ Failed to read fingerprint file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📄 File content (%d bytes):\n%s\n", len(content), string(content))

	// 创建配置，使用本地文件
	config := fingers.NewConfig().WithLocalFile(tmpFile)
	fmt.Println("📋 Config created with local file")

	// 创建 engine
	fmt.Println("🔧 Initializing Fingers engine...")
	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Printf("❌ Failed to create engine: %v\n", err)
		fmt.Printf("   Config filename: %s\n", tmpFile)

		// 尝试手动加载来查看详细错误
		fmt.Println("\n🔍 Attempting manual load for debugging...")
		if loadErr := config.Load(context.Background()); loadErr != nil {
			fmt.Printf("   Load error: %v\n", loadErr)
		} else {
			fmt.Printf("   Load succeeded, FullFingers count: %d\n", config.FullFingers.Len())
		}
		os.Exit(1)
	}
	fmt.Printf("✓ Engine initialized (%d fingerprints loaded)\n\n", engine.Count())

	targetURL := "https://nacos.lzfzkj.com"  // 基础 URL，路径由指纹的 send_data 提供
	timeout := 10 // 10秒超时

	fmt.Printf("🎯 Target: %s\n", targetURL)
	fmt.Printf("⏱️  Timeout: %d seconds\n\n", timeout)

	// 测试 Level 0 (被动模式)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📋 Test 1: Level 0 (Passive Mode)")
	fmt.Println("   - Only analyzes response content")
	fmt.Println("   - No active probing requests")
	fmt.Println("═══════════════════════════════════════════════════════════")
	ctx0 := fingers.NewContext().WithTimeout(timeout).WithLevel(0)
	results0, err := engine.HTTPMatch(ctx0, []string{targetURL})
	if err != nil {
		fmt.Printf("❌ Level 0 error: %v\n\n", err)
	} else {
		printResults("Level 0", results0)
	}

	// 测试 Level 1 (基础主动探测)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📋 Test 2: Level 1 (Basic Active Probing)")
	fmt.Println("   - Sends finger-level send_data probes")
	fmt.Println("   - Balanced speed and accuracy")
	fmt.Println("═══════════════════════════════════════════════════════════")
	ctx1 := fingers.NewContext().WithTimeout(timeout).WithLevel(1)
	results1, err := engine.HTTPMatch(ctx1, []string{targetURL})
	if err != nil {
		fmt.Printf("❌ Level 1 error: %v\n\n", err)
	} else {
		printResults("Level 1", results1)
	}

	// 测试 Level 2 (深度主动探测)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("📋 Test 3: Level 2 (Deep Active Probing)")
	fmt.Println("   - Sends finger-level + rule-level send_data probes")
	fmt.Println("   - Most accurate, most traffic")
	fmt.Println("═══════════════════════════════════════════════════════════")
	ctx2 := fingers.NewContext().WithTimeout(timeout).WithLevel(2)
	results2, err := engine.HTTPMatch(ctx2, []string{targetURL})
	if err != nil {
		fmt.Printf("❌ Level 2 error: %v\n\n", err)
	} else {
		printResults("Level 2", results2)
	}

	fmt.Println("\n✓ All tests completed")
}

func printResults(level string, targetResults []*fingers.TargetResult) {
	if len(targetResults) == 0 {
		fmt.Printf("⚠️  %s: No results found\n", level)
		return
	}

	// 由于我们只测试单个目标，取第一个 TargetResult
	targetResult := targetResults[0]

	// 检查是否有错误
	if targetResult.Error != nil {
		fmt.Printf("❌ %s: Error - %v\n", level, targetResult.Error)
		return
	}

	// 检查是否有匹配结果
	if len(targetResult.Results) == 0 {
		fmt.Printf("⚠️  %s: No results found\n", level)
		return
	}

	fmt.Printf("✓ %s: Found %d result(s)\n\n", level, len(targetResult.Results))

	for i, result := range targetResult.Results {
		fmt.Printf("  ┌─ Result #%d ─────────────────────────────────────────\n", i+1)

		if result.Framework != nil {
			fmt.Printf("  │ Framework: %s\n", result.Framework.Name)
			fmt.Printf("  │ From: %s\n", result.Framework.From)

			if result.Framework.Attributes != nil && result.Framework.Version != "" {
				fmt.Printf("  │ Version: %s\n", result.Framework.Version)
			}

			if len(result.Framework.Tags) > 0 {
				fmt.Printf("  │ Tags: %v\n", result.Framework.Tags)
			}

			if result.Framework.IsFocus {
				fmt.Printf("  │ Focus: ✓ (Important fingerprint)\n")
			}
		} else {
			fmt.Printf("  │ Framework: None detected\n")
		}

		if result.Vuln != nil {
			fmt.Printf("  │ Vulnerability: %s\n", result.Vuln.Name)
		}

		fmt.Printf("  └────────────────────────────────────────────────────\n")
	}
	fmt.Println()
}
