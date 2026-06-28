// Package manifest is the public bundle manifest spec (v1). It is the ONLY
// hub package an upstream platform imports; keep it dependency-light and
// stable — the shape is a frozen contract across the open-source split.
package manifest

// CurrentSchemaVersion is the highest manifest schema the hub understands.
const CurrentSchemaVersion = 1

// Manifest is a bundle manifest (manifest.yaml v1). See design §5.2.
type Manifest struct {
	SchemaVersion int      `yaml:"schema_version"`
	Bundle        string   `yaml:"bundle"`
	Version       string   `yaml:"version"`
	Author        string   `yaml:"author"`
	Description   string   `yaml:"description"`
	License       string   `yaml:"license"`
	Source        Source   `yaml:"source"`
	Runtime       Runtime  `yaml:"runtime"`
	Tools         []Tool   `yaml:"tools"`
	ACL           ACL      `yaml:"acl"`
	Secrets       []Secret `yaml:"secrets"`
}

// Source describes where the bundle's executable comes from.
type Source struct {
	Language  string `yaml:"language"`   // go-tinygo | oci-image
	Toolchain string `yaml:"toolchain"`  // required if go-tinygo (part of build cache-key)
	OCIImage  string `yaml:"oci_image"`  // required if oci-image
	OCIDigest string `yaml:"oci_digest"` // required if oci-image; = image config ID (sha256)
}

// Runtime describes how the bundle executes.
type Runtime struct {
	Kind         string `yaml:"kind"`            // wasm-per-call | container-long-running
	IdleTimeoutS *int   `yaml:"idle_timeout_s"`  // required if container-long-running
	MaxTokenTTLS *int   `yaml:"max_token_ttl_s"` // optional; nil = hub does not bound issuer exp
	Quotas       Quotas `yaml:"quotas"`
}

// Quotas are the per-invoke resource limits.
type Quotas struct {
	TimeoutS            int `yaml:"timeout_s"`             // enforced (wall-clock deadline of the whole invoke)
	MemoryMB            int `yaml:"memory_mb"`             // enforced (WithMemoryLimitPages)
	CPUMs               int `yaml:"cpu_ms"`                // RESERVED in Phase 1 (validator requires the field)
	InteractionTimeoutS int `yaml:"interaction_timeout_s"` // A4 Phase 2: max wait for request_input answer; required when request_input is declared
}

// Capability is a declared capability requirement of a tool.
type Capability struct {
	Name   string         `yaml:"name"`   // base capability: http|kv|secret|log|now|random|llm|kb|progress
	Config map[string]any `yaml:"config"` // e.g. http: {allow_hosts: [...]}
}

// Tool is a single callable tool exposed by the bundle.
type Tool struct {
	ID                   string         `yaml:"id"`
	Description          string         `yaml:"description"`
	InputSchema          map[string]any `yaml:"input_schema"`
	CapabilitiesRequired []Capability   `yaml:"capabilities_required"`
	// BodyFile names the markdown file (inside source.zip) returned verbatim on
	// tools/call. Required for source.language=markdown; empty otherwise.
	BodyFile string `yaml:"body_file"`
}

// ACL controls who may see/invoke the bundle.
type ACL struct {
	ExposureScope string `yaml:"exposure_scope"` // instance|tenant|principal
	TenantID      string `yaml:"tenant_id"`      // required if scope=tenant
	PrincipalID   string `yaml:"principal_id"`   // required if scope=principal
}

// Secret declares a secret slot the bundle expects (values are never stored).
type Secret struct {
	Slot        string `yaml:"slot"`
	Description string `yaml:"description"`
	Scope       string `yaml:"scope"` // opaque to the hub; vocabulary defined by the secret backend
	Required    bool   `yaml:"required"`
}
