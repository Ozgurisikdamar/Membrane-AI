package assurance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestFixturesRoundTrip loads the checked-in contract fixtures, unmarshals them
// into the Go contract types and runs Validate. The positive fixtures must pass;
// the negative one (a PASS that contradicts a critical-failure status) must be
// rejected by the same fail-safe invariant the schema encodes.
func TestFixturesRoundTrip(t *testing.T) {
	t.Parallel()
	allow := NewAllowlist(true, "assurance.agenttwin.example") // dev: fixture uses localhost

	t.Run("mapping-demo", func(t *testing.T) {
		t.Parallel()
		var m MappingRecordV1
		readJSON(t, "contracts/fixtures/mapping-demo.json", &m)
		if err := m.Validate(allow); err != nil {
			t.Fatalf("mapping fixture failed validation: %v", err)
		}
	})

	for _, name := range []string{"evidence-1.3.0-block.json", "evidence-1.3.1-pass.json"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var e ReleaseEvidenceV1
			readJSON(t, filepath.Join("contracts/fixtures", name), &e)
			if err := e.Validate(); err != nil {
				t.Fatalf("evidence fixture %s failed validation: %v", name, err)
			}
		})
	}

	t.Run("negative-pass-contradiction", func(t *testing.T) {
		t.Parallel()
		var e ReleaseEvidenceV1
		readJSON(t, "contracts/fixtures/evidence-invalid-pass-contradiction.json", &e)
		if err := e.Validate(); err == nil {
			t.Fatal("negative fixture passed validation; fail-safe invariant not enforced")
		}
	})
}

// TestEvidenceFixtureMatchesReproducedRun pins the evidence fixture to the actual
// Phase-0 reproduced run so the contract stays grounded in real AgentTwin output.
func TestEvidenceFixtureMatchesReproducedRun(t *testing.T) {
	t.Parallel()
	var e ReleaseEvidenceV1
	readJSON(t, "contracts/fixtures/evidence-1.3.0-block.json", &e)
	if e.BaselineVersion != "1.2.4" || e.CandidateVersion != "1.3.0" {
		t.Fatalf("unexpected versions %s -> %s", e.BaselineVersion, e.CandidateVersion)
	}
	if e.EvaluationRunID != "01a18c71-faf5-70b5-a459-7052a0b23f37" {
		t.Fatalf("eval run id drift: %s", e.EvaluationRunID)
	}
	if e.OverallStatus != StatusCompletedWithCriticalFailure || e.GateDecision != DecisionBlock {
		t.Fatalf("expected critical-failure BLOCK, got %s / %s", e.OverallStatus, e.GateDecision)
	}
	if len(e.ScenarioSelection) != 9 {
		t.Fatalf("expected the 9-scenario suite, got %d", len(e.ScenarioSelection))
	}
}

// TestSchemaRequiredMatchesStruct guards against drift between the published JSON
// Schemas and the Go types: every field a schema lists as required must exist as
// a json tag on the corresponding struct.
func TestSchemaRequiredMatchesStruct(t *testing.T) {
	t.Parallel()
	cases := []struct {
		schema string
		typ    reflect.Type
		verKey string
		verVal string
	}{
		{"contracts/evidence.v1.schema.json", reflect.TypeOf(ReleaseEvidenceV1{}), "schema_version", EvidenceSchemaVersion},
		{"contracts/mapping.v1.schema.json", reflect.TypeOf(MappingRecordV1{}), "schema_version", MappingSchemaVersion},
	}
	for _, c := range cases {
		t.Run(c.schema, func(t *testing.T) {
			t.Parallel()
			var doc struct {
				Required   []string                   `json:"required"`
				Properties map[string]json.RawMessage `json:"properties"`
			}
			readJSON(t, c.schema, &doc)

			tags := jsonTags(c.typ)
			for _, req := range doc.Required {
				if _, ok := tags[req]; !ok {
					t.Errorf("schema requires %q but struct %s has no such json field", req, c.typ.Name())
				}
			}

			// The schema's schema_version const must equal the Go constant.
			raw, ok := doc.Properties[c.verKey]
			if !ok {
				t.Fatalf("schema has no %q property", c.verKey)
			}
			var constHolder struct {
				Const string `json:"const"`
			}
			if err := json.Unmarshal(raw, &constHolder); err != nil {
				t.Fatalf("parse %s const: %v", c.verKey, err)
			}
			if constHolder.Const != c.verVal {
				t.Errorf("schema %s const = %q, Go const = %q", c.verKey, constHolder.Const, c.verVal)
			}
		})
	}
}

// TestAgentTwinOpenAPISnapshotSurface guards the AgentTwin contract surface the
// Membrane adapter consumes. If an upstream refresh removes or renames one of
// these routes/fields, this fails and the drift must be reconciled (see
// contracts/agenttwin/PROVENANCE.md), not papered over.
func TestAgentTwinOpenAPISnapshotSurface(t *testing.T) {
	t.Parallel()
	cases := map[string][]string{
		"contracts/agenttwin/evaluation-service.openapi.yaml": {
			"/api/v1/eval-runs",
			"startEvalRun",
			"/api/v1/eval-runs/{eval_run_id}",
			"Idempotency-Key",
		},
		"contracts/agenttwin/simulation-service.openapi.yaml": {
			"/internal/v1/simulation-pairs",
			"startSimulationPair",
			"Idempotency-Key",
		},
	}
	for file, tokens := range cases {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read snapshot %s: %v", file, err)
			}
			content := string(b)
			for _, tok := range tokens {
				if !strings.Contains(content, tok) {
					t.Errorf("AgentTwin snapshot %s no longer contains %q (upstream contract drift?)", filepath.Base(file), tok)
				}
			}
		})
	}
}

// readJSON reads and unmarshals a checked-in contract file relative to the
// package directory (the test working directory).
func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}

// jsonTags returns the set of json field names declared on a struct type,
// stripping ",omitempty" and skipping "-" and untagged fields.
func jsonTags(t reflect.Type) map[string]struct{} {
	out := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := tag
		if comma := strings.IndexByte(tag, ','); comma >= 0 {
			name = tag[:comma]
		}
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}
