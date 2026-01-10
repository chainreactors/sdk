package fingers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chainreactors/fingers/common"
	sdk "github.com/chainreactors/sdk/pkg"
	"github.com/chainreactors/utils/httputils"
)

// ========================================
// Context 实现
// ========================================

// Context Fingers 上下文
type Context struct {
	ctx context.Context
}

// NewContext 创建 Fingers 上下文
func NewContext() *Context {
	return &Context{
		ctx: context.Background(),
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) WithTimeout(timeout time.Duration) sdk.Context {
	ctx, _ := context.WithTimeout(c.ctx, timeout)
	return &Context{
		ctx: ctx,
	}
}

func (c *Context) WithCancel() (sdk.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	return &Context{
		ctx: ctx,
	}, cancel
}

// ========================================
// Result 实现
// ========================================

// MatchResult 指纹匹配结果
type MatchResult struct {
	success    bool
	err        error
	frameworks common.Frameworks
}

func (r *MatchResult) Success() bool {
	return r.success
}

func (r *MatchResult) Error() error {
	return r.err
}

func (r *MatchResult) Data() interface{} {
	return r.frameworks
}

// Frameworks 获取匹配到的指纹
func (r *MatchResult) Frameworks() common.Frameworks {
	return r.frameworks
}

// HasMatch 是否匹配到指纹
func (r *MatchResult) HasMatch() bool {
	return len(r.frameworks) > 0
}

// Count 匹配到的指纹数量
func (r *MatchResult) Count() int {
	return len(r.frameworks)
}

// ========================================
// Task 实现
// ========================================

// MatchTask 指纹匹配任务
type MatchTask struct {
	Data []byte // HTTP 响应原始数据
}

// NewMatchTask 创建匹配任务
func NewMatchTask(data []byte) *MatchTask {
	return &MatchTask{Data: data}
}

// NewMatchTaskFromResponse 从 HTTP Response 创建任务
func NewMatchTaskFromResponse(resp *http.Response) *MatchTask {
	data := httputils.ReadRaw(resp)
	return &MatchTask{Data: data}
}

func (t *MatchTask) Type() string {
	return "match"
}

func (t *MatchTask) Validate() error {
	if len(t.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}
