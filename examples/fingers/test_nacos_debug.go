package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/chainreactors/fingers/common"
	"github.com/chainreactors/sdk/fingers"
)

func main() {
	fmt.Println("🔍 Debugging Nacos Fingerprint Detection")
	fmt.Println("─────────────────────────────────────────────────────────────\n")

	targetURL := "http://nacos.lzfzkj.com"

	// 步骤1: 测试目标是否可访问
	fmt.Println("📡 Step 1: Testing target accessibility...")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(targetURL)
	if err != nil {
		fmt.Printf("❌ Failed to access target: %v\n", err)
		fmt.Println("   This might be why fingerprint detection is failing.")
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("✓ Target is accessible\n")
	fmt.Printf("  Status Code: %d\n", resp.StatusCode)
	fmt.Printf("  Content-Type: %s\n", resp.Header.Get("Content-Type"))

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response body: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Body Length: %d bytes\n", len(body))

	// 显示前500个字符
	preview := string(body)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Printf("  Body Preview:\n%s\n\n", preview)

	// 检查是否包含 nacos 关键字
	bodyStr := string(body)
	if contains := findKeywords(bodyStr, []string{"Nacos", "nacos", "<title>"}); len(contains) > 0 {
		fmt.Printf("✓ Found keywords in response: %v\n\n", contains)
	} else {
		fmt.Printf("⚠️  No nacos-related keywords found in response\n\n")
	}

	// 步骤2: 测试指纹匹配
	fmt.Println("📋 Step 2: Testing fingerprint matching...")

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

	tmpFile := "nacos_finger_debug.yaml"
	if err := os.WriteFile(tmpFile, []byte(nacosFingerprint), 0644); err != nil {
		fmt.Printf("❌ Failed to write fingerprint file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	config := fingers.NewConfig().WithLocalFile(tmpFile)
	engine, err := fingers.NewEngine(config)
	if err != nil {
		fmt.Printf("❌ Failed to create engine: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Engine initialized (%d fingerprints loaded)\n\n", engine.Count())

	// 测试 Level 1
	fmt.Println("📋 Step 3: Testing with Level 1...")
	ctx := fingers.NewContext().WithTimeout(10).WithLevel(1)
	results, err := engine.HTTPMatch(ctx, targetURL)
	if err != nil {
		fmt.Printf("❌ HTTPMatch error: %v\n", err)
	} else {
		fmt.Printf("✓ HTTPMatch completed: %d results\n", len(results))
		if len(results) > 0 {
			for i, result := range results {
				fmt.Printf("  Result %d:\n", i+1)
				if result.Framework != nil {
					fmt.Printf("    Framework: %s\n", result.Framework.Name)
					fmt.Printf("    From: %s\n", result.Framework.From)
				}
			}
		} else {
			fmt.Println("  No frameworks detected")
		}
	}

	fmt.Println("\n✓ Debug test completed")
}

func findKeywords(text string, keywords []string) []string {
	var found []string
	for _, keyword := range keywords {
		if len(text) > 0 && contains(text, keyword) {
			found = append(found, keyword)
		}
	}
	return found
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
