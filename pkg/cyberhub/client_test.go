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

func TestApplyFilterParams_Names(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{
		Names: []string{"poc-1", "poc-2", "poc-1"},
	}
	applyFilterParams(params, filter)

	got := params["names"]
	if len(got) != 2 {
		t.Fatalf("expected 2 names, got %d: %v", len(got), got)
	}
}

func TestApplyFilterParams_Severities(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{
		Severities: []string{"critical", "high"},
	}
	applyFilterParams(params, filter)

	got := params["severities"]
	if len(got) != 2 {
		t.Fatalf("expected 2 severities, got %d: %v", len(got), got)
	}
}

func TestApplyFilterParams_POCType(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{POCType: "nuclei"}
	applyFilterParams(params, filter)

	if params.Get("type") != "nuclei" {
		t.Fatalf("expected type=nuclei, got %q", params.Get("type"))
	}
}

func TestApplyFilterParams_Statuses(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{
		Statuses: []string{"active", "pending"},
	}
	applyFilterParams(params, filter)

	got := params["statuses"]
	if len(got) != 2 {
		t.Fatalf("expected 2 statuses, got %d: %v", len(got), got)
	}
}

func TestApplyFilterParams_ReviewStatus(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{ReviewStatus: "approved"}
	applyFilterParams(params, filter)

	if params.Get("review_status") != "approved" {
		t.Fatalf("expected review_status=approved, got %q", params.Get("review_status"))
	}
}

func TestApplyFilterParams_Limit(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{Limit: 10}
	applyFilterParams(params, filter)

	if params.Get("page") != "1" {
		t.Fatalf("expected page=1, got %q", params.Get("page"))
	}
	if params.Get("page_size") != "10" {
		t.Fatalf("expected page_size=10, got %q", params.Get("page_size"))
	}
}

func TestApplyDefaultPOCStatus_NoStatuses(t *testing.T) {
	params := url.Values{}
	applyDefaultPOCStatus(params)

	if params.Get("status") != "active" {
		t.Fatalf("expected default status=active, got %q", params.Get("status"))
	}
}

func TestApplyDefaultPOCStatus_WithStatuses(t *testing.T) {
	params := url.Values{}
	params.Add("statuses", "pending")
	applyDefaultPOCStatus(params)

	if params.Get("status") != "" {
		t.Fatalf("expected no default status when statuses set, got %q", params.Get("status"))
	}
}

func TestApplyDefaultPOCStatus_WithReviewStatus(t *testing.T) {
	params := url.Values{}
	params.Set("review_status", "approved")
	applyDefaultPOCStatus(params)

	if params.Get("status") != "" {
		t.Fatalf("expected no default status when review_status set, got %q", params.Get("status"))
	}
}
