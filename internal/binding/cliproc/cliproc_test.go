package cliproc

import (
	"strings"
	"testing"
)

func TestLimitedBufferUnderLimit(t *testing.T) {
	var buf LimitedBuffer
	buf.Limit = 10
	if _, err := buf.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if buf.Exceed {
		t.Fatal("should not exceed under limit")
	}
	if got := string(buf.Bytes()); got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestLimitedBufferAtLimit(t *testing.T) {
	var buf LimitedBuffer
	buf.Limit = 5
	if _, err := buf.Write([]byte("12345")); err != nil {
		t.Fatal(err)
	}
	if buf.Exceed {
		t.Fatal("exact fill should not set exceed")
	}
}

func TestLimitedBufferOverLimit(t *testing.T) {
	var buf LimitedBuffer
	buf.Limit = 5
	if _, err := buf.Write([]byte("123456789")); err != nil {
		t.Fatal(err)
	}
	if !buf.Exceed {
		t.Fatal("expected exceed")
	}
	if got := string(buf.Bytes()); got != "12345" {
		t.Fatalf("got %q", got)
	}
}

func TestLimitedBufferPostExceedWrites(t *testing.T) {
	var buf LimitedBuffer
	buf.Limit = 3
	_, _ = buf.Write([]byte("abcdef"))
	n, err := buf.Write([]byte("xyz"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("n=%d", n)
	}
	if got := string(buf.Bytes()); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestEnvironPATHFallback(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("UNRELATED", "secret")

	env := Environ(nil, nil)
	foundPATH := false
	for _, e := range env {
		if strings.HasPrefix(e, "UNRELATED=") {
			t.Fatal("unrelated var leaked")
		}
		if e == "PATH=/usr/bin:/bin" {
			foundPATH = true
		}
	}
	if !foundPATH {
		t.Fatal("expected PATH fallback")
	}
}

func TestEnvironPrefixAllowlist(t *testing.T) {
	t.Setenv("PADE_OK", "1")
	t.Setenv("KEEPER_TOKEN", "x")
	t.Setenv("RANDOM_SECRET", "nope")

	env := Environ(nil, []string{"KEEPER_"})
	hasPADE := false
	hasKeeper := false
	for _, e := range env {
		if strings.HasPrefix(e, "RANDOM_SECRET=") {
			t.Fatal("random secret leaked")
		}
		if strings.HasPrefix(e, "PADE_OK=") {
			hasPADE = false // PADE_ not in default extraPrefix here
		}
		if strings.HasPrefix(e, "KEEPER_TOKEN=") {
			hasKeeper = true
		}
	}
	if !hasKeeper {
		t.Fatal("expected KEEPER_ prefix")
	}
	_ = hasPADE

	env2 := Environ(nil, []string{"PADE_"})
	found := false
	for _, e := range env2 {
		if e == "PADE_OK=1" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected PADE_ prefix allowlist")
	}
}

func TestEnvironExactAllowlist(t *testing.T) {
	t.Setenv("KSM_CONFIG", "cfg")
	t.Setenv("OTHER", "x")

	env := Environ(map[string]struct{}{"KSM_CONFIG": {}}, nil)
	hasKSM := false
	for _, e := range env {
		if e == "KSM_CONFIG=cfg" {
			hasKSM = true
		}
		if strings.HasPrefix(e, "OTHER=") {
			t.Fatal("OTHER leaked")
		}
	}
	if !hasKSM {
		t.Fatal("expected exact allowlist entry")
	}
}
