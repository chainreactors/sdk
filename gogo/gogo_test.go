package gogo

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/gogo/v2/pkg"
)

// TestNewGogoEngine 测试引擎创建
func TestNewEngine(t *testing.T) {
	// 测试使用默认配置
	engine1 := NewEngine(nil)
	if engine1 == nil {
		t.Fatal("NewEngine(nil) should not return nil")
	}
	if engine1.inited {
		t.Error("Engine should not be initialized by default")
	}

	// 测试使用自定义配置
	opt := &pkg.RunnerOption{
		VersionLevel: 2,
		Exploit:      "auto",
	}
	engine2 := NewEngine(NewConfig())
	_ = opt
	if engine2 == nil {
		t.Fatal("NewEngine(NewConfig()) should not return nil")
	}

	// 测试空配置
	engine3 := NewEngine(nil)
	if engine3 == nil {
		t.Fatal("NewEngine(nil) should not return nil")
	}
}

// TestGogoEngineName 测试引擎名称
func TestGogoEngineName(t *testing.T) {
	engine := NewEngine(nil)
	if engine.Name() != "gogo" {
		t.Errorf("Expected engine name 'gogo', got '%s'", engine.Name())
	}
}

// TestGogoEngineSetThreads 测试设置线程数
func TestContextSetThreads(t *testing.T) {
	ctx := NewContext()

	ctx.SetThreads(500)
	if ctx.threads != 500 {
		t.Errorf("Expected threads 500, got %d", ctx.threads)
	}

	ctx.SetThreads(2000)
	if ctx.threads != 2000 {
		t.Errorf("Expected threads 2000, got %d", ctx.threads)
	}
}

// TestContext 测试 Context 实现
func TestContext(t *testing.T) {
	ctx := NewContext()

	// 测试 Context()
	if ctx.Context() == nil {
		t.Error("Context() should not return nil")
	}

	// 测试 WithTimeout
	timeoutCtx, _ := context.WithTimeout(ctx.Context(), 5*time.Second)
	ctx2 := ctx.WithContext(timeoutCtx)
	if ctx2 == nil {
		t.Error("WithTimeout should not return nil")
	}

	// 测试 WithCancel
	cancelCtx, cancel := context.WithCancel(ctx.Context())
	ctx3 := ctx.WithContext(cancelCtx)
	if ctx3 == nil {
		t.Error("WithCancel should not return nil context")
	}
	if cancel == nil {
		t.Error("WithCancel should not return nil cancel func")
	}
	cancel() // 清理

	// 测试链式调用
	baseCtx := NewContext().SetThreads(500).SetVersionLevel(3)
	chainCtx, _ := context.WithTimeout(baseCtx.Context(), 10*time.Second)
	ctx4 := baseCtx.WithContext(chainCtx)
	runCtx := ctx4
	if runCtx.threads != 500 {
		t.Errorf("Expected threads 500 after chain call, got %d", runCtx.threads)
	}
	if runCtx.opt.VersionLevel != 3 {
		t.Errorf("Expected VersionLevel 3 after chain call, got %d", runCtx.opt.VersionLevel)
	}
}

// TestConfig 测试 Config 实现
func TestConfig(t *testing.T) {
	config := NewConfig()

	// 测试 Validate
	if err := config.Validate(); err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}
}

// TestScanTask 测试 ScanTask
func TestScanTask(t *testing.T) {
	// 测试有效任务
	task1 := NewScanTask("127.0.0.1", "80,443")
	if task1.Type() != "scan" {
		t.Errorf("Expected task type 'scan', got '%s'", task1.Type())
	}
	if err := task1.Validate(); err != nil {
		t.Errorf("Valid task should pass validation: %v", err)
	}

	// 测试无效任务 - 空 IP
	task2 := NewScanTask("", "80")
	if err := task2.Validate(); err == nil {
		t.Error("Task with empty IP should fail validation")
	}

	// 测试无效任务 - 空端口
	task3 := NewScanTask("127.0.0.1", "")
	if err := task3.Validate(); err == nil {
		t.Error("Task with empty ports should fail validation")
	}
}

// TestWorkflowTask 测试 WorkflowTask
func TestWorkflowTask(t *testing.T) {
	// 测试有效任务
	workflow := &pkg.Workflow{
		Name:  "test-workflow",
		IP:    "127.0.0.1",
		Ports: "top100",
	}
	task1 := NewWorkflowTask(workflow)
	if task1.Type() != "workflow" {
		t.Errorf("Expected task type 'workflow', got '%s'", task1.Type())
	}
	if err := task1.Validate(); err != nil {
		t.Errorf("Valid task should pass validation: %v", err)
	}

	// 测试无效任务 - nil workflow
	task2 := NewWorkflowTask(nil)
	if err := task2.Validate(); err == nil {
		t.Error("Task with nil workflow should fail validation")
	}
}

// TestResult 测试 Result
func TestResult(t *testing.T) {
	// 测试成功结果
	result1 := &Result{
		success: true,
		err:     nil,
		data:    nil,
	}
	if !result1.Success() {
		t.Error("Result with success=true should return true from Success()")
	}
	if result1.Error() != nil {
		t.Error("Result with err=nil should return nil from Error()")
	}

	// 测试失败结果
	result2 := &Result{
		success: false,
		err:     context.DeadlineExceeded,
		data:    nil,
	}
	if result2.Success() {
		t.Error("Result with success=false should return false from Success()")
	}
	if result2.Error() == nil {
		t.Error("Result with error should return non-nil from Error()")
	}
}

// TestScanOne 测试单目标扫描
func TestScanOne(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := NewEngine(nil)
	if err := engine.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ctx := NewContext()

	// 扫描本地端口（应该很快）
	result := engine.ScanOne(ctx, "127.0.0.1", "65535")
	if result == nil {
		t.Fatal("ScanOne should not return nil")
	}

	// 注意：这里不检查端口是否开放，因为取决于实际环境
	t.Logf("ScanOne result: %s:%s - Status: %s", result.Ip, result.Port, result.Status)
}

// TestScanIntegration 测试实际扫描功能
func TestScanIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine := NewEngine(nil)
	if err := engine.Init(); err != nil {
		t.Skipf("Init failed (may need finger database): %v", err)
	}

	ctx := NewContext().SetThreads(50)
	timeoutCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
	defer cancel()
	ctx = ctx.WithContext(timeoutCtx)

	// 使用用户提供的IP段和top2端口
	results, err := engine.Scan(ctx, "81.68.175.32/28", "top2")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	t.Logf("Scan completed, found %d open ports", len(results))
	for _, result := range results {
		t.Logf("  %s:%s - %s (Title: %s)", result.Ip, result.Port, result.Status, result.Title)
	}

	if len(results) == 0 {
		t.Log("No open ports found, but this may be expected")
	}
}
