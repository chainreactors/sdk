package cyberhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// apiEnvelope wraps payload in the standard {"code":0,"message":"ok","data":...} format.
func apiEnvelope(data interface{}) []byte {
	resp := map[string]interface{}{
		"code":    0,
		"message": "ok",
		"data":    data,
	}
	b, _ := json.Marshal(resp)
	return b
}

func TestScanFingerprints_StandardResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fingerprints/scan" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(apiEnvelope(map[string]interface{}{
			"id":     42,
			"slug":   "test-scan",
			"status": "completed",
			"data": map[string]interface{}{
				"fingerprint": map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{
							"fingerprint_id": 9770,
							"fingerprint": map[string]interface{}{
								"id":       9770,
								"name":     "wordpress-login-probe",
								"protocol": "http",
								"alias": map[string]interface{}{
									"id":       120,
									"name":     "WordPress",
									"vendor":   "Automattic",
									"product":  "wordpress",
									"category": "CMS",
								},
							},
							"framework": map[string]interface{}{
								"name": "WordPress",
								"attributes": map[string]interface{}{
									"version": "6.7.1",
									"cpe":     "cpe:2.3:a:automattic:wordpress:*:*:*:*:*:*:*:*",
								},
								"tags": []interface{}{"cms", "blog"},
							},
						},
					},
				},
			},
		}))
	}))
	defer server.Close()

	client := newClient(server.URL, "test-key", 5*time.Second)
	resp, err := client.ScanFingerprints(context.Background(), &FingerprintScanRequest{
		TargetURL: "https://example.test",
		RuleIDs:   []uint{9770},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ScanID != 42 {
		t.Fatalf("expected ScanID=42, got %d", resp.ScanID)
	}
	if resp.Slug != "test-scan" {
		t.Fatalf("expected slug=test-scan, got %q", resp.Slug)
	}
	if resp.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", resp.Status)
	}
	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	m := resp.Matches[0]
	if m.FingerprintID != 9770 {
		t.Errorf("FingerprintID: want 9770, got %d", m.FingerprintID)
	}
	if m.AliasID != 120 {
		t.Errorf("AliasID: want 120, got %d", m.AliasID)
	}
	if m.Name != "WordPress" {
		t.Errorf("Name: want WordPress, got %q", m.Name)
	}
	if m.Protocol != "http" {
		t.Errorf("Protocol: want http, got %q", m.Protocol)
	}
	if m.Version != "6.7.1" {
		t.Errorf("Version: want 6.7.1, got %q", m.Version)
	}
	if m.Vendor != "Automattic" {
		t.Errorf("Vendor: want Automattic, got %q", m.Vendor)
	}
	if m.Product != "wordpress" {
		t.Errorf("Product: want wordpress, got %q", m.Product)
	}
	if m.Category != "CMS" {
		t.Errorf("Category: want CMS, got %q", m.Category)
	}
	if m.CPE != "cpe:2.3:a:automattic:wordpress:*:*:*:*:*:*:*:*" {
		t.Errorf("CPE: want cpe:2.3:a:automattic:wordpress:*, got %q", m.CPE)
	}
	if len(m.Tags) != 2 || m.Tags[0] != "cms" || m.Tags[1] != "blog" {
		t.Errorf("Tags: want [cms blog], got %v", m.Tags)
	}
	if m.Fingerprint == nil {
		t.Error("Fingerprint summary should not be nil")
	}
	if m.Fingerprint != nil && m.Fingerprint.Alias == nil {
		t.Error("Fingerprint.Alias should not be nil")
	}
}

func TestScanFingerprints_AutoFetchActiveRuleIDs(t *testing.T) {
	var scanCalled bool
	var sentRuleIDs []interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/fingerprints" && r.Method == http.MethodGet:
			// Listing endpoint: return 2 active fingerprints
			w.Write(apiEnvelope(map[string]interface{}{
				"fingerprints": []interface{}{
					map[string]interface{}{"id": 12798},
					map[string]interface{}{"id": 12805},
				},
				"total":     2,
				"page":      1,
				"page_size": 1000,
			}))

		case r.URL.Path == "/api/v1/fingerprints/scan" && r.Method == http.MethodPost:
			scanCalled = true
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if ids, ok := body["rule_ids"].([]interface{}); ok {
				sentRuleIDs = ids
			}
			w.Write(apiEnvelope(map[string]interface{}{
				"data": map[string]interface{}{
					"fingerprint": map[string]interface{}{
						"results": []interface{}{
							map[string]interface{}{
								"fingerprint_id": 12798,
								"fingerprint": map[string]interface{}{
									"id":       12798,
									"name":     "nginx-keyword",
									"protocol": "http",
								},
								"framework": map[string]interface{}{
									"name": "nginx-keyword",
									"attributes": map[string]interface{}{
										"version": "1.29.3",
									},
									"tags": []interface{}{"fingers"},
								},
							},
						},
					},
				},
			}))

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newClient(server.URL, "test-key", 5*time.Second)
	resp, err := client.ScanFingerprints(context.Background(), &FingerprintScanRequest{
		TargetURL: "https://example.test",
		// No RuleIDs — should auto-fetch
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !scanCalled {
		t.Fatal("scan endpoint was not called")
	}

	// Verify the auto-fetched rule IDs were sent to scan
	if len(sentRuleIDs) != 2 {
		t.Fatalf("expected 2 rule_ids sent, got %d: %v", len(sentRuleIDs), sentRuleIDs)
	}
	id0 := uint(sentRuleIDs[0].(float64))
	id1 := uint(sentRuleIDs[1].(float64))
	if (id0 != 12798 || id1 != 12805) && (id0 != 12805 || id1 != 12798) {
		t.Errorf("unexpected rule_ids: %v", sentRuleIDs)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}
	if resp.Matches[0].FingerprintID != 12798 {
		t.Errorf("FingerprintID: want 12798, got %d", resp.Matches[0].FingerprintID)
	}
	if resp.Matches[0].Version != "1.29.3" {
		t.Errorf("Version: want 1.29.3, got %q", resp.Matches[0].Version)
	}
}

func TestScanFingerprints_RawContentSkipsListing(t *testing.T) {
	var listingCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/fingerprints" && r.Method == http.MethodGet:
			listingCalled = true
			w.Write(apiEnvelope(map[string]interface{}{
				"fingerprints": []interface{}{},
				"total":        0,
			}))

		case r.URL.Path == "/api/v1/fingerprints/scan" && r.Method == http.MethodPost:
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			// Verify raw_content was forwarded
			if body["raw_content"] == nil || body["raw_content"] == "" {
				t.Error("expected raw_content in scan payload")
			}
			// Verify engine was forwarded
			if body["engine"] != "fingers" {
				t.Errorf("expected engine=fingers, got %v", body["engine"])
			}

			w.Write(apiEnvelope(map[string]interface{}{
				"data": map[string]interface{}{
					"fingerprint": map[string]interface{}{
						"results": []interface{}{
							map[string]interface{}{
								"fingerprint_id": 0,
								"fingerprint":    map[string]interface{}{},
								"framework": map[string]interface{}{
									"name": "custom-app",
									"attributes": map[string]interface{}{
										"version": "2.0.0",
									},
								},
							},
						},
					},
				},
			}))

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newClient(server.URL, "test-key", 5*time.Second)
	resp, err := client.ScanFingerprints(context.Background(), &FingerprintScanRequest{
		TargetURL:  "https://example.test",
		RawContent: "name: custom-app\nrules:\n  - body: custom",
		Engine:     "fingers",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if listingCalled {
		t.Error("listing endpoint should NOT be called when raw_content is provided")
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}
	m := resp.Matches[0]
	if m.FingerprintID != 0 {
		t.Errorf("FingerprintID: want 0 (raw_content mode), got %d", m.FingerprintID)
	}
	if m.Name != "custom-app" {
		t.Errorf("Name: want custom-app, got %q", m.Name)
	}
	if m.Version != "2.0.0" {
		t.Errorf("Version: want 2.0.0, got %q", m.Version)
	}
}

func TestScanFingerprints_LegacyFrameworksFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(apiEnvelope(map[string]interface{}{
			"frameworks": map[string]interface{}{
				"Nginx": map[string]interface{}{
					"version":  "1.25.0",
					"vendor":   "F5",
					"product":  "nginx",
					"category": "Server",
					"tags":     []interface{}{"web"},
				},
			},
		}))
	}))
	defer server.Close()

	client := newClient(server.URL, "test-key", 5*time.Second)
	resp, err := client.ScanFingerprints(context.Background(), &FingerprintScanRequest{
		TargetURL: "https://example.test",
		RuleIDs:   []uint{1},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}
	m := resp.Matches[0]
	if m.FingerprintID != 0 {
		t.Errorf("FingerprintID: want 0 (legacy), got %d", m.FingerprintID)
	}
	if m.Name != "Nginx" {
		t.Errorf("Name: want Nginx, got %q", m.Name)
	}
	if m.Version != "1.25.0" {
		t.Errorf("Version: want 1.25.0, got %q", m.Version)
	}
	if m.Vendor != "F5" {
		t.Errorf("Vendor: want F5, got %q", m.Vendor)
	}
	if m.Product != "nginx" {
		t.Errorf("Product: want nginx, got %q", m.Product)
	}
	if m.Category != "Server" {
		t.Errorf("Category: want Server, got %q", m.Category)
	}
	if len(m.Tags) != 1 || m.Tags[0] != "web" {
		t.Errorf("Tags: want [web], got %v", m.Tags)
	}
}
