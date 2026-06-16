package association_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/sdk/pkg/association"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

var (
	remoteIdx     *association.Index
	remoteIdxOnce sync.Once
	remoteIdxErr  error
)

func buildRemoteIndex(t *testing.T) *association.Index {
	t.Helper()
	url := os.Getenv("CYBERHUB_URL")
	key := os.Getenv("CYBERHUB_KEY")
	if url == "" || key == "" {
		t.Skip("CYBERHUB_URL / CYBERHUB_KEY not set, skipping integration test")
	}

	remoteIdxOnce.Do(func() {
		hub := cyberhub.NewProvider(url, key).WithTimeout(60 * time.Second)
		for attempt := 0; attempt < 3; attempt++ {
			remoteIdx, remoteIdxErr = association.BuildFromProvider(context.Background(), hub)
			if remoteIdxErr == nil {
				return
			}
			time.Sleep(2 * time.Second)
		}
	})
	if remoteIdxErr != nil {
		t.Fatalf("BuildFromProvider after retries: %v", remoteIdxErr)
	}
	return remoteIdx
}

func TestIntegration_IndexNotEmpty(t *testing.T) {
	idx := buildRemoteIndex(t)

	r := idx.Lookup(association.NewQuery())
	if len(r.Fingers) == 0 {
		t.Fatal("expected fingers from remote index")
	}
	if len(r.Aliases) == 0 {
		t.Fatal("expected aliases from remote index")
	}
	t.Logf("index loaded: fingers=%d aliases=%d templates=%d",
		len(r.Fingers), len(r.Aliases), len(r.Templates))
}

func TestIntegration_SearchSubstring(t *testing.T) {
	idx := buildRemoteIndex(t)

	tests := []struct {
		search string
		desc   string
	}{
		{"splunk", "common product name"},
		{"nginx", "well-known web server"},
		{"mysql", "database keyword"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			r := idx.Lookup(association.NewQuery().WithSearch(tc.search))
			if len(r.Fingers)+len(r.Aliases)+len(r.Templates) == 0 {
				t.Fatalf("search %q returned no results", tc.search)
			}
			t.Logf("search %q: fingers=%d aliases=%d templates=%d",
				tc.search, len(r.Fingers), len(r.Aliases), len(r.Templates))
		})
	}
}

func TestIntegration_SearchVsExactQuery(t *testing.T) {
	idx := buildRemoteIndex(t)

	exact := idx.Lookup(association.NewQuery().WithFingers("a:splunk:enterprise_security"))
	fuzzy := idx.Lookup(association.NewQuery().WithSearch("splunk"))

	if len(exact.Fingers) == 0 {
		t.Skip("exact finger 'a:splunk:enterprise_security' not found, skipping comparison")
	}

	if len(fuzzy.Fingers) < len(exact.Fingers) {
		t.Fatalf("fuzzy search should return >= exact results: fuzzy=%d exact=%d",
			len(fuzzy.Fingers), len(exact.Fingers))
	}

	t.Logf("exact fingers=%d, fuzzy fingers=%d (search expands results as expected)",
		len(exact.Fingers), len(fuzzy.Fingers))
}

func TestIntegration_SearchCaseInsensitive(t *testing.T) {
	idx := buildRemoteIndex(t)

	lower := idx.Lookup(association.NewQuery().WithSearch("nginx"))
	upper := idx.Lookup(association.NewQuery().WithSearch("NGINX"))

	if len(lower.Fingers) != len(upper.Fingers) {
		t.Fatalf("case sensitivity mismatch: lower=%d upper=%d",
			len(lower.Fingers), len(upper.Fingers))
	}
}

func TestIntegration_SearchNoMatch(t *testing.T) {
	idx := buildRemoteIndex(t)

	r := idx.Lookup(association.NewQuery().WithSearch("zzz_surely_nonexistent_xyzzy"))
	if len(r.Fingers)+len(r.Aliases)+len(r.Templates) != 0 {
		t.Fatalf("expected empty result for nonsense search, got fingers=%d aliases=%d templates=%d",
			len(r.Fingers), len(r.Aliases), len(r.Templates))
	}
}

func TestIntegration_SearchCombinedWithTerms(t *testing.T) {
	idx := buildRemoteIndex(t)

	searchOnly := idx.Lookup(association.NewQuery().WithSearch("splunk"))
	combined := idx.Lookup(association.NewQuery().WithSearch("splunk").WithTags("ai_converted"))

	if len(combined.Fingers)+len(combined.Aliases) < len(searchOnly.Fingers)+len(searchOnly.Aliases) {
		t.Log("combined query may narrow or expand depending on data, just checking it doesn't crash")
	}
	t.Logf("search-only: fingers=%d aliases=%d; combined: fingers=%d aliases=%d",
		len(searchOnly.Fingers), len(searchOnly.Aliases),
		len(combined.Fingers), len(combined.Aliases))
}

func TestIntegration_FingersWithTemplates(t *testing.T) {
	idx := buildRemoteIndex(t)

	r := idx.Lookup(association.NewQuery())
	fwt := r.FingersWithTemplates(idx)
	t.Logf("fingers with templates: %d out of %d total fingers", len(fwt), len(r.Fingers))

	for _, entry := range fwt {
		if entry.Finger == nil {
			t.Fatal("nil finger in FingersWithTemplates result")
		}
		if entry.TemplateCount <= 0 {
			t.Fatalf("finger %q has non-positive template count %d",
				entry.Finger.Name, entry.TemplateCount)
		}
	}
}
