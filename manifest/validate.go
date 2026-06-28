package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

// semverRe is a pragmatic SemVer matcher: MAJOR.MINOR.PATCH with optional
// prerelease and build metadata. Sufficient for immutable version pinning.
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// toolchainRe constrains source.toolchain to a safe charset. The toolchain is
// part of the build cache-key/path (CacheKey), so an unconstrained value could
// inject path separators or ".." and escape the build cache root.
var toolchainRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ociDigestRe matches an image config ID: optional sha256: prefix + 64 hex.
var ociDigestRe = regexp.MustCompile(`^(sha256:)?[0-9a-f]{64}$`)

// baseCapabilities is the frozen v1 set of neutral capability names (design §5.2,
// ADR-079). llm/kb/files/progress replace the retired iskra.* namespace; progress
// has no provider yet (a tool requiring it is deactivated at tools/list, §6.2).
var baseCapabilities = map[string]struct{}{
	"http": {}, "kv": {}, "secret": {}, "log": {}, "now": {}, "random": {},
	"llm": {}, "kb": {}, "files": {}, "progress": {}, "show_view": {}, "request_input": {},
}

// maxMemoryMB caps memory_mb for wasm-per-call. wazero's
// WithMemoryLimitPages(mb*16) panics when the page count exceeds 65536
// (mb > 4096), so the validator rejects it (design §8.1). The cap is
// wasm-specific — container memory is bounded by the container runtime.
const maxMemoryMB = 4096

// Validate checks a manifest against the v1 spec, failing fast on the first
// violation. Errors wrap a field name so callers (and tests) can identify the
// rule that fired.
func Validate(m *Manifest) error {
	if m.SchemaVersion < 1 || m.SchemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("manifest: unsupported schema_version %d (this hub supports 1..%d)", m.SchemaVersion, CurrentSchemaVersion)
	}
	if m.Bundle == "" {
		return fmt.Errorf("manifest: bundle is required")
	}
	if !semverRe.MatchString(m.Version) {
		return fmt.Errorf("manifest: version %q is not valid semver", m.Version)
	}

	if err := validateSource(m.Source); err != nil {
		return err
	}
	if err := validateRuntime(m.Runtime); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(m.Tools))
	for _, tool := range m.Tools {
		if err := validateTool(tool); err != nil {
			return err
		}
		if _, dup := seen[tool.ID]; dup {
			return fmt.Errorf("manifest: duplicate tool id %q", tool.ID)
		}
		seen[tool.ID] = struct{}{}
	}
	if err := validateACL(m.ACL); err != nil {
		return err
	}
	// request_input requires an explicit interaction_timeout_s (A4 Phase 2):
	// explicit values, no silent defaults. Check after field-level validation so
	// per-field errors take precedence.
	if requiresInteractionTimeout(m) && m.Runtime.Quotas.InteractionTimeoutS <= 0 {
		return fmt.Errorf("manifest: runtime.quotas.interaction_timeout_s is required and must be > 0 when request_input is declared")
	}
	// Markdown-specific invariants: no runtime execution, so capabilities and
	// secrets are not applicable; each tool must supply a .md body_file.
	if m.Source.Language == "markdown" {
		if len(m.Secrets) > 0 {
			return fmt.Errorf("manifest: markdown bundles must not declare secrets")
		}
		for _, t := range m.Tools {
			if len(t.CapabilitiesRequired) > 0 {
				return fmt.Errorf("manifest: markdown tool %q must not require capabilities", t.ID)
			}
			if t.BodyFile == "" {
				return fmt.Errorf("manifest: markdown tool %q requires body_file", t.ID)
			}
			if !strings.HasSuffix(t.BodyFile, ".md") {
				return fmt.Errorf("manifest: markdown tool %q body_file %q must end with .md", t.ID, t.BodyFile)
			}
		}
	}
	// Cross-field consistency runs last so field-level errors take precedence.
	return validateSourceRuntimeConsistency(m.Source, m.Runtime)
}

// requiresInteractionTimeout reports whether any tool in the manifest declares
// the request_input capability (A4 Phase 2).
func requiresInteractionTimeout(m *Manifest) bool {
	for _, t := range m.Tools {
		for _, c := range t.CapabilitiesRequired {
			if c.Name == "request_input" {
				return true
			}
		}
	}
	return false
}

func validateSource(s Source) error {
	switch s.Language {
	case "go-tinygo":
		if s.Toolchain == "" {
			return fmt.Errorf("manifest: source.toolchain is required for language go-tinygo")
		}
		if !toolchainRe.MatchString(s.Toolchain) {
			return fmt.Errorf("manifest: source.toolchain %q invalid (allowed: letters, digits, . _ -)", s.Toolchain)
		}
	case "oci-image":
		if s.OCIImage == "" {
			return fmt.Errorf("manifest: source.oci_image is required for language oci-image")
		}
		if s.OCIDigest == "" {
			return fmt.Errorf("manifest: source.oci_digest is required for language oci-image")
		}
		if !ociDigestRe.MatchString(s.OCIDigest) {
			return fmt.Errorf("manifest: source.oci_digest %q invalid (want optional sha256: prefix + 64 hex)", s.OCIDigest)
		}
	case "markdown":
		// markdown bundles ship .md files in source.zip; no toolchain/oci fields.
		return nil
	default:
		return fmt.Errorf("manifest: source.language %q invalid (want go-tinygo|oci-image|markdown)", s.Language)
	}
	return nil
}

func validateRuntime(r Runtime) error {
	switch r.Kind {
	case "wasm-per-call":
		// The 4096 MB cap is a wazero (wasm) constraint (§8.1); it does not
		// apply to container-long-running, whose memory is bounded by Docker.
		if r.Quotas.MemoryMB > maxMemoryMB {
			return fmt.Errorf("manifest: runtime.quotas.memory_mb %d exceeds wasm limit %d", r.Quotas.MemoryMB, maxMemoryMB)
		}
	case "container-long-running":
		if r.IdleTimeoutS == nil || *r.IdleTimeoutS <= 0 {
			return fmt.Errorf("manifest: runtime.idle_timeout_s is required and must be > 0 for container-long-running")
		}
	case "prompt":
		// markdown skills do not execute; quotas/ttl are not applicable.
		return nil
	default:
		return fmt.Errorf("manifest: runtime.kind %q invalid (want wasm-per-call|container-long-running|prompt)", r.Kind)
	}

	if r.MaxTokenTTLS != nil && *r.MaxTokenTTLS <= 0 {
		return fmt.Errorf("manifest: runtime.max_token_ttl_s must be > 0 when set (use null for unbounded)")
	}

	if r.Quotas.TimeoutS <= 0 {
		return fmt.Errorf("manifest: runtime.quotas.timeout_s is required and must be > 0")
	}
	if r.Quotas.MemoryMB <= 0 {
		return fmt.Errorf("manifest: runtime.quotas.memory_mb is required and must be > 0")
	}
	if r.Quotas.CPUMs <= 0 {
		return fmt.Errorf("manifest: runtime.quotas.cpu_ms is required and must be > 0 (reserved in Phase 1)")
	}
	return nil
}

func validateTool(t Tool) error {
	if t.ID == "" {
		return fmt.Errorf("manifest: tool.id is required")
	}
	for _, c := range t.CapabilitiesRequired {
		if c.Name == "" {
			return fmt.Errorf("manifest: tool %q has a capability with empty name", t.ID)
		}
		if !validCapabilityName(c.Name) {
			return fmt.Errorf("manifest: tool %q has unknown capability %q (want http|kv|secret|log|now|random|llm|kb|files|progress|show_view|request_input)", t.ID, c.Name)
		}
		if c.Name == "http" {
			if err := validateHTTPCapability(t.ID, c); err != nil {
				return err
			}
		}
	}
	return nil
}

// validCapabilityName reports whether name is a known base capability (design §5.2,
// ADR-079). The iskra.* namespace is retired; only names in baseCapabilities are valid.
func validCapabilityName(name string) bool {
	_, ok := baseCapabilities[name]
	return ok
}

// validateSourceRuntimeConsistency enforces the language/runtime pairing:
// go-tinygo runs as wasm-per-call, oci-image as container-long-running,
// markdown runs as prompt.
func validateSourceRuntimeConsistency(s Source, r Runtime) error {
	switch s.Language {
	case "go-tinygo":
		if r.Kind != "wasm-per-call" {
			return fmt.Errorf("manifest: language go-tinygo requires runtime.kind wasm-per-call, got %q", r.Kind)
		}
	case "oci-image":
		if r.Kind != "container-long-running" {
			return fmt.Errorf("manifest: language oci-image requires runtime.kind container-long-running, got %q", r.Kind)
		}
	case "markdown":
		if r.Kind != "prompt" {
			return fmt.Errorf("manifest: language markdown requires runtime.kind prompt, got %q", r.Kind)
		}
	}
	return nil
}

func validateHTTPCapability(toolID string, c Capability) error {
	hosts, ok := toStringSlice(c.Config["allow_hosts"])
	if !ok || len(hosts) == 0 {
		return fmt.Errorf("manifest: tool %q http capability requires non-empty allow_hosts (anti-SSRF)", toolID)
	}
	for _, h := range hosts {
		if h == "*" {
			return fmt.Errorf("manifest: tool %q http capability allow_hosts must not contain wildcard %q", toolID, h)
		}
	}
	return nil
}

func validateACL(a ACL) error {
	switch a.ExposureScope {
	case "instance":
	case "tenant":
		if a.TenantID == "" {
			return fmt.Errorf("manifest: acl.tenant_id is required for exposure_scope tenant")
		}
	case "principal":
		if a.PrincipalID == "" {
			return fmt.Errorf("manifest: acl.principal_id is required for exposure_scope principal")
		}
	default:
		return fmt.Errorf("manifest: acl.exposure_scope %q invalid (want instance|tenant|principal)", a.ExposureScope)
	}
	return nil
}

// toStringSlice accepts []string or []any (YAML decodes lists as []any).
func toStringSlice(v any) ([]string, bool) {
	switch xs := v.(type) {
	case []string:
		return xs, true
	case []any:
		out := make([]string, 0, len(xs))
		for _, e := range xs {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}
