package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/gogo"
	"github.com/chainreactors/sdk/neutron"
)

var (
	// Cyberhub 配置
	cyberhubURL = flag.String("url", "", "Cyberhub URL (e.g., http://127.0.0.1:8080)")
	apiKey      = flag.String("key", "", "Cyberhub API Key")
	source      = flag.String("source", "", "Filter by source (optional)")
	loadFingers = flag.Bool("fingers", true, "Load fingers from Cyberhub")
	loadNeutron = flag.Bool("neutron", false, "Load neutron POCs from Cyberhub")

	// 扫描配置
	target       = flag.String("target", "", "Target IP or CIDR (required)")
	ports        = flag.String("ports", "80,443,8080,8443", "Ports to scan (comma separated)")
	threads      = flag.Int("threads", 1000, "Number of threads")
	versionLevel = flag.Int("version", 0, "Version detection level (0-3)")
	exploit      = flag.String("exploit", "none", "Exploit mode (none/all/known)")
	timeout      = flag.Int("timeout", 5, "Request timeout in seconds")

	// 输出选项
	jsonOut = flag.Bool("json", false, "Output as JSON")
	verbose = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	// 验证参数
	if *target == "" {
		fmt.Println("Usage: gogo [-url <cyberhub_url> -key <api_key>] -target <ip/cidr> -ports <ports>")
		fmt.Println("\nBasic scan:")
		fmt.Println("  gogo -target 127.0.0.1 -ports 80,443")
		fmt.Println("\nWith Cyberhub fingerprints:")
		fmt.Println("  gogo -url http://127.0.0.1:8080 -key your_key -target 127.0.0.1")
		fmt.Println("\nWith Cyberhub fingerprints and POCs:")
		fmt.Println("  gogo -url ... -key ... -fingers -neutron -target 127.0.0.1")
		fmt.Println("\nFilter by source:")
		fmt.Println("  gogo -url ... -key ... -source github -target 127.0.0.1")
		fmt.Println("\nAdvanced:")
		fmt.Println("  gogo -target 192.168.1.0/24 -ports 80,443,8080 -threads 2000 -version 2")
		fmt.Println("\nNote: GoGo CLI requires Cyberhub (-url and -key) to load fingerprints and POCs.")
		fmt.Println("      For standalone usage, please use the gogo command-line tool directly.")
		os.Exit(1)
	}

	// GoGo CLI 需要 Cyberhub 配置
	if *cyberhubURL == "" || *apiKey == "" {
		fmt.Println("Error: GoGo CLI requires Cyberhub configuration")
		fmt.Println("Usage: gogo -url <cyberhub_url> -key <api_key> -target <ip/cidr>")
		fmt.Println("\nExample:")
		fmt.Println("  gogo -url http://127.0.0.1:8080 -key your_key -target 127.0.0.1 -ports 80,443")
		os.Exit(1)
	}

	ctx := context.Background()

	// 1. 加载 Fingers 和 Neutron (可选)
	var fEngine *fingers.Engine
	var neutronEng *neutron.Engine
	templatesCount := 0

	// 加载 Fingers
	if *loadFingers {
		if !*jsonOut {
			fmt.Printf("Loading fingerprints from Cyberhub (%s)...\n", *cyberhubURL)
		}

		fingersConfig := fingers.NewConfig()
		fingersConfig.WithCyberhub(*cyberhubURL, *apiKey)
		if err := fingersConfig.Load(ctx); err != nil {
			fmt.Printf("Error loading fingers config: %v\n", err)
			os.Exit(1)
		}

		if *source != "" {
			fingersConfig.SetSources(*source)
		}

		fingersEng, err := fingers.NewEngine(fingersConfig)
		if err != nil {
			fmt.Printf("Error creating fingers engine: %v\n", err)
			os.Exit(1)
		}

		_, err = fingersEng.Load(ctx)
		if err != nil {
			fmt.Printf("Error loading fingerprints: %v\n", err)
			os.Exit(1)
		}

		impl, err := fingersEng.GetFingersEngine()
		if err != nil {
			fmt.Printf("Error getting fingers engine: %v\n", err)
			os.Exit(1)
		}
		if impl != nil {
			fEngine = fingersEng
			if !*jsonOut {
				fmt.Printf("? Loaded %d HTTP fingerprints, %d Socket fingerprints\n",
					len(impl.HTTPFingers), len(impl.SocketFingers))
			}
		}
	}

	// 加载 Neutron POCs
	if *loadNeutron {
		if !*jsonOut {
			fmt.Printf("Loading POCs from Cyberhub (%s)...\n", *cyberhubURL)
		}

		neutronConfig := neutron.NewConfig()
		neutronConfig.WithCyberhub(*cyberhubURL, *apiKey)
		if err := neutronConfig.Load(ctx); err != nil {
			fmt.Printf("Error loading neutron config: %v\n", err)
			os.Exit(1)
		}

		if *source != "" {
			neutronConfig.SetSources(*source)
		}

		neutronEng, err = neutron.NewEngine(neutronConfig)
		if err != nil {
			fmt.Printf("Error creating neutron engine: %v\n", err)
			os.Exit(1)
		}

		templates, err := neutronEng.Load(ctx)
		if err != nil {
			fmt.Printf("Error loading POCs: %v\n", err)
			os.Exit(1)
		}

		templatesCount = len(templates)
		if !*jsonOut {
			fmt.Printf("? Loaded and compiled %d POCs\n", templatesCount)
		}
	}

	// 2. 创建 GoGo 引擎
	if !*jsonOut {
		fmt.Println("\nInitializing GoGo engine...")
	}

	gogoConfig := gogo.NewConfig()
	if fEngine != nil {
		gogoConfig.WithFingersEngine(fEngine)
	}
	if neutronEng != nil {
		gogoConfig.WithNeutronEngine(neutronEng)
	}
	gogoEngine := gogo.NewEngine(gogoConfig)

	if err := gogoEngine.Init(); err != nil {
		fmt.Printf("Error initializing gogo: %v\n", err)
		os.Exit(1)
	}

	if !*jsonOut {
		fmt.Println("✅ GoGo engine initialized\n")
	}

	// 3. 配置扫描参数
	gogoCtx := gogo.NewContext().
		SetThreads(*threads).
		SetVersionLevel(*versionLevel).
		SetExploit(*exploit).
		SetDelay(*timeout)

	// 4. 执行扫描
	if !*jsonOut {
		fmt.Printf("🔍 Scanning %s (ports: %s)\n", *target, *ports)
		fmt.Printf("   Threads: %d | Version Level: %d | Timeout: %ds\n\n", *threads, *versionLevel, *timeout)
	}

	scanTask := gogo.NewScanTask(*target, *ports)
	resultCh, err := gogoEngine.Execute(gogoCtx, scanTask)
	if err != nil {
		fmt.Printf("Error executing scan: %v\n", err)
		os.Exit(1)
	}

	// 5. 处理结果
	results := []map[string]interface{}{}
	aliveCount := 0

	for result := range resultCh {
		if !result.Success() {
			continue
		}

		gogoResult := result.(*gogo.Result).GOGOResult()
		if gogoResult == nil {
			continue
		}

		aliveCount++

		resultMap := map[string]interface{}{
			"ip":     gogoResult.Ip,
			"port":   gogoResult.Port,
			"status": gogoResult.Status,
		}

		if len(gogoResult.Frameworks) > 0 {
			frameworks := []string{}
			for _, fw := range gogoResult.Frameworks {
				frameworks = append(frameworks, fw.Name)
			}
			resultMap["frameworks"] = frameworks
		}

		if gogoResult.Title != "" {
			resultMap["title"] = gogoResult.Title
		}

		results = append(results, resultMap)

		// 实时输出
		if !*jsonOut {
			output := fmt.Sprintf("✓ %s:%s - %s", gogoResult.Ip, gogoResult.Port, gogoResult.Status)
			if len(gogoResult.Frameworks) > 0 {
				fwNames := []string{}
				for _, fw := range gogoResult.Frameworks {
					fwNames = append(fwNames, fw.Name)
				}
				output += fmt.Sprintf(" [%s]", strings.Join(fwNames, ", "))
			}
			if gogoResult.Title != "" {
				output += fmt.Sprintf(" (%s)", gogoResult.Title)
			}
			fmt.Println(output)
		}
	}

	// 6. 输出汇总
	if *jsonOut {
		output := map[string]interface{}{
			"target":      *target,
			"ports":       *ports,
			"alive_count": aliveCount,
			"results":     results,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("\n========================================")
		fmt.Printf("📊 Scan completed\n")
		fmt.Printf("   Alive hosts: %d\n", aliveCount)
		fmt.Println("========================================")
	}
}
