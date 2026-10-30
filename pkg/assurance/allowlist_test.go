package assurance

import (
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func TestAllowlistCheckURL(t *testing.T) {
	t.Parallel()
	prod := NewAllowlist(false, "assurance.agenttwin.example", "eu.agenttwin.example:8443")
	dev := NewAllowlist(true, "assurance.agenttwin.example")

	tests := []struct {
		name    string
		allow   *Allowlist
		raw     string
		wantErr bool
	}{
		{"https allowlisted host", prod, "https://assurance.agenttwin.example/api/v1/eval-runs", false},
		{"https allowlisted host:port", prod, "https://eu.agenttwin.example:8443/api/v1/eval-runs", false},
		{"host not in allowlist", prod, "https://evil.example/api", true},
		{"http to non-loopback rejected", prod, "http://assurance.agenttwin.example/api", true},
		{"loopback rejected when not allowed", prod, "http://localhost:8080/api", true},
		{"loopback allowed in dev", dev, "http://localhost:8080/api/v1/eval-runs", false},
		{"loopback ip allowed in dev", dev, "http://127.0.0.1:8080/api", false},
		{"ftp scheme rejected", prod, "ftp://assurance.agenttwin.example/x", true},
		{"embedded credentials rejected", prod, "https://user:pass@assurance.agenttwin.example/api", true},
		{"missing host rejected", prod, "https:///api", true},
		{"garbage rejected", prod, "://not a url", true},
		{"wrong port not allowlisted", prod, "https://eu.agenttwin.example:9999/api", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.allow.CheckURL(tc.raw)
			if tc.wantErr && err == nil {
				t.Fatalf("CheckURL(%q) = nil, want error", tc.raw)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("CheckURL(%q) = %v, want nil", tc.raw, err)
			}
			if tc.wantErr && errs.KindOf(err) != errs.KindValidation {
				t.Fatalf("CheckURL(%q) error kind = %v, want validation", tc.raw, errs.KindOf(err))
			}
		})
	}
}

func TestAllowlistBareHostMatchesAnyPort(t *testing.T) {
	t.Parallel()
	a := NewAllowlist(false, "assurance.agenttwin.example")
	for _, u := range []string{
		"https://assurance.agenttwin.example/api",
		"https://assurance.agenttwin.example:8443/api",
	} {
		if err := a.CheckURL(u); err != nil {
			t.Errorf("bare-host allowlist should accept %q: %v", u, err)
		}
	}
}

func TestMappingValidateRejectsNilAllowlist(t *testing.T) {
	t.Parallel()
	m := MappingRecordV1{SchemaVersion: MappingSchemaVersion}
	if err := m.Validate(nil); err == nil {
		t.Fatalf("Validate(nil allowlist) should error")
	}
}
