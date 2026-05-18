package client

import "testing"

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

func TestNewEngineRejectsUnknownName(t *testing.T) {
	if _, err := newEngine("missing", nil); err == nil {
		t.Fatal("expected unknown engine error")
	}
}
