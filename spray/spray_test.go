package spray

import (
	"context"
	"testing"
	"time"
)

// TestNewSprayEngine 测试引擎创建
func TestNewSprayEngine(t *testing.T) {
	// 测试使用默认配置
	engine1 := NewSprayEngine(nil)
	if engine1 == nil {
		t.Fatal("NewSprayEngine(nil) should not return nil")
	}
	if engine1.inited {
		t.Error("Engine should not be initialized by default")
	}
	if engine1.opt == nil {
		t.Error("Engine should have default config")
	}

	// 测试使用自定义配置
	opt := DefaultConfig()
	opt.Threads = 200
	engine2 := NewSprayEngine(opt)
	if engine2.opt.Threads != 200 {
		t.Errorf("Expected Threads 200, got %d", engine2.opt.Threads)
	}

	// 测试兼容性 API
	engine3 := NewEngine(nil)
	if engine3 == nil {
		t.Fatal("NewEngine(nil) should not return nil")
	}
}

// TestSprayEngineName 测试引擎名称
func TestSprayEngineName(t *testing.T) {
	engine := NewSprayEngine(nil)
	if engine.Name() != "spray" {
		t.Errorf("Expected engine name 'spray', got '%s'", engine.Name())
	}
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Method != "GET" {
		t.Errorf("Expected default Method 'GET', got '%s'", config.Method)
	}
	if config.MaxBodyLength != 100 {
		t.Errorf("Expected default MaxBodyLength 100, got %d", config.MaxBodyLength)
	}
	if config.RandomUserAgent != false {
		t.Error("Expected default RandomUserAgent false")
	}
	if config.BlackStatus != "400,410" {
		t.Errorf("Expected default BlackStatus '400,410', got '%s'", config.BlackStatus)
	}
}

// TestContext 测试 Context 实现
func TestContext(t *testing.T) {
	ctx := NewContext()

	// 测试 Context()
	if ctx.Context() == nil {
		t.Error("Context() should not return nil")
	}

	// 测试 Config()
	config := ctx.Config()
	if config == nil {
		t.Error("Config() should not return nil")
	}

	// 测试 WithTimeout
	ctx2 := ctx.WithTimeout(5 * time.Second)
	if ctx2 == nil {
		t.Error("WithTimeout should not return nil")
	}

	// 测试 WithCancel
	ctx3, cancel := ctx.WithCancel()
	if ctx3 == nil {
		t.Error("WithCancel should not return nil context")
	}
	if cancel == nil {
		t.Error("WithCancel should not return nil cancel func")
	}
	cancel() // 清理

	// 测试链式调用
	config2 := NewConfig().SetThreads(300).SetTimeout(20)
	ctx4 := NewContext().WithConfig(config2).WithTimeout(10 * time.Second)
	if ctx4.Config().(*Config).Opt.Threads != 300 {
		t.Errorf("Expected threads 300 after chain call, got %d", ctx4.Config().(*Config).Opt.Threads)
	}
}

// TestConfig 测试 Config 实现
func TestConfig(t *testing.T) {
	config := NewConfig()

	// 测试默认值
	if config.Opt == nil {
		t.Fatal("Config should have default Option")
	}

	// 测试 Validate
	if err := config.Validate(); err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}

	// 测试链式调用
	config2 := NewConfig().
		SetThreads(250).
		SetTimeout(15).
		SetMethod("POST")

	if config2.Opt.Threads != 250 {
		t.Errorf("Expected threads 250, got %d", config2.Opt.Threads)
	}
	if config2.Opt.Timeout != 15 {
		t.Errorf("Expected Timeout 15, got %d", config2.Opt.Timeout)
	}
	if config2.Opt.Method != "POST" {
		t.Errorf("Expected Method 'POST', got '%s'", config2.Opt.Method)
	}
}

// TestCheckTask 测试 CheckTask
func TestCheckTask(t *testing.T) {
	// 测试有效任务
	urls := []string{"http://example.com", "https://google.com"}
	task1 := NewCheckTask(urls)
	if task1.Type() != "check" {
		t.Errorf("Expected task type 'check', got '%s'", task1.Type())
	}
	if err := task1.Validate(); err != nil {
		t.Errorf("Valid task should pass validation: %v", err)
	}

	// 测试无效任务 - 空 URL 列表
	task2 := NewCheckTask([]string{})
	if err := task2.Validate(); err == nil {
		t.Error("Task with empty URLs should fail validation")
	}

	// 测试无效任务 - nil URL 列表
	task3 := NewCheckTask(nil)
	if err := task3.Validate(); err == nil {
		t.Error("Task with nil URLs should fail validation")
	}
}

// TestBruteTask 测试 BruteTask
func TestBruteTask(t *testing.T) {
	// 测试有效任务
	wordlist := []string{"admin", "api", "test"}
	task1 := NewBruteTask("http://example.com", wordlist)
	if task1.Type() != "brute" {
		t.Errorf("Expected task type 'brute', got '%s'", task1.Type())
	}
	if err := task1.Validate(); err != nil {
		t.Errorf("Valid task should pass validation: %v", err)
	}

	// 测试无效任务 - 空 BaseURL
	task2 := NewBruteTask("", wordlist)
	if err := task2.Validate(); err == nil {
		t.Error("Task with empty BaseURL should fail validation")
	}

	// 测试无效任务 - 空字典
	task3 := NewBruteTask("http://example.com", []string{})
	if err := task3.Validate(); err == nil {
		t.Error("Task with empty wordlist should fail validation")
	}

	// 测试无效任务 - nil字典
	task4 := NewBruteTask("http://example.com", nil)
	if err := task4.Validate(); err == nil {
		t.Error("Task with nil wordlist should fail validation")
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

// TestSetThreads 测试设置线程数
func TestSetThreads(t *testing.T) {
	engine := NewSprayEngine(nil)

	engine.SetThreads(300)
	if engine.opt.Threads != 300 {
		t.Errorf("Expected threads 300, got %d", engine.opt.Threads)
	}

	engine.SetThreads(500)
	if engine.opt.Threads != 500 {
		t.Errorf("Expected threads 500, got %d", engine.opt.Threads)
	}
}

// TestSetTimeout 测试设置超时
func TestSetTimeout(t *testing.T) {
	engine := NewSprayEngine(nil)

	engine.SetTimeout(20)
	if engine.opt.Timeout != 20 {
		t.Errorf("Expected timeout 20, got %d", engine.opt.Timeout)
	}

	engine.SetTimeout(30)
	if engine.opt.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %d", engine.opt.Timeout)
	}
}
