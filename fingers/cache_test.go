package fingers

import (
	"testing"
)

func TestCachingSenderOnlyCachesSuccessfulResponses(t *testing.T) {
	calls := 0
	sender := cachingSender(func(_ []byte) ([]byte, bool) {
		calls++
		if calls == 1 {
			return nil, false
		}
		return []byte("ok"), true
	})

	if _, ok := sender([]byte("/probe")); ok {
		t.Fatal("first failed response should not be reported as success")
	}
	if calls != 1 {
		t.Fatalf("calls after first attempt = %d, want 1", calls)
	}

	resp, ok := sender([]byte("/probe"))
	if !ok || string(resp) != "ok" {
		t.Fatalf("second response = %q, %v; want ok,true", resp, ok)
	}
	if calls != 2 {
		t.Fatalf("failed response was cached; calls = %d, want 2", calls)
	}

	resp, ok = sender([]byte("/probe"))
	if !ok || string(resp) != "ok" {
		t.Fatalf("cached response = %q, %v; want ok,true", resp, ok)
	}
	if calls != 2 {
		t.Fatalf("successful response was not cached; calls = %d, want 2", calls)
	}
}
