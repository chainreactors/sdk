package neutron

import (
	"context"
	"testing"

	"github.com/chainreactors/neutron/templates"
)

func TestConfigWithCapacity(t *testing.T) {
	config := NewConfig().WithCapacity(10)
	if config.Capacity != 10 {
		t.Fatalf("config.Capacity = %d, want 10", config.Capacity)
	}
}

func TestSetCapacityPostCreation(t *testing.T) {
	engine := &Engine{config: NewConfig()}
	if engine.Capacity() != nil {
		t.Fatal("engine should have no capacity by default")
	}
	engine.SetCapacity(5)
	if engine.Capacity() == nil {
		t.Fatal("engine should have a capacity after SetCapacity()")
	}
	if engine.Capacity().Total() != 5 {
		t.Fatalf("capacity total = %d, want 5", engine.Capacity().Total())
	}
}

func TestCapacityContextCancellation(t *testing.T) {
	dummyTemplate := &templates.Template{Id: "test-capacity"}
	engine := &Engine{
		config:    NewConfig(),
		templates: []*templates.Template{dummyTemplate},
	}
	engine.SetCapacity(1)

	// Exhaust capacity
	if err := engine.Capacity().Acquire(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	// Cancelled context should fail Acquire in executeTemplates
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	nctx := NewContext().WithContext(ctx)
	task := NewExecuteTask("http://127.0.0.1")

	_, err := engine.Execute(nctx, task)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}

	engine.Capacity().Release(1)
}

func TestNoCapacityByDefault(t *testing.T) {
	engine := &Engine{config: NewConfig()}
	if engine.Capacity() != nil {
		t.Fatal("engine should have no capacity by default")
	}
}
