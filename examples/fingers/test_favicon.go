package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/chainreactors/sdk/fingers"
)

func main() {
	fmt.Println("🧪 Testing Favicon Matching Mechanism")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	// 创建临时的 nacos 指纹文件
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
	tmpFile := "nacos_finger_favicon.yaml"
	if err := os.WriteFile(tmpFile, []byte(nacosFingerprint), 0644); err != nil {
		fmt.Printf("❌ Failed to write fingerprint file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	fmt.Println("📝 Created temporary nacos fingerprint file")

	// 创建配置，使用本地文件
	config := fingers.NewConfig().WithLocalFile(tmpFile)
	fmt.Println("📋 Config created with local file")

	// 创建 engine
	fmt.Println("🔧 Initializing Fingers engine...")
	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Printf("❌ Failed to create engine: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Engine initialized (%d fingerprints loaded)\n\n", engine.Count())

	// 测试 favicon 匹配
	faviconURL := "https://nacos.lzfzkj.com/nacos/console-ui/public/img/nacos-logo.png"
	fmt.Printf("🎯 Fetching favicon from: %s\n", faviconURL)

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 获取 favicon
	resp, err := client.Get(faviconURL)
	if err != nil {
		fmt.Printf("❌ Failed to fetch favicon: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Favicon request failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// 读取 favicon 数据
	faviconData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read favicon data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Favicon fetched successfully (%d bytes)\n\n", len(faviconData))

	// 使用 MatchFavicon 进行匹配
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📋 Testing MatchFavicon")
	fmt.Println("═══════════════════════════════════════════════════════════")

	results, err := engine.MatchFavicon(faviconData)
	if err != nil {
		fmt.Printf("❌ MatchFavicon error: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("⚠️  No favicon matches found")
		fmt.Println("\n❌ Favicon matching mechanism may not be working correctly")
		os.Exit(1)
	}

	fmt.Printf("✓ Found %d favicon match(es)\n\n", len(results))

	count := 0
	for _, result := range results {
		count++
		fmt.Printf("  ┌─ Match #%d ─────────────────────────────────────────\n", count)
		fmt.Printf("  │ Framework: %s\n", result.Name)
		fmt.Printf("  │ From: %s\n", result.From)

		if len(result.Tags) > 0 {
			fmt.Printf("  │ Tags: %v\n", result.Tags)
		}

		if result.IsFocus {
			fmt.Printf("  │ Focus: ✓ (Important fingerprint)\n")
		}

		fmt.Printf("  └────────────────────────────────────────────────────\n")
	}

	fmt.Println("\n✓ Favicon matching mechanism is working correctly!")
}
