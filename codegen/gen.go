package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"

	"github.com/iskradevs/skillsdk/manifest"
)

// knownCaps maps each base capability the SDK supports to its template file.
var knownCaps = map[string]string{
	"http":          "templates/http.tmpl",
	"kv":            "templates/kv.tmpl",
	"log":           "templates/log.tmpl",
	"now":           "templates/now.tmpl",
	"random":        "templates/random.tmpl",
	"llm":           "templates/llm.tmpl",
	"secret":        "templates/secret.tmpl",
	"kb":            "templates/kb.tmpl",
	"files":         "templates/files.tmpl",
	"progress":      "templates/progress.tmpl",
	"show_view":     "templates/showview.tmpl",
	"request_input": "templates/requestinput.tmpl",
}

func isDeferredCap(name string) bool {
	// No reserved-without-wrapper capabilities remain: progress got its wrapper
	// in ADR-082. The mechanism stays for future reserved names.
	return false
}

// Generate produces the guest SDK source for a bundle manifest: frozen ABI
// boilerplate plus typed wrappers for the UNION of capabilities_required across
// all tools. pkg is the guest package name (usually "main"). Output is
// gofmt-formatted; a format failure is returned as an error (never emit
// unformatted source).
func Generate(m manifest.Manifest, pkg string) ([]byte, error) {
	known, deferred, err := collectCaps(m)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := boilerplateTmpl.Execute(&buf, boilerplateData{Package: pkg, Deferred: deferred}); err != nil {
		return nil, fmt.Errorf("execute boilerplate: %w", err)
	}
	// Capability sections are static Go fragments (no template syntax): only the
	// boilerplate is executed as a template. Each is appended verbatim, then the
	// whole buffer is gofmt-normalized below.
	for _, name := range known {
		body, err := templatesFS.ReadFile(knownCaps[name])
		if err != nil {
			return nil, fmt.Errorf("read section %q: %w", name, err)
		}
		buf.WriteString("\n")
		buf.Write(body)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w", err)
	}
	return src, nil
}

// collectCaps returns the sorted union of known base capabilities required by
// the bundle, plus the sorted list of skipped reserved capabilities. An unknown,
// non-reserved capability is a hard error (no silent default — CLAUDE.md).
func collectCaps(m manifest.Manifest) (known, deferred []string, err error) {
	knownSet := map[string]bool{}
	defSet := map[string]bool{}
	for _, tool := range m.Tools {
		for _, capReq := range tool.CapabilitiesRequired {
			name := capReq.Name
			switch {
			case knownCaps[name] != "":
				knownSet[name] = true
			case isDeferredCap(name):
				defSet[name] = true
			default:
				return nil, nil, fmt.Errorf("unknown capability %q (tool %q): no generator section and not a reserved capability", name, tool.ID)
			}
		}
	}
	return sortedKeys(knownSet), sortedKeys(defSet), nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
