package client

import "testing"

func TestClientZombie(t *testing.T) {
	c := New()
	defer c.Close()

	engine, err := c.Zombie()
	if err != nil {
		t.Fatalf("get zombie engine: %v", err)
	}
	if engine.Name() != "zombie" {
		t.Fatalf("unexpected zombie engine name: %s", engine.Name())
	}
}
