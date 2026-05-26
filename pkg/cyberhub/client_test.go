package cyberhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/chainreactors/sdk/pkg/types"
)

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

	t.Run("Draft orthogonal to Statuses/ReviewStatus", func(t *testing.T) {
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
		if got := params["statuses"]; len(got) != 2 {
			t.Fatalf("expected 2 statuses, got %v", got)
		}
	})

	t.Run("nil filter does not panic and omits with_draft", func(t *testing.T) {
		params := url.Values{}
		applyFilterParams(params, nil)

		if _, ok := params["with_draft"]; ok {
			t.Fatalf("expected no with_draft param for nil filter, got %v", params["with_draft"])
		}
	})
}

func TestWithDraftBuilder(t *testing.T) {
	filter := NewExportFilter().WithDraft(true)
	if !filter.Draft {
		t.Fatalf("expected Draft=true after WithDraft(true)")
	}
	if got := filter.WithDraft(false); got.Draft {
		t.Fatalf("expected WithDraft(false) to clear the flag")
	}
}

func TestProviderExportFingersReturnsRawContentFields(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		data := fingerprintExportListResponse{
			Fingerprints: []FingerprintExport{
				{
					Finger:          &types.Finger{Name: "pending-hub", Protocol: "http"},
					Engine:          "fingerprinthub",
					Source:          "unit-source",
					SourceNames:     []string{"unit-source"},
					RawContent:      "approved-yaml",
					RawContentDraft: "pending-yaml",
				},
			},
		}
		resp := apiResponse{Code: 0, Message: "ok", Data: data}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	filter := NewExportFilter().WithReviewStatus("pending").WithDraft(true)
	p := NewProvider(server.URL, "test-key").WithFilter(filter).WithTimeout(5 * time.Second)
	records, err := p.ExportFingers(context.Background())
	if err != nil {
		t.Fatalf("ExportFingers failed: %v", err)
	}

	if got := captured.Get("with_draft"); got != "true" {
		t.Fatalf("expected with_draft=true, got %q", got)
	}
	if got := captured.Get("review_status"); got != "pending" {
		t.Fatalf("expected review_status=pending, got %q", got)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	record := records[0]
	if record.Finger == nil || record.Finger.Name != "pending-hub" {
		t.Fatalf("expected embedded finger pending-hub, got %#v", record.Finger)
	}
	if record.Engine != "fingerprinthub" {
		t.Fatalf("expected engine fingerprinthub, got %q", record.Engine)
	}
	if record.RawContent != "approved-yaml" {
		t.Fatalf("expected approved raw_content, got %q", record.RawContent)
	}
	if record.RawContentDraft != "pending-yaml" {
		t.Fatalf("expected pending raw_content_draft, got %q", record.RawContentDraft)
	}
	if got := record.EffectiveRawContent(); got != "pending-yaml" {
		t.Fatalf("expected effective raw content to prefer draft, got %q", got)
	}
	if got := (FingerprintExport{RawContent: "approved-yaml"}).EffectiveRawContent(); got != "approved-yaml" {
		t.Fatalf("expected effective raw content to fall back to approved, got %q", got)
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

// TestProvider_DraftBehavior verifies that ExportFilter.Draft reaches the
// server as ?with_draft=true via both Provider.Fingers and Provider.POCs.
func TestProvider_DraftBehavior(t *testing.T) {
	var captured url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query()
		var data interface{}
		switch {
		case r.URL.Path == "/api/v1/fingerprints/export":
			data = fingerprintListResponse{}
		case r.URL.Path == "/api/v1/pocs/export":
			data = pocListResponse{}
		}
		resp := apiResponse{Code: 0, Message: "ok", Data: data}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ctx := context.Background()

	t.Run("fingers without WithDraft omits with_draft", func(t *testing.T) {
		captured = nil
		p := NewProvider(server.URL, "test-key").WithTimeout(5 * time.Second)
		if _, _, err := p.Fingers(ctx); err != nil {
			t.Fatalf("Fingers failed: %v", err)
		}
		if _, ok := captured["with_draft"]; ok {
			t.Fatalf("expected no with_draft param by default, got %v", captured["with_draft"])
		}
	})

	t.Run("fingers with WithDraft(true) sends with_draft=true", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithDraft(true).WithReviewStatus("pending")
		p := NewProvider(server.URL, "test-key").WithFilter(filter).WithTimeout(5 * time.Second)
		if _, _, err := p.Fingers(ctx); err != nil {
			t.Fatalf("Fingers failed: %v", err)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
		if got := captured.Get("review_status"); got != "pending" {
			t.Fatalf("expected review_status=pending, got %q", got)
		}
	})

	t.Run("pocs without WithDraft omits with_draft and keeps default active", func(t *testing.T) {
		captured = nil
		p := NewProvider(server.URL, "test-key").WithTimeout(5 * time.Second)
		if _, err := p.POCs(ctx); err != nil {
			t.Fatalf("POCs failed: %v", err)
		}
		if _, ok := captured["with_draft"]; ok {
			t.Fatalf("expected no with_draft param by default, got %v", captured["with_draft"])
		}
		if got := captured.Get("status"); got != "active" {
			t.Fatalf("expected default status=active, got %q", got)
		}
	})

	t.Run("pocs with WithDraft(true) alone keeps default active but adds with_draft=true", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithDraft(true)
		p := NewProvider(server.URL, "test-key").WithFilter(filter).WithTimeout(5 * time.Second)
		if _, err := p.POCs(ctx); err != nil {
			t.Fatalf("POCs failed: %v", err)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
		// Draft alone does not change the row-filter; caller must add
		// WithStatuses / WithReviewStatus to pull pending rows.
		if got := captured.Get("status"); got != "active" {
			t.Fatalf("expected default status=active when Draft=true without status filter, got %q", got)
		}
	})

	t.Run("pocs with WithReviewStatus + WithDraft pulls pending drafts", func(t *testing.T) {
		captured = nil
		filter := NewExportFilter().WithReviewStatus("pending").WithDraft(true)
		p := NewProvider(server.URL, "test-key").WithFilter(filter).WithTimeout(5 * time.Second)
		if _, err := p.POCs(ctx); err != nil {
			t.Fatalf("POCs failed: %v", err)
		}
		if got := captured.Get("with_draft"); got != "true" {
			t.Fatalf("expected with_draft=true, got %q", got)
		}
		if got := captured.Get("review_status"); got != "pending" {
			t.Fatalf("expected review_status=pending, got %q", got)
		}
		// review_status suppresses the default active.
		if got := captured.Get("status"); got != "" {
			t.Fatalf("expected status= empty, got %q", got)
		}
	})
}
