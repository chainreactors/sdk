package cyberhub

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chainreactors/sdk/pkg/types"
)

// FingerprintExport is one raw fingerprint export row returned by CyberHub.
// Finger is the parsed rule used by the scan engine; RawContent is the
// approved/effective YAML and RawContentDraft is the pending draft YAML when
// requested with ExportFilter.WithDraft(true).
type FingerprintExport struct {
	*types.Finger   `json:",inline" yaml:",inline"`
	Alias           *types.Alias `json:"alias,omitempty" yaml:"alias,omitempty"`
	Engine          string       `json:"engine,omitempty" yaml:"engine,omitempty"`
	Source          string       `json:"source,omitempty" yaml:"source,omitempty"`
	SourceNames     []string     `json:"source_names,omitempty" yaml:"source_names,omitempty"`
	RawContent      string       `json:"raw_content,omitempty" yaml:"raw_content,omitempty"`
	RawContentDraft string       `json:"raw_content_draft,omitempty" yaml:"raw_content_draft,omitempty"`
}

type fingerprintExportListResponse struct {
	Fingerprints []FingerprintExport `json:"fingerprints"`
	Total        int                 `json:"total"`
	Page         int                 `json:"page"`
	PageSize     int                 `json:"page_size"`
}

// ExportFingers exports full fingerprint records, including raw_content and
// raw_content_draft fields.
func (p *Provider) ExportFingers(ctx context.Context) ([]FingerprintExport, error) {
	params := url.Values{}
	params.Set("with_fingerprint", "true")
	applyFilterParams(params, p.filter)

	endpoint := fmt.Sprintf("%s/fingerprints/export?%s", p.client().baseURL, params.Encode())

	var response fingerprintExportListResponse
	if err := p.client().doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("export fingers failed: %w", err)
	}
	return response.Fingerprints, nil
}
