package codegen

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/iskradevs/skillsdk/manifest"
)

var update = flag.Bool("update", false, "update golden files")

func manifestWith(caps ...string) manifest.Manifest {
	tool := manifest.Tool{ID: "t"}
	for _, c := range caps {
		tool.CapabilitiesRequired = append(tool.CapabilitiesRequired, manifest.Capability{Name: c})
	}
	return manifest.Manifest{Tools: []manifest.Tool{tool}}
}

func TestGenerate_Golden(t *testing.T) {
	cases := []struct {
		name string
		m    manifest.Manifest
	}{
		{"all_caps", manifestWith("http", "kv", "log", "now", "random")},
		{"zero_caps", manifestWith()},
		{"now_only", manifestWith("now")},
		{"progress_only", manifestWith("progress")},
		{"kb_files", manifestWith("kb", "files")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Generate(tc.m, "main")
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			golden := filepath.Join("testdata", "golden", tc.name+".go.golden")
			if *update {
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run with -update first): %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("golden mismatch for %s (run -update to refresh):\n--- got ---\n%s", tc.name, got)
			}
		})
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	m := manifestWith("random", "http", "now", "kv", "log")
	a, err := Generate(m, "main")
	if err != nil {
		t.Fatalf("generate a: %v", err)
	}
	b, err := Generate(m, "main")
	if err != nil {
		t.Fatalf("generate b: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("generation is not deterministic")
	}
}

func TestGenerate_UnknownCapFailsFast(t *testing.T) {
	if _, err := Generate(manifestWith("bogus"), "main"); err == nil {
		t.Fatal("want error for unknown capability")
	}
}

func TestGenerate_EmitsKBAndFilesWrappers(t *testing.T) {
	src, err := Generate(manifestWith("kb", "files"), "main")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{"func KB(", "func FileRead(", "func FileList("} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("generated SDK missing %q", want)
		}
	}
}

func TestCodegen_ShowViewWrapper(t *testing.T) {
	out, err := Generate(manifestWith("show_view"), "main")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !bytes.Contains(out, []byte("func ShowView(")) {
		t.Errorf("generated guest missing ShowView wrapper:\n%s", out)
	}
	if !bytes.Contains(out, []byte(`capInvoke("show_view"`)) {
		t.Errorf("ShowView must call capInvoke(\"show_view\"):\n%s", out)
	}
}

func TestCodegen_RequestInputWrapper(t *testing.T) {
	out, err := Generate(manifestWith("request_input"), "main")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !bytes.Contains(out, []byte("func RequestInput(")) {
		t.Errorf("generated guest missing RequestInput wrapper:\n%s", out)
	}
	if !bytes.Contains(out, []byte(`capInvoke("request_input"`)) {
		t.Errorf("RequestInput must call capInvoke(\"request_input\"):\n%s", out)
	}
}

func TestCodegen_RequestInputStandaloneCompiles(t *testing.T) {
	// A request_input-only skill must not reference ShowViewKV (Review C4).
	out, err := Generate(manifestWith("request_input"), "main")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if bytes.Contains(out, []byte("ShowViewKV")) {
		t.Errorf("request_input-only generated code must not reference ShowViewKV:\n%s", out)
	}
}

func TestGenerate_UnionDedupes(t *testing.T) {
	m := manifest.Manifest{Tools: []manifest.Tool{
		{ID: "a", CapabilitiesRequired: []manifest.Capability{{Name: "now"}}},
		{ID: "b", CapabilitiesRequired: []manifest.Capability{{Name: "now"}}},
	}}
	got, err := Generate(m, "main")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if n := bytes.Count(got, []byte("func Now()")); n != 1 {
		t.Fatalf("Now() emitted %d times, want 1", n)
	}
}
