package neutron

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chainreactors/sdk/pkg/types"
	"gopkg.in/yaml.v3"
)

func startTLSServer(t *testing.T, handler http.Handler) (addr string, cleanup func()) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost", Organization: []string{"TestOrg"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	server.StartTLS()
	return server.Listener.Addr().String(), server.Close
}

func TestSSLTemplateCompileAndExecute(t *testing.T) {
	addr, cleanup := startTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer cleanup()

	tpl := parseTemplateForTest(t, `id: ssl-cert-check
info:
  name: SSL Certificate Check
  severity: info
ssl:
  - address: "`+addr+`"
    matchers:
      - type: word
        words:
          - "localhost"
        part: subject_cn
`)

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected 1 compiled SSL template, got %d", len(compiled))
	}

	result, err := compiled[0].Execute(addr, nil)
	if err != nil {
		t.Fatalf("ssl execute: %v", err)
	}
	if result == nil || !result.Matched {
		t.Fatal("expected SSL template to match subject_cn=localhost")
	}
}

func TestSSLTemplateExtractCertFields(t *testing.T) {
	addr, cleanup := startTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer cleanup()

	tpl := parseTemplateForTest(t, `id: ssl-extract-org
info:
  name: SSL Extract Org
  severity: info
ssl:
  - address: "`+addr+`"
    extractors:
      - type: regex
        part: subject_org
        regex:
          - "(.+)"
    matchers:
      - type: word
        words:
          - "TestOrg"
        part: subject_org
`)

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected 1 compiled template, got %d", len(compiled))
	}

	result, err := compiled[0].Execute(addr, nil)
	if err != nil {
		t.Fatalf("ssl execute: %v", err)
	}
	if result == nil || !result.Matched {
		t.Fatal("expected SSL template to match subject_org=TestOrg")
	}
}

func TestSSLTemplateRejectsUnsupportedOptions(t *testing.T) {
	raw := `id: ssl-unsupported
info:
  name: test
  severity: info
ssl:
  - address: "127.0.0.1:443"
    tls_version_enum: true
    matchers:
      - type: word
        words:
          - "test"
`
	var tpl types.Template
	if err := yaml.Unmarshal([]byte(raw), &tpl); err != nil {
		t.Fatalf("parse: %v", err)
	}

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{&tpl})
	if len(compiled) != 0 {
		t.Fatal("expected unsupported ssl option to fail compilation")
	}
}

func TestSSLTemplateTLSAlias(t *testing.T) {
	raw := `id: tls-alias
info:
  name: TLS Alias Test
  severity: info
tls:
  - address: "127.0.0.1:443"
    matchers:
      - type: word
        words:
          - "test"
`
	var tpl types.Template
	if err := yaml.Unmarshal([]byte(raw), &tpl); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(tpl.RequestsTLS) == 0 {
		t.Fatal("expected tls: field to populate RequestsTLS")
	}

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{&tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected tls alias template to compile, got %d", len(compiled))
	}
}
