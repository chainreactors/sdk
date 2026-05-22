package cyberhub

import (
	"context"
	"time"

	"github.com/chainreactors/fingers/alias"
	"github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/neutron/templates"
)

// Provider 是 cyberhub 数据源
type Provider struct {
	url     string
	apiKey  string
	timeout time.Duration
	filter  *ExportFilter
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

// Fingers 导出指纹与别名数据
func (p *Provider) Fingers(ctx context.Context) (fingers.Fingers, []*alias.Alias, error) {
	c := newClient(p.url, p.apiKey, p.timeout)
	return c.exportFingers(ctx, p.filter)
}

// POCs 导出 POC 模板数据
func (p *Provider) POCs(ctx context.Context) ([]*templates.Template, error) {
	c := newClient(p.url, p.apiKey, p.timeout)
	responses, err := c.exportPOCs(ctx, p.filter)
	if err != nil {
		return nil, err
	}
	tpls := make([]*templates.Template, 0, len(responses))
	for _, resp := range responses {
		if resp.Template != nil {
			tpls = append(tpls, resp.Template)
		}
	}
	return tpls, nil
}
