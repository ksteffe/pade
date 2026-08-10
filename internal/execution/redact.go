package execution

import (
	"io"
	"unicode/utf8"
)

const redactedPlaceholder = "[REDACTED]"

// secretRedactor is a streaming writer that replaces exact occurrences of
// known secret values with [REDACTED]. It is defense in depth for agent
// transcripts — not a security boundary.
type secretRedactor struct {
	dst      io.Writer
	secrets  []string
	maxLen   int
	pending  []byte
	writeErr error
}

func newSecretRedactor(dst io.Writer, secrets []string) *secretRedactor {
	uniq := make([]string, 0, len(secrets))
	seen := map[string]struct{}{}
	maxLen := 0
	for _, s := range secrets {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		uniq = append(uniq, s)
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}
	return &secretRedactor{dst: dst, secrets: uniq, maxLen: maxLen}
}

func (r *secretRedactor) Write(p []byte) (int, error) {
	if r.writeErr != nil {
		return 0, r.writeErr
	}
	if len(r.secrets) == 0 {
		n, err := r.dst.Write(p)
		if err != nil {
			r.writeErr = err
		}
		return n, err
	}
	r.pending = append(r.pending, p...)
	if err := r.flush(false); err != nil {
		r.writeErr = err
		return 0, err
	}
	return len(p), nil
}

// Close flushes any remaining buffered bytes after redaction.
func (r *secretRedactor) Close() error {
	if r.writeErr != nil {
		return r.writeErr
	}
	if err := r.flush(true); err != nil {
		r.writeErr = err
		return err
	}
	return nil
}

func (r *secretRedactor) flush(final bool) error {
	for {
		idx, secretLen := r.findSecret(r.pending)
		if idx < 0 {
			break
		}
		if idx > 0 {
			if _, err := r.dst.Write(r.pending[:idx]); err != nil {
				return err
			}
		}
		if _, err := r.dst.Write([]byte(redactedPlaceholder)); err != nil {
			return err
		}
		r.pending = r.pending[idx+secretLen:]
	}
	if final {
		if len(r.pending) > 0 {
			if _, err := r.dst.Write(r.pending); err != nil {
				return err
			}
			r.pending = r.pending[:0]
		}
		return nil
	}
	// Keep a suffix that might be a partial secret match.
	keep := r.maxLen - 1
	if keep < 0 {
		keep = 0
	}
	if keep > len(r.pending) {
		keep = len(r.pending)
	}
	// Avoid splitting a UTF-8 rune at the keep boundary.
	emitEnd := len(r.pending) - keep
	for emitEnd > 0 && emitEnd < len(r.pending) && !utf8.RuneStart(r.pending[emitEnd]) {
		emitEnd--
	}
	if emitEnd > 0 {
		if _, err := r.dst.Write(r.pending[:emitEnd]); err != nil {
			return err
		}
		r.pending = r.pending[emitEnd:]
	}
	return nil
}

func (r *secretRedactor) findSecret(buf []byte) (index, length int) {
	index = -1
	for _, s := range r.secrets {
		sb := []byte(s)
		if len(sb) == 0 || len(sb) > len(buf) {
			continue
		}
		for i := 0; i+len(sb) <= len(buf); i++ {
			match := true
			for j := 0; j < len(sb); j++ {
				if buf[i+j] != sb[j] {
					match = false
					break
				}
			}
			if match {
				if index < 0 || i < index || (i == index && len(sb) > length) {
					index = i
					length = len(sb)
				}
				break
			}
		}
	}
	return index, length
}

func collectSecrets(envMaps ...map[string]string) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, m := range envMaps {
		for _, v := range m {
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
