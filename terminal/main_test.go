package terminal

// The test binary's entry point.
//
// pageSlowRuntimeSafety (page.go) is upstream's build option
// `build_options.slow_runtime_safety`. Upstream sets it only in Debug
// builds, and Zig's tests run as a Debug build, so every translated test
// ran with the slow integrity checks on (the debug corpus replay did too).
// The Go default is off, as in upstream's release builds (the ReleaseSafe
// recorder of the corpus and shipped Ghostty). The test binary switches it
// on here, before any test runs, so all tests and the corpus replay keep
// running with the checks.

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	pageSlowRuntimeSafety = true
	os.Exit(m.Run())
}

// setPageSlowRuntimeSafety sets pageSlowRuntimeSafety to v for the rest of
// the test and restores the previous value when the test ends. Call it
// before the test creates any Page or PageList (their pause counters only
// count while the option is on). A test that calls it must not run in
// parallel.
func setPageSlowRuntimeSafety(t *testing.T, v bool) {
	t.Helper()
	old := pageSlowRuntimeSafety
	pageSlowRuntimeSafety = v
	t.Cleanup(func() { pageSlowRuntimeSafety = old })
}
