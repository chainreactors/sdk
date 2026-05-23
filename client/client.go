package client

import (
	"errors"
	"fmt"
	"sync"

	"github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/gogo"
	"github.com/chainreactors/sdk/neutron"
	"github.com/chainreactors/sdk/pkg/association"
	"github.com/chainreactors/sdk/pkg/cyberhub"
	"github.com/chainreactors/sdk/spray"
	"github.com/chainreactors/sdk/zombie"
)

type Option func(*options)

type options struct {
	provider         *cyberhub.Provider
	resourceProvider func(string) []byte

	fingersConfig *fingers.Config
	neutronConfig *neutron.Config
	gogoConfig    *gogo.Config
	sprayConfig   *spray.Config
	zombieConfig  *zombie.Config
}

func WithProvider(p *cyberhub.Provider) Option {
	return func(o *options) { o.provider = p }
}

func WithResourceProvider(rp func(string) []byte) Option {
	return func(o *options) { o.resourceProvider = rp }
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

type Client struct {
	opts options
	mu   sync.Mutex

	fingers *fingers.Engine
	neutron *neutron.Engine
	gogo    *gogo.GogoEngine
	spray   *spray.SprayEngine
	zombie  *zombie.Engine
}

func New(opts ...Option) *Client {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{opts: o}
}

// ========================================
// 依赖解析
// ========================================

func (c *Client) ensureFingers() error {
	if c.fingers != nil {
		return nil
	}

	cfg := c.opts.fingersConfig
	if cfg == nil {
		cfg = fingers.NewConfig()
		if c.opts.provider != nil {
			cfg.WithProvider(c.opts.provider)
		}
	}

	eng, err := fingers.NewEngine(cfg)
	if err != nil {
		return fmt.Errorf("create fingers engine: %w", err)
	}
	c.fingers = eng
	return nil
}

func (c *Client) ensureNeutron() error {
	if c.neutron != nil {
		return nil
	}

	cfg := c.opts.neutronConfig
	if cfg == nil {
		cfg = neutron.NewConfig()
		if c.opts.provider != nil {
			cfg.WithProvider(c.opts.provider)
		}
	}

	eng, err := neutron.NewEngine(cfg)
	if err != nil {
		return fmt.Errorf("create neutron engine: %w", err)
	}
	c.neutron = eng
	return nil
}

func (c *Client) ensureGogo() error {
	if c.gogo != nil {
		return nil
	}

	if err := c.ensureFingers(); err != nil {
		return fmt.Errorf("gogo requires fingers: %w", err)
	}
	if err := c.ensureNeutron(); err != nil {
		return fmt.Errorf("gogo requires neutron: %w", err)
	}

	cfg := c.opts.gogoConfig
	if cfg == nil {
		cfg = gogo.NewConfig()
	}
	if cfg.FingersEngine == nil {
		cfg.WithFingersEngine(c.fingers)
	}
	if cfg.NeutronEngine == nil {
		cfg.WithNeutronEngine(c.neutron)
	}
	if cfg.Provider == nil && c.opts.provider != nil {
		cfg.WithProvider(c.opts.provider)
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}

	eng := gogo.NewGogoEngine(cfg)
	if err := eng.Init(); err != nil {
		return fmt.Errorf("init gogo engine: %w", err)
	}
	c.gogo = eng
	return nil
}

func (c *Client) ensureSpray() error {
	if c.spray != nil {
		return nil
	}

	if err := c.ensureFingers(); err != nil {
		return fmt.Errorf("spray requires fingers: %w", err)
	}

	cfg := c.opts.sprayConfig
	if cfg == nil {
		cfg = spray.NewConfig()
	}
	if cfg.FingersEngine == nil {
		cfg.WithFingersEngine(c.fingers)
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}

	eng := spray.NewSprayEngine(cfg)
	if err := eng.Init(); err != nil {
		return fmt.Errorf("init spray engine: %w", err)
	}
	c.spray = eng
	return nil
}

func (c *Client) ensureZombie() error {
	if c.zombie != nil {
		return nil
	}

	cfg := c.opts.zombieConfig
	if cfg == nil {
		cfg = zombie.NewConfig()
	}
	if cfg.ResourceProvider == nil && c.opts.resourceProvider != nil {
		cfg.WithResourceProvider(c.opts.resourceProvider)
	}

	eng := zombie.NewEngine(cfg)
	if err := eng.Init(); err != nil {
		return fmt.Errorf("init zombie engine: %w", err)
	}
	c.zombie = eng
	return nil
}

// ========================================
// 类型安全的访问方法
// ========================================

func (c *Client) Fingers() (*fingers.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureFingers(); err != nil {
		return nil, err
	}
	return c.fingers, nil
}

func (c *Client) Gogo() (*gogo.GogoEngine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureGogo(); err != nil {
		return nil, err
	}
	return c.gogo, nil
}

func (c *Client) Spray() (*spray.SprayEngine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureSpray(); err != nil {
		return nil, err
	}
	return c.spray, nil
}

func (c *Client) Neutron() (*neutron.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureNeutron(); err != nil {
		return nil, err
	}
	return c.neutron, nil
}

func (c *Client) Zombie() (*zombie.Engine, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureZombie(); err != nil {
		return nil, err
	}
	return c.zombie, nil
}

// ========================================
// 关联索引
// ========================================

// Index returns the association index from the GoGo engine.
// Returns nil if GoGo has not been initialized yet.
func (c *Client) Index() *association.Index {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gogo != nil {
		return c.gogo.Index()
	}
	return nil
}

// BuildIndex constructs an association index from the fingers and neutron
// engines, initializing them if needed. Unlike Index(), this does not
// require GoGo to be initialized.
func (c *Client) BuildIndex() (*association.Index, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureFingers(); err != nil {
		return nil, err
	}
	if err := c.ensureNeutron(); err != nil {
		return nil, err
	}

	idx := association.NewIndex()
	idx.BuildWithFingers(c.fingers.Fingers(), c.fingers.Aliases(), c.neutron.Get())
	return idx, nil
}

// ========================================
// 生命周期
// ========================================

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	if c.gogo != nil {
		errs = append(errs, c.gogo.Close())
	}
	if c.spray != nil {
		errs = append(errs, c.spray.Close())
	}
	if c.neutron != nil {
		errs = append(errs, c.neutron.Close())
	}
	if c.fingers != nil {
		errs = append(errs, c.fingers.Close())
	}
	if c.zombie != nil {
		errs = append(errs, c.zombie.Close())
	}

	c.gogo = nil
	c.spray = nil
	c.neutron = nil
	c.fingers = nil
	c.zombie = nil

	return errors.Join(errs...)
}
