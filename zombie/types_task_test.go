package zombie

import (
	"context"
	"errors"
	"testing"

	"github.com/chainreactors/parsers"
	zombiecore "github.com/chainreactors/zombie/core"
	zombiepkg "github.com/chainreactors/zombie/pkg"
)

func TestContextSettersCloneAndClamp(t *testing.T) {
	ctx := NewContext().
		SetThreads(12).
		SetTimeout(7).
		SetTop(3).
		SetFirstOnly(false).
		SetNoUnauth(true)

	ctx.SetThreads(0).SetTimeout(0).SetTop(-1)
	if ctx.threads != 12 || ctx.timeout != 7 || ctx.top != 3 {
		t.Fatalf("invalid setter values should be ignored: %+v", ctx)
	}

	child := ctx.WithContext(context.Background())
	if child == ctx {
		t.Fatal("WithContext should clone the SDK context")
	}
	if child.threads != 12 || child.timeout != 7 || child.top != 3 || child.firstOnly || !child.noUnauth {
		t.Fatalf("clone lost fields: %+v", child)
	}
}

func TestTargetAddressAndWeakpassValidation(t *testing.T) {
	if got := (Target{IP: "127.0.0.1"}).Address(); got != "127.0.0.1" {
		t.Fatalf("address without port = %q", got)
	}
	if got := (Target{IP: "127.0.0.1", Port: "22"}).Address(); got != "127.0.0.1:22" {
		t.Fatalf("address with port = %q", got)
	}

	if err := NewWeakpassTask(nil).Validate(); err == nil {
		t.Fatal("expected empty targets to fail")
	}
	if err := NewWeakpassTask([]Target{{Service: "ssh"}}).Validate(); err == nil {
		t.Fatal("expected missing IP to fail")
	}
	if err := NewWeakpassTask([]Target{{IP: "127.0.0.1"}}).Validate(); err == nil {
		t.Fatal("expected missing service to fail")
	}
	if err := NewWeakpassTask([]Target{{IP: "127.0.0.1", Service: "ssh"}}).Validate(); err != nil {
		t.Fatalf("valid weakpass task failed: %v", err)
	}
}

func TestExpandTasksUsesExplicitAuthsAndNoUnauth(t *testing.T) {
	if err := zombiepkg.Load(); err != nil {
		t.Fatalf("load zombie resources: %v", err)
	}

	ctx := NewContext().SetNoUnauth(true).SetFirstOnly(false)
	task := &WeakpassTask{
		Targets: []Target{{IP: "127.0.0.1", Service: "ssh"}},
		Auths:   []Auth{{Username: "root", Password: "toor"}},
	}

	ztasks := expandTasks(ctx, task)
	if len(ztasks) != 1 {
		t.Fatalf("expanded tasks len = %d, want 1", len(ztasks))
	}
	got := ztasks[0].ZombieResult
	if got.IP != "127.0.0.1" || got.Service != "ssh" || got.Username != "root" || got.Password != "toor" {
		t.Fatalf("unexpected expanded zombie task: %+v", got)
	}
	if got.Mod != parsers.ZombieModBrute {
		t.Fatalf("expected brute task, got %v", got.Mod)
	}
}

func TestZombieExecutionErrorClassification(t *testing.T) {
	for _, err := range []error{
		nil,
		zombiepkg.ErrorWrongUserOrPwd,
		zombiepkg.NotImplUnauthorized,
		zombiecore.ErrNoUnauth,
	} {
		if isZombieExecutionError(err) {
			t.Fatalf("expected %v to be non-execution error", err)
		}
	}
	if !isZombieExecutionError(errors.New("network failed")) {
		t.Fatal("expected generic error to count as execution error")
	}
}
