// Not a Zig file: this stands for the compile-time platform checks that
// libghostty-vt makes in its keyboard encoders (`builtin.os.tag == .macos`
// and `builtin.target.os.tag.isDarwin()` in input/key_encode.zig,
// input/key_mods.zig and input/key.zig).
//
// The Go library is platform-neutral (project owner's decision,
// 2026-10-04): it takes the macOS behaviour when it runs on an Apple
// platform and the other behaviour everywhere else, exactly as upstream does
// at compile time. Go decides at run time from platformGOOS, so that tests
// can select either behaviour on any machine.
//
// Mapping of the two Zig checks:
//   - `os.tag == .macos`  -> platformGOOS == "darwin"
//   - `os.tag.isDarwin()` -> platformGOOS == "darwin" or "ios" (Zig's Darwin
//     family also includes tvOS, watchOS, visionOS and DriverKit, which Go
//     does not target)
//
// The per-file functions inputKeyEncodeIsMacos, inputKeyEncodeIsDarwin,
// inputKeyModsIsDarwin and inputKeyIsDarwin keep their names and read this.

package terminal

import "runtime"

// platformGOOS is the platform whose behaviour the encoders follow. It is
// runtime.GOOS, set once at start-up. Tests set it to "linux", "darwin" or
// "ios" and restore it afterwards, through one shared helper
// (testPlatform), so tests that change it must not run in parallel.
var platformGOOS = runtime.GOOS

// platformIsMacos stands for `builtin.os.tag == .macos`.
func platformIsMacos() bool { return platformGOOS == "darwin" }

// platformIsDarwin stands for `builtin.target.os.tag.isDarwin()`.
func platformIsDarwin() bool { return platformGOOS == "darwin" || platformGOOS == "ios" }
