package main

import (
	"fmt"
	"log"

	"github.com/chainreactors/sdk/client"
	"github.com/chainreactors/sdk/gogo"
	"github.com/chainreactors/sdk/spray"
)

func main() {
	fmt.Println("=== SDK 新架构使用示例 ===\n")

	// 创建 SDK 客户端
	c := client.New()
	defer c.Close()

	// 示例1: Fingers - 指纹识别
	fmt.Println("1. Fingers 指纹识别")
	fmt.Println("----------------------------------------")
	testFingers(c)
	fmt.Println()

	// 示例2: Gogo - 端口扫描
	fmt.Println("2. Gogo 端口扫描")
	fmt.Println("----------------------------------------")
	testGogo(c)
	fmt.Println()

	// 示例3: Spray - URL 检测
	fmt.Println("3. Spray URL 检测")
	fmt.Println("----------------------------------------")
	testSpray(c)
	fmt.Println()
}

func testFingers(c *client.Client) {
	// 获取 Fingers 引擎
	fingersEngine, err := c.Fingers()
	if err != nil {
		log.Printf("❌ 获取 Fingers 引擎失败: %v\n", err)
		return
	}

	// 测试数据
	testData := []byte("HTTP/1.1 200 OK\r\nServer: nginx/1.18.0\r\n\r\n")

	// 调用 Match 方法
	frameworks, err := fingersEngine.Match(testData)
	if err != nil {
		log.Printf("❌ 匹配失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 识别到 %d 个指纹\n", len(frameworks))
	count := 0
	for _, fw := range frameworks {
		if count < 3 {
			fmt.Printf("   - %s\n", fw.Name)
			count++
		}
	}
}

func testGogo(c *client.Client) {
	// 获取 Gogo 引擎
	gogoEngine, err := c.Gogo()
	if err != nil {
		log.Printf("❌ 获取 Gogo 引擎失败: %v\n", err)
		return
	}

	// 创建上下文
	ctx := gogo.NewContext()

	// 调用 Scan 方法
	results, err := gogoEngine.Scan(ctx, "127.0.0.1", "80,443")
	if err != nil {
		log.Printf("❌ 扫描失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 扫描完成，发现 %d 个开放端口\n", len(results))
}

func testSpray(c *client.Client) {
	// 获取 Spray 引擎
	sprayEngine, err := c.Spray()
	if err != nil {
		log.Printf("❌ 获取 Spray 引擎失败: %v\n", err)
		return
	}

	// 创建上下文
	ctx := spray.NewContext()

	// 测试 URL
	urls := []string{
		"http://httpbin.org",
		"http://example.com",
	}

	// 调用 Check 方法
	results, err := sprayEngine.Check(ctx, urls)
	if err != nil {
		log.Printf("❌ 检测失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 检测完成，收到 %d 个响应\n", len(results))
}
