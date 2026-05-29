package cyberhub

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/sdk/pkg/types"
	"gopkg.in/yaml.v3"
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

// Filter 返回当前筛选条件
func (p *Provider) Filter() *ExportFilter {
	return p.filter
}

func (p *Provider) client() *client {
	p.once.Do(func() {
		p.cli = newClient(p.url, p.apiKey, p.timeout)
	})
	return p.cli
}

// Fingers 导出指纹与别名数据。当 ExportFilter.Draft 为 true 且记录含
// RawContentDraft 时，优先从 draft YAML 解析指纹替换 approved 版本。
func (p *Provider) Fingers(ctx context.Context) (types.Fingers, []*types.Alias, error) {
	records, err := p.client().exportFingers(ctx, p.filter)
	if err != nil {
		return nil, nil, err
	}
	useDraft := p.filter != nil && p.filter.Draft
	var allFingers types.Fingers
	var allAliases []*types.Alias
	for _, r := range records {
		finger := r.Finger
		if useDraft && r.RawContentDraft != "" {
			if parsed := parseFinger(r.RawContentDraft); parsed != nil {
				finger = parsed
			}
		}
		if finger != nil {
			allFingers = append(allFingers, finger)
		}
		if r.Alias != nil {
			allAliases = append(allAliases, r.Alias)
		}
	}
	return allFingers, allAliases, nil
}

func parseFinger(raw string) *types.Finger {
	var finger types.Finger
	if err := yaml.Unmarshal([]byte(raw), &finger); err == nil && finger.Name != "" {
		return &finger
	}
	return nil
}

// ExportFingers 导出完整指纹记录，包含 RawContent 与 RawContentDraft。
// 自动修正 Engine 字段：CyberHub 可能将 xray 数据标记为 fingerprinthub，
// 通过 Finger.Tags 中的 neutron/xray/ai_converted 标签识别并修正为 xray。
func (p *Provider) ExportFingers(ctx context.Context) ([]FingerprintExport, error) {
	records, err := p.client().exportFingers(ctx, p.filter)
	if err != nil {
		return nil, err
	}
	for i := range records {
		records[i].Engine = resolveEngine(records[i].Engine, records[i].Finger)
	}
	return records, nil
}

// resolveEngine 根据 tags 修正 CyberHub 返回的 engine 字段。
func resolveEngine(engine string, finger *types.Finger) string {
	if engine == "xray" || engine == "fingers" || engine == "" {
		return engine
	}
	if finger == nil {
		return engine
	}
	for _, tag := range finger.Tags {
		switch strings.ToLower(tag) {
		case "neutron", "xray", "ai_converted":
			return "xray"
		}
	}
	return engine
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
