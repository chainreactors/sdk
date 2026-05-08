package zombie

import (
	"context"
	"testing"
)

func TestEngineInitAndValidate(t *testing.T) {
	engine := NewEngine(NewConfig())
	if err := engine.Init(); err != nil {
		t.Fatalf("init zombie engine: %v", err)
	}
	if engine.Name() != "zombie" {
		t.Fatalf("unexpected engine name: %s", engine.Name())
	}

	_, err := engine.Execute(NewContext().WithContext(context.Background()), NewWeakpassTask(nil))
	if err == nil {
		t.Fatal("expected empty weakpass task to fail validation")
	}
}

func TestConfigWithCapacity(t *testing.T) {
	engine := NewEngine(NewConfig().WithCapacity(200))
	if engine.Capacity() == nil {
		t.Fatal("engine should have a capacity after WithCapacity()")
	}
	if engine.Capacity().Total() != 200 {
		t.Fatalf("capacity total = %d, want 200", engine.Capacity().Total())
	}
}

func TestSetCapacityPostCreation(t *testing.T) {
	engine := NewEngine(nil)
	if engine.Capacity() != nil {
		t.Fatal("engine should have no capacity by default")
	}
	engine.SetCapacity(300)
	if engine.Capacity() == nil {
		t.Fatal("engine should have a capacity after SetCapacity()")
	}
	if engine.Capacity().Total() != 300 {
		t.Fatalf("capacity total = %d, want 300", engine.Capacity().Total())
	}
}

func TestCapacityContextCancellation(t *testing.T) {
	engine := NewEngine(NewConfig().WithCapacity(10))
	if err := engine.Init(); err != nil {
		t.Fatalf("init zombie engine: %v", err)
	}

	// Exhaust capacity
	if err := engine.Capacity().Acquire(context.Background(), 10); err != nil {
		t.Fatal(err)
	}

	// Cancelled context should fail Acquire
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	zctx := NewContext().SetThreads(5).WithContext(ctx)

	_, err := engine.Execute(zctx, NewWeakpassTask([]Target{{IP: "127.0.0.1", Port: "22", Service: "ssh"}}))
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}

	engine.Capacity().Release(10)
}

func TestNoCapacityByDefault(t *testing.T) {
	engine := NewEngine(nil)
	if engine.Capacity() != nil {
		t.Fatal("engine should have no capacity by default")
	}
}
