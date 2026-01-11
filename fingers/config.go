package fingers

import (
	"context"
	"fmt"

	"github.com/chainreactors/fingers/alias"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

// NewConfig 创建默认配置
func NewConfig() *Config {
	base := cyberhub.NewConfig()
	return &Config{
		Config:        *base,
		EnableEngines: nil,
		FullFingers:   FullFingers{},
	}
}

// ========================================
// Config 配置
// ========================================

// Config Fingers SDK 配置
type Config struct {
	cyberhub.Config

	// 引擎配置
	EnableEngines []string
	FullFingers   FullFingers
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 如果配置了 Cyberhub URL，必须提供 API Key
	if c.CyberhubURL != "" && c.APIKey == "" {
		return fmt.Errorf("api_key is required when cyberhub_url is set")
	}
	if err := c.Config.Validate(); err != nil {
		return err
	}
	return nil
}

// IsRemoteEnabled 是否启用远程加载
func (c *Config) IsRemoteEnabled() bool {
	return c.CyberhubURL != "" && c.APIKey != ""
}

// SetEnableEngines 设置启用的引擎列表
func (c *Config) SetEnableEngines(engines []string) *Config {
	c.EnableEngines = engines
	return c
}

// WithCyberhub 设置远程加载配置（不立即拉取）
func (c *Config) WithCyberhub(url, apiKey string) *Config {
	c.CyberhubURL = url
	c.APIKey = apiKey
	c.Filename = ""
	c.FullFingers = FullFingers{}
	return c
}

// WithLocalFile 设置本地文件加载（不立即读取）
func (c *Config) WithLocalFile(filename string) *Config {
	c.Filename = filename
	c.CyberhubURL = ""
	c.APIKey = ""
	c.FullFingers = FullFingers{}
	return c
}

// WithFingers 设置指纹数据
func (c *Config) WithFingers(fingers fingersEngine.Fingers) *Config {
	aliases := c.FullFingers.Aliases()
	c.FullFingers = (FullFingers{}).Merge(fingers, aliases)
	return c
}

// WithAliases 设置别名数据
func (c *Config) WithAliases(aliases []*alias.Alias) *Config {
	fingers := c.FullFingers.Fingers()
	c.FullFingers = (FullFingers{}).Merge(fingers, aliases)
	return c
}

// WithFilter filters current FullFingers using predicate.
func (c *Config) WithFilter(predicate func(*FullFinger) bool) *Config {
	if c == nil {
		return c
	}
	c.FullFingers = c.FullFingers.Filter(predicate)
	return c
}

// Load 执行数据加载
func (c *Config) Load(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.FullFingers.Len() > 0 {
		return nil
	}
	if c.Filename != "" {
		fingers, err := loadFingersFromPath(c.Filename)
		if err != nil {
			return err
		}
		c.FullFingers = (FullFingers{}).Merge(fingers, nil)
		return nil
	}
	if c.IsRemoteEnabled() {
		client := cyberhub.NewClient(c.CyberhubURL, c.APIKey, c.Timeout)
		fingersData, aliases, err := client.ExportFingers(ctx, "", c.ExportFilter)
		if err != nil {
			return err
		}
		c.FullFingers = (FullFingers{}).Merge(fingersData, aliases)
		return nil
	}
	return fmt.Errorf("no data source configured")
}

type FullFinger struct {
	Finger *fingersEngine.Finger
	Alias  *alias.Alias
}

type FullFingers struct {
	Items map[string]*FullFinger
}

// Fingers returns finger list from FullFingers.
func (f FullFingers) Fingers() fingersEngine.Fingers {
	if len(f.Items) == 0 {
		return nil
	}
	fingers := make(fingersEngine.Fingers, 0, len(f.Items))
	for _, item := range f.Items {
		if item == nil || item.Finger == nil {
			continue
		}
		fingers = append(fingers, item.Finger)
	}
	return fingers
}

// Aliases returns alias list from FullFingers.
func (f FullFingers) Aliases() []*alias.Alias {
	if len(f.Items) == 0 {
		return nil
	}
	aliases := make([]*alias.Alias, 0, len(f.Items))
	for _, item := range f.Items {
		if item == nil || item.Alias == nil {
			continue
		}
		aliases = append(aliases, item.Alias)
	}
	return aliases
}

// Len returns item count.
func (f FullFingers) Len() int {
	return len(f.Items)
}

// Append adds a single FullFinger.
func (f FullFingers) Append(item *FullFinger) FullFingers {
	if item == nil {
		return f
	}
	if f.Items == nil {
		f.Items = make(map[string]*FullFinger)
	}
	if item.Finger != nil && item.Finger.Name != "" {
		f.Items[item.Finger.Name] = item
		return f
	}
	return f
}

// Merge appends fingers and aliases into FullFingers.
func (f FullFingers) Merge(fingers fingersEngine.Fingers, aliases []*alias.Alias) FullFingers {
	if len(fingers) == 0 && len(aliases) == 0 {
		return f
	}
	if f.Items == nil {
		f.Items = make(map[string]*FullFinger)
	}
	for _, finger := range fingers {
		f = f.Append(&FullFinger{Finger: finger})
	}
	for _, aliasItem := range aliases {
		if aliasItem == nil || aliasItem.Name == "" {
			continue
		}
		if item, ok := f.Items[aliasItem.Name]; ok && item != nil {
			item.Alias = aliasItem
		}
	}
	return f
}

// Filter returns a filtered copy of FullFingers using predicate.
func (f FullFingers) Filter(predicate func(*FullFinger) bool) FullFingers {
	if predicate == nil || len(f.Items) == 0 {
		return f
	}
	filtered := FullFingers{
		Items: make(map[string]*FullFinger),
	}
	for key, item := range f.Items {
		if predicate(item) {
			filtered.Items[key] = item
		}
	}
	return filtered
}
