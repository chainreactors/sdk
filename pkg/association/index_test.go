package association

import (
	"sort"
	"testing"

	"github.com/chainreactors/neutron/templates"
)

func TestFingerPOCIndexBuildQueryAndClear(t *testing.T) {
	idx := NewFingerPOCIndex()
	idx.BuildFromTemplates([]*templates.Template{
		nil,
		{Id: "CVE-1", Fingers: []string{"nginx", "php"}},
		{Id: "CVE-2", Fingers: []string{"nginx"}},
	})

	if fingerCount, pocCount := idx.Count(); fingerCount != 2 || pocCount != 2 {
		t.Fatalf("count = (%d, %d), want (2, 2)", fingerCount, pocCount)
	}

	nginxPOCs := idx.GetPOCsByFinger("nginx")
	sort.Strings(nginxPOCs)
	if len(nginxPOCs) != 2 || nginxPOCs[0] != "CVE-1" || nginxPOCs[1] != "CVE-2" {
		t.Fatalf("unexpected nginx pocs: %v", nginxPOCs)
	}
	if !idx.HasAssociatedPOC("php") || idx.GetPOCCountByFinger("php") != 1 {
		t.Fatalf("expected php to have one poc")
	}
	if got := idx.GetFingersByPOC("CVE-1"); len(got) != 2 {
		t.Fatalf("expected two fingers for CVE-1, got %v", got)
	}

	hasPOC := idx.GetFingerHasPOCMap()
	hasPOC["nginx"] = false
	if !idx.HasAssociatedPOC("nginx") {
		t.Fatal("GetFingerHasPOCMap should return a copy")
	}

	idx.Clear()
	if fingerCount, pocCount := idx.Count(); fingerCount != 0 || pocCount != 0 {
		t.Fatalf("count after clear = (%d, %d), want (0, 0)", fingerCount, pocCount)
	}
}
