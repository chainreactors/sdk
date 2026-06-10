package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/gogo"
	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/association"
	"github.com/chainreactors/sdk/pkg/types"
	"github.com/chainreactors/sdk/proton"
	"github.com/chainreactors/sdk/spray"
	"github.com/chainreactors/sdk/zombie"
)

type Option func(*options)

type options struct {
	providers        []types.Provider
	resourceProvider func(string) []byte
	indexOptions     *association.IndexOptions
	proxy            []string // 全局默认代理，下沉到各引擎（可被引擎 Config / Context 覆盖）

	fingersConfig *fingers.Config
	neutronConfig *neutron.Config
	gogoConfig    *gogo.Config
	sprayConfig   *spray.Config
	zombieConfig  *zombie.Config
	protonConfig  *proton.Config
}

func WithProvider(providers ...types.Provider) Option {
	return func(o *options) { o.providers = append(o.providers, providers...) }
}

func WithResourceProvider(rp func(string) []byte) Option {
	return func(o *options) { o.resourceProvider = rp }
}

// WithProxy 设置全局默认代理（支持多级代理链），应用于 gogo / spray / zombie
// 所有引擎。各引擎 Config.WithProxy 或 Context.SetProxy 可覆盖此默认值。
// 支持 proxyclient 的全部协议，例如：
//
//	client.New(client.WithProxy("socks5://127.0.0.1:1080"))
//	client.New(client.WithProxy("http://a:8080", "socks5://b:1080")) // 代理链
func WithProxy(proxies ...string) Option {
	return func(o *options) { o.proxy = proxies }
}

// WithIndex enables the association index on this client.
// Pass nil to use default IndexOptions; pass a pointer to customize.
func WithIndex(opts *association.IndexOptions) Option {
	return func(o *options) {
		if opts == nil {
			opts = &association.IndexOptions{}
		}
		o.indexOptions = opts
	}
}

func WithFingersConfig(cfg *fingers.Config) Option {
	return func(o *options) { o.fingersConfig = cfg }
}

func WithNeutronConfig(cfg *neutron.Config) Option {
	return func(o *options) { o.neutronConfig = cfg }
}

func WithGogoConfig(cfg *gogo.Config) Option {
	return func(o *options) { o.gogoConfig = cfg }
}

func WithSprayConfig(cfg *spray.Config) Option {
	return func(o *options) { o.sprayConfig = cfg }
}

func WithZombieConfig(cfg *zombie.Config) Option {
	return func(o *options) { o.zombieConfig = cfg }
}

func WithProtonConfig(cfg *proton.Config) Option {
	return func(o *options) { o.protonConfig = cfg }
}

type Client struct {
	opts options
	mu   sync.Mutex

	fingers lazy[*fingers.Engine]
	neutron lazy[*neutron.Engine]
	gogo    lazy[*gogo.GogoEngine]
	spray   lazy[*spray.SprayEngine]
	zombie  lazy[*zombie.Engine]
	proton  lazy[*proton.Engine]
	index   lazy[*association.Index]
}

func New(opts ...Option) *Client {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	c := &Client{opts: o}
	c.fingers.initFn = c.initFingers
	c.neutron.initFn = c.initNeutron
	c.gogo.initFn = c.initGogo
	c.spray.initFn = c.initSpray
	c.zombie.initFn = c.initZombie
	c.proton.initFn = c.initProton
	c.index.initFn = c.initIndex
	return c
}

// ========================================
// 引擎工厂（自动加载：首次访问时按需创建）
// ========================================

func (c *Client) initFingers() (*fingers.Engine, error) {
	cfg := c.opts.fingersConfig
	if cfg == nil {
		cfg = fingers.NewConfig()
		if len(c.opts.providers) > 0 {
			cfg.WithProvider(c.opts.providers...)
		}
	}
	if len(cfg.Proxy) == 0 && len(c.opts.proxy) > 0 {
		cfg.Proxy = c.opts.proxy
	}

	eng, err := fingers.NewEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("create fingers engine: %w", err)
	}
	return eng, nil
}

func (c *Client) initNeutron() (*neutron.Engine, error) {
	cfg := c.opts.neutronConfig
	if cfg == nil {
		cfg = neutron.NewConfig()
		if len(c.opts.providers) > 0 {
			cfg.WithProvider(c.opts.providers...)
		}
	}
	if len(cfg.Proxy) == 0 && len(c.opts.proxy) > 0 {
		cfg.Proxy = c.opts.proxy
	}

	eng, err := neutron.NewEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("create neutron engine: %w", err)
	}
	return eng, nil
}

func (c *Client) initGogo() (*gogo.GogoEngine, error) {
	fingersEng, err := c.fingers.get()
	if err != nil {
		return nil, fmt.Errorf("gogo requires fingers: %w", err)
	}
	neutronEng, err := c.neutron.get()
	if err != nil {
		return nil, fmt.Errorf("gogo requires neutron: %w", err)
	}

	cfg := c.opts.gogoConfig
	if cfg == nil {
		cfg = gogo.NewConfig()
	}
	if cfg.FingersEngine == nil {
		cfg.WithFingersEngine(fingersEng)
	}
	if cfg.NeutronEngine == nil {
		cfg.WithNeutronEngine(neutronEng)
	}
	if len(cfg.Providers) == 0 && len(c.opts.providers) > 0 {
		cfg.WithProvider(c.opts.providers...)
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}
	if len(cfg.Proxy) == 0 && len(c.opts.proxy) > 0 {
		cfg.Proxy = c.opts.proxy
	}

	eng := gogo.NewGogoEngine(cfg)
	if err := eng.Init(); err != nil {
		return nil, fmt.Errorf("init gogo engine: %w", err)
	}
	return eng, nil
}

func (c *Client) initSpray() (*spray.SprayEngine, error) {
	fingersEng, err := c.fingers.get()
	if err != nil {
		return nil, fmt.Errorf("spray requires fingers: %w", err)
	}

	cfg := c.opts.sprayConfig
	if cfg == nil {
		cfg = spray.NewConfig()
	}
	if len(cfg.Providers) == 0 && len(c.opts.providers) > 0 {
		cfg.WithProvider(c.opts.providers...)
	}
	if cfg.FingersEngine == nil {
		cfg.WithFingersEngine(fingersEng)
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}
	if len(cfg.Proxy) == 0 && len(c.opts.proxy) > 0 {
		cfg.Proxy = c.opts.proxy
	}

	eng := spray.NewSprayEngine(cfg)
	if err := eng.Init(); err != nil {
		return nil, fmt.Errorf("init spray engine: %w", err)
	}
	return eng, nil
}

func (c *Client) initZombie() (*zombie.Engine, error) {
	cfg := c.opts.zombieConfig
	if cfg == nil {
		cfg = zombie.NewConfig()
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}
	if len(cfg.Proxy) == 0 && len(c.opts.proxy) > 0 {
		cfg.Proxy = c.opts.proxy
	}

	eng := zombie.NewEngine(cfg)
	if err := eng.Init(); err != nil {
		return nil, fmt.Errorf("init zombie engine: %w", err)
	}
	return eng, nil
}

func (c *Client) initProton() (*proton.Engine, error) {
	cfg := c.opts.protonConfig
	if cfg == nil {
		cfg = proton.NewConfig()
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}

	eng := proton.NewEngine(cfg)
	if err := eng.Init(); err != nil {
		return nil, fmt.Errorf("init proton engine: %w", err)
	}
	return eng, nil
}

func (c *Client) initIndex() (*association.Index, error) {
	if c.opts.indexOptions == nil {
		return nil, nil
	}

	fingersEng, err := c.fingers.get()
	if err != nil {
		return nil, fmt.Errorf("index requires fingers: %w", err)
	}
	neutronEng, err := c.neutron.get()
	if err != nil {
		return nil, fmt.Errorf("index requires neutron: %w", err)
	}

	idx := association.NewIndexWithOptions(*c.opts.indexOptions)
	idx.BuildWithFingers(fingersEng.Fingers(), fingersEng.Aliases(), neutronEng.Get())
	return idx, nil
}

// ========================================
// 引擎访问（自动加载：首次调用触发初始化）
// ========================================

func (c *Client) Fingers() (*fingers.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fingers.get()
}

func (c *Client) Neutron() (*neutron.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.neutron.get()
}

func (c *Client) Gogo() (*gogo.GogoEngine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gogo.get()
}

func (c *Client) Spray() (*spray.SprayEngine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.spray.get()
}

func (c *Client) Zombie() (*zombie.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.zombie.get()
}

func (c *Client) Proton() (*proton.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.proton.get()
}

// ========================================
// 手动加载（人工配置后显式预加载，提前捕获错误）
// ========================================

// Load 按名称预加载指定引擎，在启动阶段提前发现配置错误。
// 不传参数则加载所有已配置的引擎。
//
//	c.Load("proton", "zombie")           // 仅加载指定引擎
//	c.Load()                              // 加载全部已配置引擎
func (c *Client) Load(names ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(names) == 0 {
		return c.loadConfigured()
	}
	for _, name := range names {
		if err := c.loadByName(name); err != nil {
			return fmt.Errorf("load %s: %w", name, err)
		}
	}
	return nil
}

func (c *Client) loadByName(name string) error {
	switch name {
	case "fingers":
		_, err := c.fingers.get()
		return err
	case "neutron":
		_, err := c.neutron.get()
		return err
	case "gogo":
		_, err := c.gogo.get()
		return err
	case "spray":
		_, err := c.spray.get()
		return err
	case "zombie":
		_, err := c.zombie.get()
		return err
	case "proton":
		_, err := c.proton.get()
		return err
	default:
		return fmt.Errorf("unknown engine: %s", name)
	}
}

func (c *Client) loadConfigured() error {
	if c.opts.fingersConfig != nil || len(c.opts.providers) > 0 {
		if _, err := c.fingers.get(); err != nil {
			return fmt.Errorf("load fingers: %w", err)
		}
	}
	if c.opts.neutronConfig != nil || len(c.opts.providers) > 0 {
		if _, err := c.neutron.get(); err != nil {
			return fmt.Errorf("load neutron: %w", err)
		}
	}
	if c.opts.gogoConfig != nil {
		if _, err := c.gogo.get(); err != nil {
			return fmt.Errorf("load gogo: %w", err)
		}
	}
	if c.opts.sprayConfig != nil {
		if _, err := c.spray.get(); err != nil {
			return fmt.Errorf("load spray: %w", err)
		}
	}
	if c.opts.zombieConfig != nil {
		if _, err := c.zombie.get(); err != nil {
			return fmt.Errorf("load zombie: %w", err)
		}
	}
	if c.opts.protonConfig != nil {
		if _, err := c.proton.get(); err != nil {
			return fmt.Errorf("load proton: %w", err)
		}
	}
	if c.opts.indexOptions != nil {
		if _, err := c.index.get(); err != nil {
			return fmt.Errorf("load index: %w", err)
		}
	}
	return nil
}

// ========================================
// 关联索引
// ========================================

// Index returns the association index, initializing it if WithIndex
// was set. Returns nil if the index is not enabled.
func (c *Client) Index() (*association.Index, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.index.get()
}

// Lookup queries the association index with the given query.
// Shorthand for c.Index() + idx.Lookup(q).
func (c *Client) Lookup(q *association.Query) (*association.QueryResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx, err := c.index.get()
	if err != nil {
		return nil, err
	}
	if idx == nil {
		return nil, fmt.Errorf("index not enabled: use WithIndex() when creating client")
	}
	return idx.Lookup(q), nil
}

// LookupResult extracts association terms from an engine result and
// queries the index. Accepts GOGOResult, SprayResult, or ZombieResult
// wrapped in a types.Result.
func (c *Client) LookupResult(r types.Result) (*association.QueryResult, error) {
	return c.Lookup(association.QueryFromResult(r))
}

// LookupByFinger queries the index by fingerprint names.
func (c *Client) LookupByFinger(names ...string) (*association.QueryResult, error) {
	return c.Lookup(association.NewQuery().WithFingers(names...))
}

// LookupByCVE queries the index by CVE IDs.
func (c *Client) LookupByCVE(cves ...string) (*association.QueryResult, error) {
	return c.Lookup(association.NewQuery().WithCVEs(cves...))
}

// BuildIndex constructs a standalone association index from the
// Provider, independent of the client's index lifecycle. Useful when
// you need an index with different options than the client's.
func (c *Client) BuildIndex(ctx context.Context, opts ...association.IndexOption) (*association.Index, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fingersEng, err := c.fingers.get()
	if err != nil {
		return nil, err
	}
	neutronEng, err := c.neutron.get()
	if err != nil {
		return nil, err
	}

	idx := association.NewIndex(opts...)
	idx.BuildWithFingers(fingersEng.Fingers(), fingersEng.Aliases(), neutronEng.Get())
	return idx, nil
}

// ========================================
// 生命周期
// ========================================

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return errors.Join(
		c.gogo.close(),
		c.spray.close(),
		c.neutron.close(),
		c.fingers.close(),
		c.zombie.close(),
		c.proton.close(),
	)
}
