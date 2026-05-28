package fingers

import (
	"context"
	"fmt"

	"github.com/chainreactors/sdk/pkg/types"
)

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{}
}

// Config Fingers SDK 配置
type Config struct {
	Providers     []types.Provider
	EnableEngines []string
	FullFingers   FullFingers
	MatchDetail   bool
}

// Validate 验证配置
func (c *Config) Validate() error {
	return nil
}

// SetEnableEngines 设置启用的引擎列表
func (c *Config) SetEnableEngines(engines []string) *Config {
	c.EnableEngines = engines
	return c
}

// WithProvider 追加数据源，支持多次调用自动合并
func (c *Config) WithProvider(providers ...types.Provider) *Config {
	c.Providers = append(c.Providers, providers...)
	return c
}

// WithFingers 设置指纹数据
func (c *Config) WithFingers(fingers types.Fingers) *Config {
	aliases := c.FullFingers.Aliases()
	c.FullFingers = (FullFingers{}).Merge(fingers, aliases)
	return c
}

// WithAliases 设置别名数据
func (c *Config) WithAliases(aliases []*types.Alias) *Config {
	fingers := c.FullFingers.Fingers()
	c.FullFingers = (FullFingers{}).Merge(fingers, aliases)
	return c
}

// WithMatchDetail enables matcher metadata on match results.
func (c *Config) WithMatchDetail() *Config {
	c.MatchDetail = true
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
	if len(c.Providers) > 0 {
		for _, p := range c.Providers {
			fingersData, aliases, err := p.Fingers(ctx)
			if err != nil {
				return err
			}
			c.FullFingers = c.FullFingers.Merge(fingersData, aliases)
		}
		return nil
	}

	return fmt.Errorf("no data source configured: use WithProvider() or WithFingers()")
}

type FullFinger struct {
	Finger *types.Finger
	Alias  *types.Alias
}

type FullFingers struct {
	Items map[string]*FullFinger
}

// Fingers returns finger list from FullFingers.
func (f FullFingers) Fingers() types.Fingers {
	if len(f.Items) == 0 {
		return nil
	}
	fingers := make(types.Fingers, 0, len(f.Items))
	for _, item := range f.Items {
		if item == nil || item.Finger == nil {
			continue
		}
		fingers = append(fingers, item.Finger)
	}
	return fingers
}

// Aliases returns alias list from FullFingers.
func (f FullFingers) Aliases() []*types.Alias {
	if len(f.Items) == 0 {
		return nil
	}
	aliases := make([]*types.Alias, 0, len(f.Items))
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
func (f FullFingers) Merge(fingers types.Fingers, aliases []*types.Alias) FullFingers {
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
