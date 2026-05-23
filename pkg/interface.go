// Package sdk 提供 chainreactors 统一的 SDK 接口定义
package sdk

import "github.com/chainreactors/sdk/pkg/types"

// ========================================
// 引擎工厂类型
// ========================================

// EngineFactory 引擎工厂函数类型
type EngineFactory func(config interface{}) (Engine, error)

// RegisterFunc 注册函数类型（用于引擎）
type RegisterFunc func(name string, factory EngineFactory)

// globalRegister 全局引擎注册函数（由外层 SDK 包设置）
var globalRegister RegisterFunc

// pendingRegistrations 缓存早期的注册请求（在 SetRegisterFunc 之前）
var pendingRegistrations []struct {
	name    string
	factory EngineFactory
}

// SetRegisterFunc 设置全局引擎注册函数（仅由 SDK 根包调用）
func SetRegisterFunc(fn RegisterFunc) {
	globalRegister = fn

	// 执行所有缓存的注册
	for _, reg := range pendingRegistrations {
		globalRegister(reg.name, reg.factory)
	}
	pendingRegistrations = nil
}

// Register 注册引擎（供各引擎包的 init() 调用）
func Register(name string, factory EngineFactory) {
	if globalRegister != nil {
		globalRegister(name, factory)
	} else {
		// 缓存注册请求，等待 SetRegisterFunc 调用
		pendingRegistrations = append(pendingRegistrations, struct {
			name    string
			factory EngineFactory
		}{name, factory})
	}
}

// ========================================
// 核心概念 1: Engine - 引擎
// ========================================

type Engine = types.Engine

// ========================================
// 核心概念 2: Context - 上下文
// ========================================

type Context = types.Context

type Config = types.Config

// ========================================
// 核心概念 3: Task - 任务
// ========================================

type Task = types.Task

// ========================================
// 核心概念 4: Result - 结果
// ========================================

type Result = types.Result
