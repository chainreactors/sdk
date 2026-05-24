package cyberhub

import (
	"context"
	"sync"
	"time"

	"github.com/chainreactors/sdk/pkg/types"
)

// Provider 是 cyberhub 数据源
type Provider struct {
	url     string
	apiKey  string
	timeout time.Duration
	filter  *ExportFilter

	once sync.Once
	cli  *client
}

// NewProvider 创建 cyberhub 数据源
func NewProvider(url, apiKey string) *Provider {
	return &Provider{
		url:     url,
		apiKey:  apiKey,
		timeout: 10 * time.Second,
	}
}

// WithFilter 设置导出筛选条件
func (p *Provider) WithFilter(f *ExportFilter) *Provider {
	p.filter = f
	return p
}

// WithTimeout 设置请求超时
func (p *Provider) WithTimeout(d time.Duration) *Provider {
	p.timeout = d
	return p
}

func (p *Provider) client() *client {
	p.once.Do(func() {
		p.cli = newClient(p.url, p.apiKey, p.timeout)
	})
	return p.cli
}

// Fingers 导出指纹与别名数据
func (p *Provider) Fingers(ctx context.Context) (types.Fingers, []*types.Alias, error) {
	return p.client().exportFingers(ctx, p.filter)
}

// POCs 导出 POC 模板数据
func (p *Provider) POCs(ctx context.Context) ([]*types.Template, error) {
	responses, err := p.client().exportPOCs(ctx, p.filter)
	if err != nil {
		return nil, err
	}
	tpls := make([]*types.Template, 0, len(responses))
	for _, resp := range responses {
		if resp.Template != nil {
			tpls = append(tpls, resp.Template)
		}
	}
	return tpls, nil
}
