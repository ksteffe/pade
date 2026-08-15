// Package cliproc provides shared subprocess I/O and environment helpers for
// CLI-backed binding adapters (1Password, Keeper Commander, and similar).
package cliproc

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
)

// MaxOutput is the per-stream capture limit for CLI stdout/stderr (1 MiB).
const MaxOutput = 1 << 20

// ErrOutputLimit means a CLI stream exceeded MaxOutput (or the buffer Limit).
var ErrOutputLimit = errors.New("cli output exceeded size limit")

// LimitedBuffer captures output up to Limit bytes and sets Exceed when more arrives.
type LimitedBuffer struct {
	Buf    bytes.Buffer
	Limit  int
	Exceed bool
}

// Write implements io.Writer.
func (l *LimitedBuffer) Write(p []byte) (int, error) {
	if l.Exceed {
		return len(p), nil
	}
	remain := l.Limit - l.Buf.Len()
	if remain <= 0 {
		l.Exceed = true
		return len(p), nil
	}
	if len(p) > remain {
		_, _ = l.Buf.Write(p[:remain])
		l.Exceed = true
		return len(p), nil
	}
	return l.Buf.Write(p)
}

// Bytes returns the captured bytes.
func (l *LimitedBuffer) Bytes() []byte { return l.Buf.Bytes() }

// String returns the captured string.
func (l *LimitedBuffer) String() string { return l.Buf.String() }

// Ensure LimitedBuffer implements io.Writer.
var _ io.Writer = (*LimitedBuffer)(nil)

// Environ builds a deliberate subprocess environment from the current process
// environment. extraExact and extraPrefix extend the shared allowlist
// (for example OP_* for 1Password or KEEPER_* for Commander).
//
// PATH is always present: if the parent environment omitted it, a minimal
// /usr/bin:/bin fallback is added so CLI tools still resolve.
func Environ(extraExact map[string]struct{}, extraPrefix []string) []string {
	allowExact := map[string]struct{}{
		"PATH":     {},
		"HOME":     {},
		"USER":     {},
		"LOGNAME":  {},
		"SHELL":    {},
		"TMPDIR":   {},
		"TMP":      {},
		"TEMP":     {},
		"LANG":     {},
		"LANGUAGE": {},
		"TZ":       {},
		"TERM":     {},
	}
	for k := range extraExact {
		allowExact[k] = struct{}{}
	}
	allowPrefix := []string{"LC_", "XDG_"}
	allowPrefix = append(allowPrefix, extraPrefix...)

	env := os.Environ()
	out := make([]string, 0, len(env)/4+8)
	hasPATH := false
	for _, e := range env {
		key, _, _ := strings.Cut(e, "=")
		if _, ok := allowExact[key]; ok {
			if key == "PATH" {
				hasPATH = true
			}
			out = append(out, e)
			continue
		}
		for _, p := range allowPrefix {
			if strings.HasPrefix(key, p) {
				out = append(out, e)
				break
			}
		}
	}
	if !hasPATH {
		out = append(out, "PATH=/usr/bin:/bin")
	}
	return out
}
