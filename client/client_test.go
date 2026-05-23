package client

import "testing"

func TestNewWithoutOptions(t *testing.T) {
	c := New()
	defer c.Close()

	if c.opts.provider != nil {
		t.Fatal("expected nil provider with no options")
	}
}

func TestClientEnginesAreTypeSafeAndCached(t *testing.T) {
	c := New()
	defer c.Close()

	fingers1, err := c.Fingers()
	if err != nil {
		t.Fatalf("get fingers engine: %v", err)
	}
	fingers2, err := c.Fingers()
	if err != nil {
		t.Fatalf("get fingers engine again: %v", err)
	}
	if fingers1 != fingers2 {
		t.Fatal("expected Fingers() to return cached engine")
	}

	gogoEngine, err := c.Gogo()
	if err != nil {
		t.Fatalf("get gogo engine: %v", err)
	}
	if gogoEngine.Name() != "gogo" {
		t.Fatalf("unexpected gogo engine name: %s", gogoEngine.Name())
	}

	sprayEngine, err := c.Spray()
	if err != nil {
		t.Fatalf("get spray engine: %v", err)
	}
	if sprayEngine.Name() != "spray" {
		t.Fatalf("unexpected spray engine name: %s", sprayEngine.Name())
	}

	neutronEngine, err := c.Neutron()
	if err != nil {
		t.Fatalf("get neutron engine: %v", err)
	}
	if neutronEngine.Name() != "neutron" {
		t.Fatalf("unexpected neutron engine name: %s", neutronEngine.Name())
	}

	zombieEngine, err := c.Zombie()
	if err != nil {
		t.Fatalf("get zombie engine: %v", err)
	}
	if zombieEngine.Name() != "zombie" {
		t.Fatalf("unexpected zombie engine name: %s", zombieEngine.Name())
	}
}

func TestGogoResolvesDependencies(t *testing.T) {
	c := New()
	defer c.Close()

	if c.fingers != nil || c.neutron != nil {
		t.Fatal("engines should be nil before first access")
	}

	_, err := c.Gogo()
	if err != nil {
		t.Fatalf("get gogo engine: %v", err)
	}

	c.mu.Lock()
	hasFinger := c.fingers != nil
	hasNeutron := c.neutron != nil
	c.mu.Unlock()

	if !hasFinger {
		t.Fatal("expected fingers engine to be created as gogo dependency")
	}
	if !hasNeutron {
		t.Fatal("expected neutron engine to be created as gogo dependency")
	}
}

func TestSprayResolvesFingersDependency(t *testing.T) {
	c := New()
	defer c.Close()

	_, err := c.Spray()
	if err != nil {
		t.Fatalf("get spray engine: %v", err)
	}

	c.mu.Lock()
	hasFinger := c.fingers != nil
	c.mu.Unlock()

	if !hasFinger {
		t.Fatal("expected fingers engine to be created as spray dependency")
	}
}

func TestCloseResetsEngines(t *testing.T) {
	c := New()

	_, _ = c.Fingers()
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	c.mu.Lock()
	isNil := c.fingers == nil
	c.mu.Unlock()

	if !isNil {
		t.Fatal("expected engines to be nil after Close")
	}
}

func TestIndexBeforeGogo(t *testing.T) {
	c := New()
	defer c.Close()

	if idx := c.Index(); idx != nil {
		t.Fatal("expected nil index before gogo is initialized")
	}
}
