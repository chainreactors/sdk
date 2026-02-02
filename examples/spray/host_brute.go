package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/spray"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: host_brute <target_ip> <hosts_file> [limit]")
		fmt.Println("Example: host_brute 110.75.231.10 hosts.txt 10000")
		os.Exit(1)
	}

	targetIP := os.Args[1]
	hostsFile := os.Args[2]
	limit := 0

	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &limit)
	}

	// 读取 hosts 文件
	hosts, err := readHosts(hostsFile, limit)
	if err != nil {
		fmt.Printf("Error reading hosts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🎯 Host Collision Attack\n")
	fmt.Printf("   Target: %s\n", targetIP)
	fmt.Printf("   Hosts: %d\n", len(hosts))
	fmt.Println()

	// 创建 Spray 引擎
	engine := spray.NewEngine(nil)
	if err := engine.Init(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 配置 host 模式
	ctx := spray.NewContext().
		SetThreads(100).
		SetTimeout(10).
		SetMod("host")

	// 执行暴力破解
	task := spray.NewBruteTask("http://"+targetIP, hosts)
	resultCh, err := engine.Execute(ctx, task)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 处理结果
	count := 0
	for result := range resultCh {
		if result.Success() {
			sprayResult := result.(*spray.Result).SprayResult()
			if sprayResult != nil && sprayResult.Status != 0 {
				count++
				fmt.Printf("[%d] %s - Status: %d, Length: %d",
					count, sprayResult.UrlString, sprayResult.Status, sprayResult.BodyLength)
				if sprayResult.Title != "" {
					fmt.Printf(", Title: %s", sprayResult.Title)
				}
				fmt.Println()
			}
		}
	}

	fmt.Printf("\n✓ Found %d valid hosts\n", count)
}

func readHosts(filename string, limit int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			hosts = append(hosts, line)
			if limit > 0 && len(hosts) >= limit {
				break
			}
		}
	}

	return hosts, scanner.Err()
}
