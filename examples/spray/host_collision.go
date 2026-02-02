package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/spray"
)

var (
	// 目标配置
	targetIP   = flag.String("target", "", "Target IP or URL (e.g., 192.168.1.1 or http://192.168.1.1)")
	hostFile   = flag.String("hosts", "", "File containing host names to test (one per line)")
	hostList   = flag.String("host-list", "", "Comma-separated list of hosts (e.g., example.com,test.com)")

	// 扫描配置
	threads    = flag.Int("threads", 20, "Number of concurrent threads")
	timeout    = flag.Int("timeout", 10, "Request timeout in seconds")
	method     = flag.String("method", "GET", "HTTP method")
	path       = flag.String("path", "/", "Request path")

	// 输出配置
	verbose    = flag.Bool("v", false, "Verbose output (show all responses)")
	showBody   = flag.Bool("body", false, "Show response body preview")
	outputFile = flag.String("o", "", "Output file for results")
)

func main() {
	flag.Parse()

	// 验证参数
	if *targetIP == "" {
		printUsage()
		os.Exit(1)
	}

	if *hostFile == "" && *hostList == "" {
		fmt.Println("Error: Must specify either -hosts (file) or -host-list (comma-separated)")
		printUsage()
		os.Exit(1)
	}

	// 1. 读取 Host 列表
	var hosts []string
	var err error

	if *hostFile != "" {
		hosts, err = readHostsFromFile(*hostFile)
		if err != nil {
			fmt.Printf("Error reading hosts file: %v\n", err)
			os.Exit(1)
		}
	} else if *hostList != "" {
		hosts = strings.Split(*hostList, ",")
		for i := range hosts {
			hosts[i] = strings.TrimSpace(hosts[i])
		}
	}

	if len(hosts) == 0 {
		fmt.Println("Error: No hosts to test")
		os.Exit(1)
	}

	fmt.Printf("🎯 Host Collision Detection\n")
	fmt.Printf("   Target: %s\n", *targetIP)
	fmt.Printf("   Hosts: %d\n", len(hosts))
	fmt.Printf("   Path: %s\n", *path)
	fmt.Printf("   Threads: %d | Timeout: %ds\n\n", *threads, *timeout)

	// 2. 准备目标 URL
	target := normalizeTarget(*targetIP, *path)

	// 3. 创建 Spray 引擎
	sprayEngine := spray.NewEngine(nil)
	if err := sprayEngine.Init(); err != nil {
		fmt.Printf("Error initializing spray: %v\n", err)
		os.Exit(1)
	}

	// 4. 准备输出文件
	var outputWriter *bufio.Writer
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		outputWriter = bufio.NewWriter(f)
		defer outputWriter.Flush()
	}

	// 5. 对每个 Host 进行测试
	results := make(map[string]*HostResult)

	for _, host := range hosts {
		result := testHost(sprayEngine, target, host, *method, *threads, *timeout)
		results[host] = result

		// 实时输出结果
		printHostResult(host, result, *verbose, *showBody)

		if outputWriter != nil {
			outputWriter.WriteString(formatHostResult(host, result) + "\n")
		}
	}

	// 6. 输出汇总和分析
	fmt.Println("\n========================================")
	fmt.Println("📊 Summary")
	fmt.Println("========================================")

	analyzeResults(results)

	if *outputFile != "" {
		fmt.Printf("\n✓ Results saved to: %s\n", *outputFile)
	}
}

type HostResult struct {
	Host       string
	Status     int
	Title      string
	Length     int
	Success    bool
	Error      error
}

func testHost(engine *spray.SprayEngine, target, host, method string, threads, timeout int) *HostResult {
	// 创建带有自定义 Host 头的上下文
	ctx := spray.NewContext().
		SetThreads(threads).
		SetTimeout(timeout).
		SetMethod(method).
		SetHost(host)

	// 执行检测
	task := spray.NewCheckTask([]string{target})
	resultCh, err := engine.Execute(ctx, task)

	if err != nil {
		return &HostResult{
			Host:    host,
			Success: false,
			Error:   err,
		}
	}

	// 获取结果
	for result := range resultCh {
		if !result.Success() {
			return &HostResult{
				Host:    host,
				Success: false,
				Error:   result.Error(),
			}
		}

		sprayResult := result.(*spray.Result).SprayResult()
		if sprayResult != nil {
			return &HostResult{
				Host:    host,
				Status:  sprayResult.Status,
				Title:   sprayResult.Title,
				Length:  sprayResult.BodyLength,
				Success: true,
			}
		}
	}

	return &HostResult{
		Host:    host,
		Success: false,
	}
}

func printHostResult(host string, result *HostResult, verbose, showBody bool) {
	if !result.Success {
		if verbose {
			fmt.Printf("✗ %-30s [ERROR] %v\n", host, result.Error)
		}
		return
	}

	statusIcon := getStatusIcon(result.Status)
	fmt.Printf("%s %-30s [%d] Length: %-6d", statusIcon, host, result.Status, result.Length)

	if result.Title != "" {
		fmt.Printf(" Title: %s", result.Title)
	}

	fmt.Println()
}

func formatHostResult(host string, result *HostResult) string {
	if !result.Success {
		return fmt.Sprintf("%s\t[ERROR]\t%v", host, result.Error)
	}

	return fmt.Sprintf("%s\t[%d]\t%d\t%s", host, result.Status, result.Length, result.Title)
}

func getStatusIcon(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "✓"
	case status >= 300 && status < 400:
		return "↪"
	case status >= 400 && status < 500:
		return "⚠"
	case status >= 500:
		return "✗"
	default:
		return "?"
	}
}

func analyzeResults(results map[string]*HostResult) {
	// 统计状态码分布
	statusCount := make(map[int]int)
	lengthGroups := make(map[int][]string)

	successCount := 0
	for host, result := range results {
		if result.Success {
			successCount++
			statusCount[result.Status]++
			lengthGroups[result.Length] = append(lengthGroups[result.Length], host)
		}
	}

	fmt.Printf("Total Hosts: %d | Successful: %d | Failed: %d\n\n",
		len(results), successCount, len(results)-successCount)

	// 显示状态码分布
	fmt.Println("Status Code Distribution:")
	for status, count := range statusCount {
		fmt.Printf("  [%d]: %d hosts\n", status, count)
	}

	// 查找可能的虚拟主机（响应长度不同）
	fmt.Println("\nPotential Virtual Hosts (by response length):")
	uniqueLengths := 0
	for length, hosts := range lengthGroups {
		if len(hosts) > 0 {
			uniqueLengths++
			fmt.Printf("  Length %d (%d hosts):\n", length, len(hosts))
			for _, host := range hosts {
				if result, ok := results[host]; ok {
					fmt.Printf("    - %s [%d] %s\n", host, result.Status, result.Title)
				}
			}
		}
	}

	if uniqueLengths > 1 {
		fmt.Println("\n💡 Found different response lengths - possible virtual hosts detected!")
	} else {
		fmt.Println("\n⚠ All responses have similar length - may be default responses")
	}
}

func normalizeTarget(target, path string) string {
	// 确保目标有协议
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}

	// 移除路径（如果有）
	if idx := strings.Index(target, "/"); idx > 8 {
		target = target[:idx]
	}

	// 添加路径
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return target + path
}

func readHostsFromFile(filename string) ([]string, error) {
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
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hosts, nil
}

func printUsage() {
	fmt.Println("Host Collision Detection Tool")
	fmt.Println("\nUsage:")
	fmt.Println("  host_collision -target <ip/url> -hosts <file> [options]")
	fmt.Println("  host_collision -target <ip/url> -host-list <hosts> [options]")
	fmt.Println("\nExamples:")
	fmt.Println("  # Test from file")
	fmt.Println("  host_collision -target 192.168.1.1 -hosts hosts.txt")
	fmt.Println("")
	fmt.Println("  # Test from list")
	fmt.Println("  host_collision -target http://192.168.1.1 -host-list example.com,test.com,admin.example.com")
	fmt.Println("")
	fmt.Println("  # With custom path and options")
	fmt.Println("  host_collision -target 192.168.1.1 -hosts hosts.txt -path /admin -threads 50 -v")
	fmt.Println("")
	fmt.Println("  # Save results to file")
	fmt.Println("  host_collision -target 192.168.1.1 -hosts hosts.txt -o results.txt")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}
