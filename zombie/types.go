package zombie

import (
	"context"
	"fmt"

	"github.com/chainreactors/parsers"
	sdk "github.com/chainreactors/sdk/pkg"
)

type Context struct {
	ctx       context.Context
	threads   int
	timeout   int
	top       int
	firstOnly bool
	noUnauth  bool
}

var _ sdk.Context = (*Context)(nil)

func NewContext() *Context {
	return &Context{
		ctx:       context.Background(),
		threads:   100,
		timeout:   5,
		firstOnly: true,
	}
}

func (c *Context) WithContext(ctx context.Context) *Context {
	return &Context{
		ctx:       ctx,
		threads:   c.threads,
		timeout:   c.timeout,
		top:       c.top,
		firstOnly: c.firstOnly,
		noUnauth:  c.noUnauth,
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) SetThreads(threads int) *Context {
	if threads > 0 {
		c.threads = threads
	}
	return c
}

func (c *Context) SetTimeout(timeout int) *Context {
	if timeout > 0 {
		c.timeout = timeout
	}
	return c
}

func (c *Context) SetTop(top int) *Context {
	if top >= 0 {
		c.top = top
	}
	return c
}

func (c *Context) SetFirstOnly(firstOnly bool) *Context {
	c.firstOnly = firstOnly
	return c
}

func (c *Context) SetNoUnauth(noUnauth bool) *Context {
	c.noUnauth = noUnauth
	return c
}

type Config struct {
	Capacity         int
	ResourceProvider func(string) []byte
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Validate() error {
	return nil
}

// WithCapacity sets the total capacity for concurrent thread usage across all
// simultaneous invocations. When set, each Execute call acquires its thread
// count from this shared bucket and blocks if capacity is exhausted.
func (c *Config) WithCapacity(total int) *Config {
	c.Capacity = total
	return c
}

// WithResourceProvider sets a provider used by the underlying zombie package
// to load templates/keywords/rules. When nil, zombie falls back to its
// standalone embedded defaults.
func (c *Config) WithResourceProvider(provider func(string) []byte) *Config {
	c.ResourceProvider = provider
	return c
}

type Target struct {
	IP       string            `json:"ip"`
	Port     string            `json:"port"`
	Service  string            `json:"service"`
	Scheme   string            `json:"scheme,omitempty"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Param    map[string]string `json:"param,omitempty"`
}

func (t Target) Address() string {
	if t.Port == "" {
		return t.IP
	}
	return t.IP + ":" + t.Port
}

type WeakpassTask struct {
	Targets   []Target
	Users     []string
	Passwords []string
	Auths     []Auth
}

type Auth struct {
	Username string
	Password string
}

func NewWeakpassTask(targets []Target) *WeakpassTask {
	return &WeakpassTask{Targets: targets}
}

func (t *WeakpassTask) Type() string {
	return "weakpass"
}

func (t *WeakpassTask) Validate() error {
	if len(t.Targets) == 0 {
		return fmt.Errorf("targets cannot be empty")
	}
	for i, target := range t.Targets {
		if target.IP == "" {
			return fmt.Errorf("targets[%d].IP cannot be empty", i)
		}
		if target.Service == "" {
			return fmt.Errorf("targets[%d].Service cannot be empty", i)
		}
	}
	return nil
}

type Result struct {
	success bool
	err     error
	data    *parsers.ZombieResult
}

func (r *Result) Success() bool {
	return r.success
}

func (r *Result) Error() error {
	return r.err
}

func (r *Result) Data() interface{} {
	return r.data
}

func (r *Result) ZombieResult() *parsers.ZombieResult {
	return r.data
}
