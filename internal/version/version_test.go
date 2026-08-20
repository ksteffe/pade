package version

import "testing"

func TestStringDevDefaults(t *testing.T) {
	oldV, oldC, oldB := Version, Commit, BuildTime
	t.Cleanup(func() {
		Version, Commit, BuildTime = oldV, oldC, oldB
	})
	Version, Commit, BuildTime = "dev", "unknown", ""
	if got := String(); got != "dev" {
		t.Fatalf("String() = %q, want dev", got)
	}
}

func TestStringRelease(t *testing.T) {
	oldV, oldC, oldB := Version, Commit, BuildTime
	t.Cleanup(func() {
		Version, Commit, BuildTime = oldV, oldC, oldB
	})
	Version = "v0.1.0"
	Commit = "360b4bd"
	BuildTime = "2026-08-20T03:00:00Z"
	want := "v0.1.0 (360b4bd, built 2026-08-20T03:00:00Z)"
	if got := String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
