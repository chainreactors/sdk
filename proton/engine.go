package proton

import (
	"fmt"
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

func NewEngine(config *Config) *Engine {
	if config == nil {
		config = NewConfig()
	}
	e := &Engine{config: config}
	if config.Capacity > 0 {
		e.capacity = types.NewCapacity(config.Capacity)
	}
	return e
}

func (e *Engine) Init() error {
	e.mu.Lock()
	defer e.mu.Unlock()

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
		if err := e.Init(); err != nil {
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
	}

	switch t := task.(type) {
	case *ScanTask:
		return e.executeScan(runCtx, t)
	case *ScanDataTask:
		return e.executeScanData(runCtx, t)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}
}

func (e *Engine) Close() error {
	return nil
}

// ========================================
// Convenience API
// ========================================

func (e *Engine) Scan(ctx *Context, target string) ([]*Finding, error) {
	return e.collect(e.ScanStream(ctx, target))
}

func (e *Engine) ScanStream(ctx *Context, target string) (<-chan *Finding, error) {
	task := NewScanTask(target)
	return e.typedStream(ctx, task)
}

func (e *Engine) ScanData(data []byte, filePath string) []Finding {
	if !e.inited {
		if err := e.Init(); err != nil {
			return nil
		}
	}
	if e.scanner == nil || len(e.scanner.Groups) == 0 {
		return nil
	}
	var findings []Finding
	for _, group := range e.scanner.Groups {
		findings = append(findings, e.scanner.ScanData(data, filePath, group)...)
	}
	return findings
}

// ========================================
// Internal execution
// ========================================

func (e *Engine) executeScan(ctx *Context, task *ScanTask) (<-chan types.Result, error) {
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
		e.scanner.Scan(task.Target, func(f Finding) {
			findingCount++
			select {
			case resultCh <- &ScanResult{success: true, data: &f}:
			case <-ctx.Context().Done():
				return
			}
		})

		ctx.emitStats(types.Stats{
			Engine:   e.Name(),
			Task:     task.Type(),
			Targets:  1,
			Tasks:    e.scanner.Stats.Files,
			Requests: e.scanner.Stats.Files,
			Results:  findingCount,
			Duration: time.Since(started),
		})
	}()

	return resultCh, nil
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
			findings := e.scanner.ScanData(task.Data, task.FilePath, group)
			for i := range findings {
				findingCount++
				select {
				case resultCh <- &ScanResult{success: true, data: &findings[i]}:
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

func (e *Engine) typedStream(ctx *Context, task types.Task) (<-chan *Finding, error) {
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	findingCh := make(chan *Finding, 100)
	go func() {
		defer close(findingCh)
		for result := range resultCh {
			if data, ok := types.ResultData[*Finding](result); result.Success() && ok && data != nil {
				findingCh <- data
			}
		}
	}()
	return findingCh, nil
}

func (e *Engine) collect(ch <-chan *Finding, err error) ([]*Finding, error) {
	if err != nil {
		return nil, err
	}
	var findings []*Finding
	for f := range ch {
		findings = append(findings, f)
	}
	return findings, nil
}
