// Package version reports release metadata injected at link time.
package version

import (
	"fmt"
	"strings"
)

// Defaults are replaced via -ldflags for release builds.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = ""
)

// String returns a human-readable version line (SemVer + commit + optional build time).
func String() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		v = "dev"
	}
	out := v
	commit := strings.TrimSpace(Commit)
	if commit != "" && commit != "unknown" {
		out = fmt.Sprintf("%s (%s", out, commit)
		bt := strings.TrimSpace(BuildTime)
		if bt != "" && bt != "unknown" {
			out += ", built " + bt
		}
		out += ")"
	}
	return out
}
