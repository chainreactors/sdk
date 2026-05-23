package cyberhub

import "github.com/chainreactors/sdk/pkg/types"

type fingerprintResponse struct {
	*types.Finger `json:",inline" yaml:",inline"`
	Alias         *types.Alias `json:"alias,omitempty" yaml:"alias,omitempty"`
}

type fingerprintListResponse struct {
	Fingerprints []fingerprintResponse `json:"fingerprints"`
	Total        int                   `json:"total"`
	Page         int                   `json:"page"`
	PageSize     int                   `json:"page_size"`
}

type pocResponse struct {
	*types.Template `json:",inline" yaml:",inline"`
}

type pocListResponse struct {
	POCs     []pocResponse `json:"pocs"`
	Total    int           `json:"total"`
	Exported int           `json:"exported"`
}

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
