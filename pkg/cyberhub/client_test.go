package cyberhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"
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

func TestApplyFilterParams_Statuses(t *testing.T) {
	params := url.Values{}
	params.Add("statuses", "active")

	filter := &ExportFilter{
		Statuses: []string{"active", "pending", "draft", ""},
	}

	applyFilterParams(params, filter)

	got := params["statuses"]
	sort.Strings(got)
	want := []string{"active", "draft", "pending"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestApplyFilterParams_ReviewStatus(t *testing.T) {
	params := url.Values{}
	filter := &ExportFilter{
		ReviewStatus: " pending ",
	}

	applyFilterParams(params, filter)

	if got := params.Get("review_status"); got != "pending" {
		t.Fatalf("expected review_status=pending, got %q", got)
	}
}

func TestApplyFilterParams_Draft(t *testing.T) {
	t.Run("default false omits with_draft", func(t *testing.T) {
		params := url.Values{}
		applyFilterParams(params, &ExportFilter{})

		if _, ok := params["with_draft"]; ok {
			t.Fatalf("expected no with_draft param when Draft=false, got %v", params["with_draft"])
		}
	})

	t.Run("Draft=true transmits with_draft=true", func(t *testing.T) {
		params := url.Values{}
		applyFilterParams(params, &ExportFilter{Draft: true})

		if got := params.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
	})

	t.Run("Draft orthogonal to ReviewStatus/Statuses", func(t *testing.T) {
		params := url.Values{}
		filter := &ExportFilter{
			Statuses:     []string{"pending", "draft"},
			ReviewStatus: "pending",
			Draft:        true,
		}

		applyFilterParams(params, filter)

		if got := params.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
		if got := params.Get("review_status"); got != "pending" {
			t.Fatalf("expected review_status=pending, got %q", got)
		}
		statuses := params["statuses"]
		sort.Strings(statuses)
		if len(statuses) != 2 || statuses[0] != "draft" || statuses[1] != "pending" {
			t.Fatalf("expected statuses=[draft pending], got %v", statuses)
		}
	})
}

func TestApplyDefaultPOCStatus(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(url.Values)
		wantStatus string
		wantSkip   bool
	}{
		{
			name:       "no filter falls back to active",
			setup:      func(p url.Values) {},
			wantStatus: "active",
		},
		{
			name: "explicit single status preserved",
			setup: func(p url.Values) {
				p.Set("status", "pending")
			},
			wantStatus: "pending",
		},
		{
			name: "explicit multi statuses suppresses default",
			setup: func(p url.Values) {
				p.Add("statuses", "active")
				p.Add("statuses", "pending")
			},
			wantSkip: true,
		},
		{
			name: "review_status alone suppresses default",
			setup: func(p url.Values) {
				p.Set("review_status", "pending")
			},
			wantSkip: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := url.Values{}
			tc.setup(params)
			applyDefaultPOCStatus(params)

			if tc.wantSkip {
				if got := params.Get("status"); got != "" {
					t.Fatalf("expected status= unset, got %q", got)
				}
				return
			}
			if got := params.Get("status"); got != tc.wantStatus {
				t.Fatalf("expected status=%q, got %q", tc.wantStatus, got)
			}
		})
	}
}

// TestExportPOCs_StatusBehavior verifies the end-to-end request shape against a mock
// Cyberhub backend: default (active only), explicit Statuses (overrides default),
// and ReviewStatus (suppresses default).
func TestExportPOCs_StatusBehavior(t *testing.T) {
	var captured url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		resp := APIResponse{
			Code:    0,
			Message: "ok",
			Data: POCListResponse{
				POCs:     []POCResponse{},
				Total:    0,
				Exported: 0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	ctx := context.Background()

	t.Run("default exports active only", func(t *testing.T) {
		captured = nil
		if _, err := client.ExportPOCs(ctx, nil, nil, "", "", nil); err != nil {
			t.Fatalf("ExportPOCs failed: %v", err)
		}
		if got := captured.Get("status"); got != "active" {
			t.Fatalf("expected default status=active, got %q", got)
		}
		if got := captured["statuses"]; len(got) != 0 {
			t.Fatalf("expected no statuses= param, got %v", got)
		}
	})

	t.Run("explicit Statuses overrides default", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithStatuses("active", "pending", "draft")
		if _, err := client.ExportPOCs(ctx, nil, nil, "", "", filter); err != nil {
			t.Fatalf("ExportPOCs failed: %v", err)
		}
		if got := captured.Get("status"); got != "" {
			t.Fatalf("expected status= empty, got %q", got)
		}
		got := captured["statuses"]
		sort.Strings(got)
		want := []string{"active", "draft", "pending"}
		if len(got) != len(want) {
			t.Fatalf("expected statuses=%v, got %v", want, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected statuses=%v, got %v", want, got)
			}
		}
	})

	t.Run("ReviewStatus suppresses default active", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithReviewStatus("pending")
		if _, err := client.ExportPOCs(ctx, nil, nil, "", "", filter); err != nil {
			t.Fatalf("ExportPOCs failed: %v", err)
		}
		if got := captured.Get("status"); got != "" {
			t.Fatalf("expected status= empty (so review-pending POCs not filtered out), got %q", got)
		}
		if got := captured.Get("review_status"); got != "pending" {
			t.Fatalf("expected review_status=pending, got %q", got)
		}
	})

	t.Run("Draft alone wires with_draft=true and keeps default active", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithDraft(true)
		if _, err := client.ExportPOCs(ctx, nil, nil, "", "", filter); err != nil {
			t.Fatalf("ExportPOCs failed: %v", err)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
		// Draft alone does not touch the default active fallback —
		// caller still needs WithStatuses / WithReviewStatus to list pending rows.
		if got := captured.Get("status"); got != "active" {
			t.Fatalf("expected default status=active, got %q", got)
		}
	})

	t.Run("ReviewStatus pending + Draft true returns pending draft content", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithReviewStatus("pending").WithDraft(true)
		if _, err := client.ExportPOCs(ctx, nil, nil, "", "", filter); err != nil {
			t.Fatalf("ExportPOCs failed: %v", err)
		}
		if got := captured.Get("review_status"); got != "pending" {
			t.Fatalf("expected review_status=pending, got %q", got)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
		if got := captured.Get("status"); got != "" {
			t.Fatalf("expected status= empty (review_status suppresses default), got %q", got)
		}
	})

	t.Run("default omits with_draft", func(t *testing.T) {
		captured = nil
		if _, err := client.ExportPOCs(ctx, nil, nil, "", "", nil); err != nil {
			t.Fatalf("ExportPOCs failed: %v", err)
		}
		if _, ok := captured["with_draft"]; ok {
			t.Fatalf("expected no with_draft param by default, got %v", captured["with_draft"])
		}
	})

	t.Run("names default exports active only", func(t *testing.T) {
		captured = nil
		if _, err := client.ExportPOCsByNames(ctx, []string{"example-poc"}); err != nil {
			t.Fatalf("ExportPOCsByNames failed: %v", err)
		}
		if got := captured.Get("status"); got != "active" {
			t.Fatalf("expected default status=active, got %q", got)
		}
		if got := captured["statuses"]; len(got) != 0 {
			t.Fatalf("expected no statuses= param, got %v", got)
		}
		if got := captured["names"]; len(got) != 1 || got[0] != "example-poc" {
			t.Fatalf("expected names=[example-poc], got %v", got)
		}
	})

	t.Run("names explicit Statuses overrides default", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithStatuses("pending")
		if _, err := client.ExportPOCsByNamesWithFilter(ctx, []string{"pending-poc"}, filter); err != nil {
			t.Fatalf("ExportPOCsByNamesWithFilter failed: %v", err)
		}
		if got := captured.Get("status"); got != "" {
			t.Fatalf("expected status= empty, got %q", got)
		}
		if got := captured["statuses"]; len(got) != 1 || got[0] != "pending" {
			t.Fatalf("expected statuses=[pending], got %v", got)
		}
	})
}

// TestExportFingerprints_DraftBehavior verifies WithDraft also wires the
// fingerprint export path (no default-active fallback applies on this endpoint).
func TestExportFingerprints_DraftBehavior(t *testing.T) {
	var captured url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		resp := APIResponse{
			Code:    0,
			Message: "ok",
			Data: FingerprintListResponse{
				Fingerprints: []FingerprintResponse{},
				Total:        0,
				Page:         1,
				PageSize:     0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	ctx := context.Background()

	t.Run("default omits with_draft", func(t *testing.T) {
		captured = nil
		if _, err := client.ExportFingerprints(ctx, true, "", nil); err != nil {
			t.Fatalf("ExportFingerprints failed: %v", err)
		}
		if _, ok := captured["with_draft"]; ok {
			t.Fatalf("expected no with_draft param by default, got %v", captured["with_draft"])
		}
	})

	t.Run("WithDraft(true) sets with_draft=true", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithDraft(true)
		if _, err := client.ExportFingerprints(ctx, true, "", filter); err != nil {
			t.Fatalf("ExportFingerprints failed: %v", err)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
	})

	t.Run("ReviewStatus pending + Draft true wires both", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithReviewStatus("pending").WithDraft(true)
		if _, err := client.ExportFingerprints(ctx, true, "", filter); err != nil {
			t.Fatalf("ExportFingerprints failed: %v", err)
		}
		if got := captured.Get("review_status"); got != "pending" {
			t.Fatalf("expected review_status=pending, got %q", got)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
	})
}
