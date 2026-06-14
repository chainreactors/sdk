package proton

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chainreactors/neutron/protocols"
	"github.com/chainreactors/proton/proton/file"
	"github.com/chainreactors/sdk/pkg/types"
)

type Engine struct {
	scanner  *file.Scanner
	config   *Config
	capacity *types.Capacity
	mu       sync.Mutex
	inited   bool
}

func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = NewConfig()
	}
	e := &Engine{config: config}
	if config.Capacity > 0 {
		e.capacity = types.NewCapacity(config.Capacity)
	}
	if err := e.init(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) init() error {
	if e.inited {
		return nil
	}

	rules, err := e.config.Load()
	if err != nil {
		return fmt.Errorf("load proton templates: %w", err)
	}

	execOpts := &protocols.ExecuterOptions{
		Options: &protocols.Options{TextOnly: e.config.TextOnly},
	}
	e.scanner = file.NewScanner(rules, execOpts)
	e.inited = true
	return nil
}

func (e *Engine) Name() string {
	return "proton"
}

func (e *Engine) SetCapacity(total int) {
	if total > 0 {
		e.capacity = types.NewCapacity(total)
	}
}

func (e *Engine) Capacity() *types.Capacity {
	return e.capacity
}

func (e *Engine) Scanner() *file.Scanner {
	return e.scanner
}

func (e *Engine) Execute(ctx types.Context, task types.Task) (<-chan types.Result, error) {
	if !e.inited {
		if err := e.init(); err != nil {
			return nil, err
		}
	}
	if err := task.Validate(); err != nil {
		return nil, err
	}

	var runCtx *Context
	if ctx == nil {
		runCtx = NewContext()
	} else {
		var ok bool
		runCtx, ok = ctx.(*Context)
		if !ok {
			return nil, fmt.Errorf("unsupported context type: %T", ctx)
		}
		if runCtx == nil {
			runCtx = NewContext()
		}
	}

	switch t := task.(type) {
	case *ScanDataTask:
		return e.executeScanData(runCtx, t)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}
}

func (e *Engine) Close() error {
	return nil
}

// ScanData 对内存数据执行敏感信息匹配，label 用于标记数据来源（如文件名）。
func (e *Engine) ScanData(data []byte, label string) []Finding {
	if !e.inited {
		if err := e.init(); err != nil {
			return nil
		}
	}
	if e.scanner == nil || len(e.scanner.Groups) == 0 {
		return nil
	}
	var findings []Finding
	for _, group := range e.scanner.Groups {
		findings = append(findings, e.scanner.ScanData(data, label, group)...)
	}
	return findings
}

// ScanBlock 对二进制数据块执行滑动窗口匹配（适用于进程内存、网络流等非文本数据）。
func (e *Engine) ScanBlock(data []byte, label string) []Finding {
	if !e.inited {
		if err := e.init(); err != nil {
			return nil
		}
	}
	if e.scanner == nil || len(e.scanner.Groups) == 0 {
		return nil
	}
	var findings []Finding
	for _, group := range e.scanner.Groups {
		findings = append(findings, e.scanner.ScanBlock(data, label, group)...)
	}
	return findings
}

// NewLineWriter 返回流式文本扫描器（io.WriteCloser）。
// 写入的数据按行缓冲，每行完成时自动匹配，Close 时 flush 剩余数据。
func (e *Engine) NewLineWriter(label string, callback func(Finding)) io.WriteCloser {
	return e.scanner.NewLineWriter(label, callback)
}

// NewBlockWriter 返回流式二进制扫描器（io.WriteCloser）。
// 写入的数据以滑动窗口方式匹配，适用于进程内存、网络流等非文本数据。
func (e *Engine) NewBlockWriter(label string, callback func(Finding)) io.WriteCloser {
	return e.scanner.NewBlockWriter(label, callback)
}

func (e *Engine) executeScanData(ctx *Context, task *ScanDataTask) (<-chan types.Result, error) {
	if e.capacity != nil {
		if err := e.capacity.Acquire(ctx.Context(), 1); err != nil {
			return nil, err
		}
	}

	started := time.Now()
	resultCh := make(chan types.Result, 100)

	go func() {
		defer close(resultCh)
		if e.capacity != nil {
			defer e.capacity.Release(1)
		}

		var findingCount int64
		for _, group := range e.scanner.Groups {
			findings := e.scanner.ScanData(task.Data, task.Label, group)
			for i := range findings {
				findingCount++
				select {
				case resultCh <- types.NewResult(true, nil, &findings[i]):
				case <-ctx.Context().Done():
					return
				}
			}
		}

		ctx.emitStats(types.Stats{
			Engine:   e.Name(),
			Task:     task.Type(),
			Targets:  1,
			Results:  findingCount,
			Duration: time.Since(started),
		})
	}()

	return resultCh, nil
}
