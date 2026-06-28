// Command skillgen is the author tool from the Iskra Skill SDK: it generates the
// guest SDK (mcphub_gen.go) from a manifest, mirroring `mcphub gen`. It needs no
// server config — only a manifest file.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/iskradevs/skillsdk/codegen"
	"github.com/iskradevs/skillsdk/manifest"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "gen":
		err = runGen(os.Args[2:], os.Stdout)
	case "init":
		err = runInit(os.Args[2:], os.Stdout)
	case "validate":
		err = runValidate(os.Args[2:], os.Stdout)
	case "pack":
		err = runPack(os.Args[2:], os.Stdout)
	case "check":
		err = runCheck(os.Args[2:], os.Stdout)
	case "version":
		fmt.Fprintln(os.Stdout, versionString())
		return
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillgen %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

const usage = `usage:
  skillgen init <bundle> [dir]                 scaffold a new skill
  skillgen gen --manifest m.yaml --out dir     (re)generate mcphub_gen.go
  skillgen validate <manifest.yaml>            validate manifest (no acl, toolchain)
  skillgen check <dir>                         local TinyGo build smoke (optional)
  skillgen pack <dir>                          build flat source.zip for upload
  skillgen version`

// runGen reads a manifest, generates the guest SDK and writes <out>/mcphub_gen.go.
func runGen(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	manifestPath := fs.String("manifest", "", "path to manifest.yaml")
	outDir := fs.String("out", "", "output directory for mcphub_gen.go")
	pkg := fs.String("package", "main", "guest package name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *manifestPath == "" || *outDir == "" {
		return fmt.Errorf("gen requires --manifest and --out")
	}
	raw, err := os.ReadFile(*manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var m manifest.Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	src, err := codegen.Generate(m, *pkg)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}
	outPath := filepath.Join(*outDir, "mcphub_gen.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Fprintf(stdout, "wrote %s (%d bytes)\n", outPath, len(src))
	return nil
}
