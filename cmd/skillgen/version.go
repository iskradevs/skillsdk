package main

import (
	_ "embed"
	"fmt"
	"strings"
)

// sdkVersion is the SDK's own version, bumped on releases.
const sdkVersion = "0.1.0"

// mcphubRev is the abbreviated git tree hash of services/mcphub the vendored
// codegen/manifest were copied from (written by scripts/sync.sh). It lets an
// author confirm the tool matches the hub's ABI.
//
//go:embed mcphub_rev.txt
var mcphubRev string

func versionString() string {
	return fmt.Sprintf("skillgen %s (mcphub %s)", sdkVersion, strings.TrimSpace(mcphubRev))
}
