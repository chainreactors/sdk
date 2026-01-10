package cyberhub

import (
	"net/url"
	"testing"
)

func TestFirstFilter(t *testing.T) {
	filter := &ExportFilter{}
	got := firstFilter([]*ExportFilter{nil, filter})
	if got != filter {
		t.Fatalf("expected first non-nil filter")
	}
}

func TestApplyFilterParams_DedupTags(t *testing.T) {
	params := url.Values{}
	params.Add("tags", "foo")

	filter := &ExportFilter{
		Tags: []string{"foo", "bar", ""},
	}

	applyFilterParams(params, filter)

	got := params["tags"]
	if len(got) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(got), got)
	}

	seen := map[string]int{}
	for _, tag := range got {
		seen[tag]++
	}
	if seen["foo"] != 1 || seen["bar"] != 1 {
		t.Fatalf("unexpected tags after dedup: %v", got)
	}
}

func TestApplyFilterParams_DedupSources(t *testing.T) {
	params := url.Values{}
	params.Add("sources", "alpha")
	filter := &ExportFilter{
		Sources: []string{"alpha", "beta", ""},
	}

	applyFilterParams(params, filter)

	got := params["sources"]
	if len(got) != 2 {
		t.Fatalf("expected 2 sources, got %d: %v", len(got), got)
	}

	seen := map[string]int{}
	for _, source := range got {
		seen[source]++
	}
	if seen["alpha"] != 1 || seen["beta"] != 1 {
		t.Fatalf("unexpected sources after dedup: %v", got)
	}
}

func TestApplyFilterParams_LimitOnly(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{
		Limit: 10,
	}

	applyFilterParams(params, filter)

	if params.Get("page") != "1" {
		t.Fatalf("expected page=1, got %q", params.Get("page"))
	}
	if params.Get("page_size") != "10" {
		t.Fatalf("expected page_size=10, got %q", params.Get("page_size"))
	}
}
