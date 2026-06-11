package spray

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
)

// TestEmitStats_PanicAfterContextCancel reproduces the w3 pipeline panic.
//
// Scenario: consumer sets OnStats callback → context times out → consumer
// shuts down (closes its channel) → SDK defer calls emitStats → callback
// sends to closed channel → panic: send on closed channel.
//
// This test MUST panic before the fix and pass after.
func TestEmitStats_PanicAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate pipeline's p.events channel
	events := make(chan struct{}, 1)

	c := &Context{
		ctx: ctx,
		statsHandler: func(stats types.Stats) {
			events <- struct{}{} // this panics when events is closed
		},
	}

	// Step 1: consumer abandons — context cancel + close channel
	cancel()
	close(events)

	// Step 2: SDK defer fires emitStats after consumer is gone
	panicked := true
	func() {
		defer func() {
			if r := recover(); r == nil {
				panicked = false
			}
		}()
		c.emitStats(types.Stats{Engine: "spray"})
	}()

	if panicked {
		t.Fatal("emitStats called statsHandler after context was cancelled, " +
			"causing send on closed channel — this is the bug being fixed")
	}
}

func TestEmitStats_NilContext(t *testing.T) {
	var c *Context
	c.emitStats(types.Stats{})
}

func TestEmitStats_NilHandler(t *testing.T) {
	c := &Context{ctx: context.Background()}
	c.emitStats(types.Stats{})
}

func TestEmitStats_ContextActive(t *testing.T) {
	var called atomic.Bool
	c := &Context{
		ctx: context.Background(),
		statsHandler: func(stats types.Stats) {
			called.Store(true)
		},
	}
	c.emitStats(types.Stats{Engine: "test"})
	if !called.Load() {
		t.Fatal("statsHandler should be called when context is active")
	}
}
