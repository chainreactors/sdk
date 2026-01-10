// Package sdk 提供 chainreactors 统一的 SDK 接口定义
package sdk

import (
	"context"
	"io"
	"time"
)

// ========================================
// 引擎工厂类型
// ========================================

// EngineFactory 引擎工厂函数类型
type EngineFactory func(config interface{}) (Engine, error)

// RegisterFunc 注册函数类型
type RegisterFunc func(name string, factory EngineFactory)

// globalRegister 全局注册函数（由外层 SDK 包设置）
var globalRegister RegisterFunc

// SetRegisterFunc 设置全局注册函数（仅由 SDK 根包调用）
func SetRegisterFunc(fn RegisterFunc) {
	globalRegister = fn
}

// Register 注册引擎（供各引擎包的 init() 调用）
func Register(name string, factory EngineFactory) {
	if globalRegister != nil {
		globalRegister(name, factory)
	}
}

// ========================================
// 核心概念 1: Engine - 引擎
// ========================================

// Engine 是所有 SDK 引擎的核心接口
// 职责：初始化、执行任务、返回结果
type Engine interface {
	// Name 返回引擎名称
	Name() string

	// Execute 执行任务，返回结果 channel
	// 这是唯一的执行方法，所有功能通过不同的 Task 类型实现
	Execute(ctx Context, task Task) (<-chan Result, error)

	// Close 关闭引擎，清理资源
	io.Closer
}

// ========================================
// 核心概念 2: Context - 上下文
// ========================================

// Context 执行上下文，包含运行时控制信息
// 职责：超时控制、取消信号
type Context interface {
	// Context 返回标准 context.Context（用于超时和取消）
	Context() context.Context

	// WithTimeout 返回新的 Context（包含超时）
	WithTimeout(timeout time.Duration) Context

	// WithCancel 返回新的 Context（可取消）
	WithCancel() (Context, context.CancelFunc)
}

// Config 配置接口（最小化）
type Config interface {
	// Validate 验证配置
	Validate() error
}

// ========================================
// 核心概念 3: Task - 任务
// ========================================

// Task 任务定义
// 职责：定义要执行的任务
type Task interface {
	// Type 返回任务类型
	Type() string

	// Validate 验证任务参数
	Validate() error
}

// ========================================
// 核心概念 4: Result - 结果
// ========================================

// Result 执行结果
// 职责：返回执行结果和状态
type Result interface {
	// Success 是否成功
	Success() bool

	// Error 返回错误（如果有）
	Error() error

	// Data 返回结果数据（具体类型由各引擎定义）
	Data() interface{}
}
