package manifest

import (
	"strings"
	"testing"
)

// validFixture returns a minimal, fully valid manifest. Each test mutates a
// single field to assert one validation rule.
func validFixture() *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		Bundle:        "my-tool",
		Version:       "1.0.0",
		Author:        "Acme <ops@acme.com>",
		Description:   "Sends Slack messages",
		License:       "Apache-2.0",
		Source: Source{
			Language:  "go-tinygo",
			Toolchain: "tinygo-0.41.1",
		},
		Runtime: Runtime{
			Kind:   "wasm-per-call",
			Quotas: Quotas{TimeoutS: 30, MemoryMB: 128, CPUMs: 5000},
		},
		Tools: []Tool{{
			ID:          "slack_post",
			Description: "Post message to Slack channel",
			InputSchema: map[string]any{"type": "object"},
			CapabilitiesRequired: []Capability{
				{Name: "http", Config: map[string]any{"allow_hosts": []any{"api.slack.com"}}},
				{Name: "secret"},
			},
		}},
		ACL: ACL{ExposureScope: "instance"},
		Secrets: []Secret{{
			Slot: "slack_token", Description: "Slack Bot OAuth token", Scope: "principal", Required: true,
		}},
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mut     func(*Manifest)
		wantErr string
	}{
		{"baseline_ok", func(m *Manifest) {}, ""},
		{"schema_version_too_new", func(m *Manifest) { m.SchemaVersion = 99 }, "schema_version"},
		{"http_without_allow_hosts", func(m *Manifest) {
			m.Tools[0].CapabilitiesRequired = []Capability{{Name: "http"}}
		}, "allow_hosts"},
		{"http_wildcard_host", func(m *Manifest) {
			m.Tools[0].CapabilitiesRequired = []Capability{{Name: "http", Config: map[string]any{"allow_hosts": []any{"*"}}}}
		}, "wildcard"},
		{"bad_semver", func(m *Manifest) { m.Version = "1.x" }, "semver"},
		{"container_without_idle_timeout", func(m *Manifest) {
			m.Runtime.Kind = "container-long-running"
			m.Runtime.IdleTimeoutS = nil
		}, "idle_timeout_s"},
		{"oci_without_digest", func(m *Manifest) {
			m.Source.Language = "oci-image"
			m.Source.OCIImage = "registry/img:tag"
			m.Source.OCIDigest = ""
		}, "oci_digest"},
		{"gotinygo_without_toolchain", func(m *Manifest) {
			m.Source.Language = "go-tinygo"
			m.Source.Toolchain = ""
		}, "toolchain"},
		{"missing_quotas_timeout", func(m *Manifest) { m.Runtime.Quotas.TimeoutS = 0 }, "timeout_s"},
		{"missing_quotas_memory", func(m *Manifest) { m.Runtime.Quotas.MemoryMB = 0 }, "memory_mb"},
		{"memory_mb_too_large", func(m *Manifest) { m.Runtime.Quotas.MemoryMB = 8192 }, "memory_mb"},
		{"missing_cpu_ms", func(m *Manifest) { m.Runtime.Quotas.CPUMs = 0 }, "cpu_ms"},
		{"acl_tenant_without_id", func(m *Manifest) {
			m.ACL.ExposureScope = "tenant"
			m.ACL.TenantID = ""
		}, "tenant_id"},
		{"max_token_ttl_negative", func(m *Manifest) { v := -1; m.Runtime.MaxTokenTTLS = &v }, "max_token_ttl_s"},
		{"toolchain_path_traversal", func(m *Manifest) { m.Source.Toolchain = "../../etc" }, "toolchain"},
		{"oci_digest_bad_format", func(m *Manifest) {
			m.Source.Language = "oci-image"
			m.Source.OCIImage = "registry/img:tag"
			m.Source.OCIDigest = "not-a-digest"
			m.Runtime.Kind = "container-long-running"
			idle := 300
			m.Runtime.IdleTimeoutS = &idle
		}, "oci_digest"},
		{"language_kind_mismatch", func(m *Manifest) {
			m.Source.Language = "oci-image"
			m.Source.OCIImage = "registry/img:tag"
			m.Source.OCIDigest = "sha256:" + strings.Repeat("a", 64)
			// runtime.kind stays wasm-per-call (from fixture) -> inconsistent
		}, "container-long-running"},
		{"duplicate_tool_id", func(m *Manifest) {
			m.Tools = append(m.Tools, Tool{ID: m.Tools[0].ID, Description: "dup"})
		}, "duplicate tool id"},
		{"unknown_capability", func(m *Manifest) {
			m.Tools[0].CapabilitiesRequired = []Capability{{Name: "foobar"}}
		}, "unknown capability"},
		{"iskra_capability_rejected", func(m *Manifest) {
			m.Tools[0].CapabilitiesRequired = []Capability{{Name: "iskra.kb"}}
		}, "unknown capability"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := validFixture()
			c.mut(m)
			err := Validate(m)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("expected error containing %q, got %v", c.wantErr, err)
			}
		})
	}
}

func TestValidate_ShowViewCapability(t *testing.T) {
	m := validFixture()
	m.Tools[0].CapabilitiesRequired = []Capability{{Name: "show_view"}}
	if err := Validate(m); err != nil {
		t.Fatalf("show_view should be a valid capability: %v", err)
	}
}

func TestValidate_AcceptsLLMAndReservedCaps(t *testing.T) {
	for _, name := range []string{"llm", "kb", "files", "progress"} {
		m := validFixture()
		m.Tools[0].CapabilitiesRequired = []Capability{{Name: name}}
		if err := Validate(m); err != nil {
			t.Fatalf("capability %q must be valid, got: %v", name, err)
		}
	}
	m := validFixture()
	m.Tools[0].CapabilitiesRequired = []Capability{{Name: "iskra.llm.chat"}}
	if err := Validate(m); err == nil {
		t.Fatal("iskra.* namespace must be rejected after retire")
	}
}

func TestValidate_RequestInputCapability(t *testing.T) {
	m := validFixture()
	m.Tools[0].CapabilitiesRequired = []Capability{{Name: "request_input"}}
	m.Runtime.Quotas.InteractionTimeoutS = 300
	if err := Validate(m); err != nil {
		t.Fatalf("request_input + interaction_timeout_s should be valid: %v", err)
	}
}

func TestValidate_RequestInputRequiresInteractionTimeout(t *testing.T) {
	m := validFixture()
	m.Tools[0].CapabilitiesRequired = []Capability{{Name: "request_input"}}
	m.Runtime.Quotas.InteractionTimeoutS = 0 // not set
	if err := Validate(m); err == nil {
		t.Fatal("request_input without interaction_timeout_s must fail validation")
	}
}

func TestValidate_InteractionTimeoutWithoutRequestInputOK(t *testing.T) {
	m := validFixture()
	// Non-interactive skill: interaction_timeout_s not required (ignored).
	if err := Validate(m); err != nil {
		t.Fatalf("non-interactive skill must not require interaction_timeout_s: %v", err)
	}
}

func validMarkdownManifest() *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		Bundle:        "p.uuid.refund",
		Version:       "1.0.0",
		Source:        Source{Language: "markdown"},
		Runtime:       Runtime{Kind: "prompt"},
		Tools:         []Tool{{ID: "refund_proc", Description: "when a refund is requested", BodyFile: "refund.md"}},
		ACL:           ACL{ExposureScope: "principal", PrincipalID: "uuid"},
	}
}

func TestValidate_Markdown_OK(t *testing.T) {
	if err := Validate(validMarkdownManifest()); err != nil {
		t.Fatalf("valid markdown manifest rejected: %v", err)
	}
}

func TestValidate_Markdown_RequiresBodyFile(t *testing.T) {
	m := validMarkdownManifest()
	m.Tools[0].BodyFile = ""
	if err := Validate(m); err == nil {
		t.Fatal("expected error for markdown tool without body_file")
	}
}

func TestValidate_Markdown_BodyFileMustBeMD(t *testing.T) {
	m := validMarkdownManifest()
	m.Tools[0].BodyFile = "refund.txt"
	if err := Validate(m); err == nil {
		t.Fatal("expected error for non-.md body_file")
	}
}

func TestValidate_Markdown_RejectsCapabilities(t *testing.T) {
	m := validMarkdownManifest()
	m.Tools[0].CapabilitiesRequired = []Capability{{Name: "http", Config: map[string]any{"allow_hosts": []any{"example.com"}}}}
	if err := Validate(m); err == nil {
		t.Fatal("expected error: markdown tools cannot declare capabilities")
	}
}

func TestValidate_Markdown_RejectsSecrets(t *testing.T) {
	m := validMarkdownManifest()
	m.Secrets = []Secret{{Slot: "token", Scope: "account"}}
	if err := Validate(m); err == nil {
		t.Fatal("expected error: markdown bundles cannot declare secrets")
	}
}

func TestValidate_Markdown_RejectsWrongRuntime(t *testing.T) {
	m := validMarkdownManifest()
	m.Runtime.Kind = "wasm-per-call"
	if err := Validate(m); err == nil {
		t.Fatal("expected consistency error: markdown requires runtime.kind prompt")
	}
}
