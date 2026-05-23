package zombie

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/sdk/pkg/types"
	zombiecore "github.com/chainreactors/zombie/core"
	zombiepkg "github.com/chainreactors/zombie/pkg"
	"github.com/panjf2000/ants/v2"
)

type Engine struct {
	inited   bool
	config   *Config
	capacity *types.Capacity
	mu       sync.Mutex
}

func newResult(success bool, err error, data *types.ZombieResult) types.Result {
	return types.NewResult(success, err, data)
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
	if e.config != nil && e.config.ResourceProvider != nil {
		zombiepkg.SetResourceProvider(e.config.ResourceProvider)
	}
	if err := zombiepkg.Load(); err != nil {
		return err
	}
	e.inited = true
	return nil
}

func (e *Engine) Name() string {
	return "zombie"
}

// SetCapacity configures a capacity limit on an already-created engine.
func (e *Engine) SetCapacity(total int) {
	if total > 0 {
		e.capacity = types.NewCapacity(total)
	}
}

// Capacity returns the engine's capacity bucket, or nil if unconfigured.
func (e *Engine) Capacity() *types.Capacity {
	return e.capacity
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
	case *WeakpassTask:
		return e.executeWeakpass(runCtx, t)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type())
	}
}

func (e *Engine) Close() error {
	return nil
}

func (e *Engine) executeWeakpass(ctx *Context, task *WeakpassTask) (<-chan types.Result, error) {
	threads := ctx.threads
	if e.capacity != nil {
		if err := e.capacity.Acquire(ctx.Context(), threads); err != nil {
			return nil, err
		}
	}

	ztasks := expandTasks(ctx, task)
	started := time.Now()
	var requests int64
	var results int64
	var errors int64

	resultCh := make(chan types.Result, ctx.threads)
	pool, err := ants.NewPoolWithFunc(ctx.threads, func(i interface{}) {
		defer i.(*workItem).wg.Done()
		item := i.(*workItem)
		atomic.AddInt64(&requests, 1)
		result := runTask(item.ctx, item.task)
		if result == nil {
			return
		}
		if isZombieExecutionError(result.Err) {
			atomic.AddInt64(&errors, 1)
		}
		if result.OK {
			atomic.AddInt64(&results, 1)
		}
		if result.OK && item.firstOnly {
			item.task.Canceler()
		}

		select {
		case resultCh <- newResult(result.OK, result.Err, result.ZombieResult):
		case <-item.ctx.Done():
		}
	})
	if err != nil {
		if e.capacity != nil {
			e.capacity.Release(threads)
		}
		return nil, err
	}

	go func() {
		defer close(resultCh)
		defer pool.Release()
		if e.capacity != nil {
			defer e.capacity.Release(threads)
		}
		defer func() {
			ctx.emitStats(types.Stats{
				Engine:   e.Name(),
				Task:     task.Type(),
				Targets:  int64(len(task.Targets)),
				Tasks:    int64(len(ztasks)),
				Requests: atomic.LoadInt64(&requests),
				Results:  atomic.LoadInt64(&results),
				Errors:   atomic.LoadInt64(&errors),
				Duration: time.Since(started),
			})
		}()

		var wg sync.WaitGroup
		for _, ztask := range ztasks {
			select {
			case <-ctx.Context().Done():
				wg.Wait()
				return
			default:
			}

			wg.Add(1)
			if err := pool.Invoke(&workItem{
				ctx:       ctx.Context(),
				wg:        &wg,
				task:      ztask,
				firstOnly: ctx.firstOnly,
			}); err != nil {
				atomic.AddInt64(&errors, 1)
				wg.Done()
				select {
				case resultCh <- newResult(false, err, ztask.ZombieResult):
				case <-ctx.Context().Done():
					return
				}
			}
		}
		wg.Wait()
	}()

	return resultCh, nil
}

type workItem struct {
	ctx       context.Context
	wg        *sync.WaitGroup
	task      *zombiepkg.Task
	firstOnly bool
}

func runTask(ctx context.Context, task *zombiepkg.Task) *zombiepkg.Result {
	select {
	case <-ctx.Done():
		return zombiepkg.NewResult(task, ctx.Err())
	case <-task.Context.Done():
		return zombiepkg.NewResult(task, task.Context.Err())
	default:
	}

	resultCh := make(chan *zombiepkg.Result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- zombiepkg.NewResult(task, fmt.Errorf("panic: %v\n%s", r, debug.Stack()))
			}
		}()

		if task.Mod == types.ZombieModUnauth {
			resultCh <- zombiecore.Unauth(task)
			return
		}
		resultCh <- zombiecore.Brute(task)
	}()

	timeout := time.Duration(task.Timeout*2) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		return zombiepkg.NewResult(task, ctx.Err())
	case <-task.Context.Done():
		return zombiepkg.NewResult(task, task.Context.Err())
	case <-time.After(timeout):
		task.Canceler()
		return zombiepkg.NewResult(task, fmt.Errorf("task timed out after %s", timeout))
	}
}

func isZombieExecutionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, zombiepkg.ErrorWrongUserOrPwd) ||
		errors.Is(err, zombiepkg.NotImplUnauthorized) ||
		errors.Is(err, zombiecore.ErrNoUnauth) {
		return false
	}
	return true
}

func expandTasks(ctx *Context, task *WeakpassTask) []*zombiepkg.Task {
	var tasks []*zombiepkg.Task
	for _, target := range task.Targets {
		target = normalizeTarget(target)
		if target.Service == "" {
			continue
		}

		targetCtx, cancel := context.WithCancel(ctx.Context())
		if !ctx.noUnauth {
			tasks = append(tasks, newZombieTask(targetCtx, cancel, ctx.timeout, target, "", "", types.ZombieModUnauth))
		}

		auths := authPairs(ctx, task, target)
		for _, auth := range auths {
			tasks = append(tasks, newZombieTask(targetCtx, cancel, ctx.timeout, target, auth.Username, auth.Password, types.ZombieModBrute))
			if ctx.firstOnly && target.Username != "" && target.Password != "" {
				break
			}
		}
	}
	return tasks
}

func authPairs(ctx *Context, task *WeakpassTask, target Target) []Auth {
	if len(task.Auths) > 0 {
		return task.Auths
	}
	if target.Username != "" || target.Password != "" {
		return []Auth{{Username: target.Username, Password: target.Password}}
	}

	users := task.Users
	if len(users) == 0 {
		users = zombiepkg.UseDefaultUser(target.Service, ctx.top)
	}

	passwords := task.Passwords
	if len(passwords) == 0 {
		passwords = zombiepkg.UseDefaultPassword(target.Service, ctx.top)
	}

	auths := make([]Auth, 0, len(users)*len(passwords))
	for _, user := range users {
		for _, password := range passwords {
			auths = append(auths, Auth{Username: user, Password: password})
		}
	}
	return auths
}

func normalizeTarget(target Target) Target {
	target.Service = strings.ToLower(strings.TrimSpace(target.Service))
	if target.Service == "" {
		return target
	}
	if service, ok := zombiepkg.Services.Get(target.Service); ok {
		target.Service = service.Name
	}
	if target.Port == "" {
		target.Port = zombiepkg.Services.DefaultPort(target.Service)
	}
	if target.Scheme == "" {
		target.Scheme = target.Service
	}
	return target
}

func newZombieTask(ctx context.Context, cancel context.CancelFunc, timeout int, target Target, username, password string, mod types.ZombieTaskMod) *zombiepkg.Task {
	return &zombiepkg.Task{
		ZombieResult: &types.ZombieResult{
			IP:       target.IP,
			Port:     target.Port,
			Service:  target.Service,
			Scheme:   target.Scheme,
			Username: username,
			Password: password,
			Param:    target.Param,
			Mod:      mod,
		},
		Timeout:  timeout,
		Context:  ctx,
		Canceler: cancel,
	}
}

func (e *Engine) Weakpass(ctx *Context, task *WeakpassTask) ([]*types.ZombieResult, error) {
	resultCh, err := e.WeakpassStream(ctx, task)
	if err != nil {
		return nil, err
	}

	var results []*types.ZombieResult
	for result := range resultCh {
		results = append(results, result)
	}
	return results, nil
}

func (e *Engine) WeakpassStream(ctx *Context, task *WeakpassTask) (<-chan *types.ZombieResult, error) {
	resultCh, err := e.Execute(ctx, task)
	if err != nil {
		return nil, err
	}

	zombieResultCh := make(chan *types.ZombieResult, 100)
	go func() {
		defer close(zombieResultCh)
		for result := range resultCh {
			if data, ok := types.ResultData[*types.ZombieResult](result); result.Success() && ok && data != nil {
				zombieResultCh <- data
			}
		}
	}()
	return zombieResultCh, nil
}
