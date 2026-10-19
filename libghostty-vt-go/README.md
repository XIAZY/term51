# libghostty-vt-go

**This is a port of libghostty-vt to Go.** libghostty-vt is the terminal
emulation core of [Ghostty](https://github.com/ghostty-org/ghostty); the
port follows its Zig source at tag v1.3.1 (commit 332b2aef) file by file
and function by function. It is MIT licensed, like the original, and it is
the terminal core of term51, the repository it is part of. It was written
in the first week of October 2026 and signed off on the 6th. It is frozen
at v0.1.0 by the project owner's decision: a change to the port needs his
word, and is then held to the rules below, which the port was written by.

It is a library, not a terminal: it parses what programs write, keeps the
screen, scrollback, modes and colours, reflows on resize, and encodes keys
and pastes. Drawing, the shell and its pseudo-terminal, and the answers to
programs' queries belong to the application, as they do with upstream.

**The rule of the port:** the same features and behaviour as upstream, in
the form a Go user expects. Every upstream test is translated, and about
190,000 recorded cases compare the terminal's full state, byte for byte,
with what the pinned upstream library produced. Pure Go: no cgo, no
dependencies, and no platform-specific code apart from the key encoding's
macOS option. It has been built and run on Linux (arm64) and on QNX
(BlackBerry 10, 32-bit ARM).

## The rules of the port

These were set by the project owner and the project lead while the port
was made, and every change since has been held to them. The details, and
every place the port has to differ from upstream, are in `docs/mapping.md`.

1. **Faithful, but in the form a Go user expects.** The same features and
   behaviour as upstream, within the scope set by the owner: libghostty-vt
   without Kitty graphics, tmux control mode and search, which are compiled
   out upstream too and are left for later. One Go file per Zig file with
   the same name, one Go function per Zig function in the same order with
   the same control flow, and upstream's comments carried over. No redesign
   and no optimisation during the port; the speed work came after it, on
   its own branches, measured against upstream
   (`docs/post-port-optimizations.md`).
2. **Ordinary Go memory.** Pages are Go slices of rows and cells, references
   are pointers or indexes, the garbage collector frees. No raw byte blocks
   and no offset arithmetic; functions that only compute byte layouts or
   hand memory back have no Go counterpart; allocator parameters are
   dropped. Capacity limits are kept as counts, so behaviour that depends on
   them (when a page splits, when the scrollback is trimmed) is unchanged.
3. **Every upstream test is translated**, one Go test per Zig test with the
   same name and the same assertions. The only tests left out are those that
   check memory layout alone, each listed with its reason and approved. No
   test is skipped or weakened; a disputed test goes to the project lead.
   Every edge is tested as well: the smallest and largest valid value, one
   past each limit, empty input, every screen edge and margin.
4. **The recorded corpus is the judge.** About 190,000 cases recorded from
   the pinned upstream library compare the terminal's full state, byte for
   byte, with what upstream produced, also with the same input fed in pieces
   at every byte boundary. A case that fails upstream's own integrity checks
   in a Debug build is never recorded as an expected result. The port
   reproduces upstream's defects on purpose; deliberate fixes come after
   everything passes and are listed separately.
5. **Standard library only.** No cgo and no dependencies. Where the Go
   standard library has a direct equivalent of a component, it is used
   behind a wrapper that keeps upstream's name, and upstream's tests for that
   component run against the wrapper; if they fail, the difference goes to
   the project lead before anything changes. The parser, the stream, the
   terminal, the screen, the page list and the style set are never swapped
   for a library equivalent. Unicode width and grapheme tables are
   generated from the Unicode version Go ships, and every upstream test
   whose expectation changes with Go's Unicode version is listed.
6. **Errors are upstream's errors.** Each Zig error is one Go sentinel with
   the same name, returned bare and never wrapped, so `err.Error()` is Zig's
   error name.
7. **No SIMD.** Upstream's scalar path is the reference; its SIMD fast paths
   are not ported.
8. **One pinned upstream.** Tag v1.3.1 and nothing from upstream's
   development branch; re-pinning is the owner's decision alone.
9. **Proven on 32-bit ARM.** The port was signed off with the tests run on
   the phone it was made for. Every test run compiles for 32-bit ARM and
   checks that no multiply-add is fused; a change that could behave
   differently on a 32-bit processor (arithmetic on plain `int` or `uint`, a
   changed integer width) runs its tests on the phone once more. Nothing is
   run under emulation.
10. **The public API mirrors upstream's.** Package `vt` has the top-level
    names of upstream's `src/lib_vt.zig`, generated from the port as aliases
    and thin wrappers; nothing is invented. Drawing, the shell and its
    pseudo-terminal, and the answers to programs' queries belong to the
    application, as they do upstream.
11. **Every change on its own branch**, reviewed by the other of the two
    developers, who reads the diff against the Zig source and reruns every
    test and every recorded case (`docs/verification.md`, "Since the
    sign-off").

## Use

```go
import (
    "github.com/XIAZY/term51/libghostty-vt-go/vt"
    "github.com/XIAZY/term51/libghostty-vt-go/vt/input"
)

term, err := vt.NewTerminal(vt.TerminalOptions{
    Cols: 80, Rows: 24,
    MaxScrollback: 1 << 20,
    Colors:        vt.TerminalColorsDefault(),
    DefaultModes:  vt.NewModePacked(),
})
stream := term.VtStream()
stream.NextSlice(outputOfTheProgram)

rs := vt.RenderStateEmpty()
rs.Update(&term)                 // a snapshot of the screen, to draw from

input.EncodeKey(w, keyEvent, input.KeyEncodeOptionsFromTerminal(&term))
```

`term.VtStream()` applies everything a program writes and ignores the
queries it sends ("where is the cursor", "what terminal are you"). To
answer them, give `vt.NewStream` your own handler that wraps
`vt.NewReadonlyHandler`; see `ExampleNewStream`.

`vt/example_test.go` has complete, tested examples: reading the screen as
text and cell by cell, key encoding, and a stream handler that answers a
program's query.

## Packages

- `vt`: the public API, the top-level names of upstream's `src/lib_vt.zig`
  (`Terminal`, `Stream`, `Screen`, `RenderState`, `Selection`, `Style`, ...).
- `vt/osc`, `vt/sgr`, `vt/modes`, `vt/color`, `vt/page`, `vt/search`,
  `vt/formatter`, `vt/input` and the others: upstream's namespaces.
- `terminal`: the port itself. The `vt` packages are generated from it as
  aliases and thin wrappers (`scripts/genvt`), so a `vt.Terminal` is a
  `terminal.Terminal`.

`docs/public-api.md` says which types have exported fields, what happens on
wrong arguments, and what remains of allocators and `Deinit`.

## How the port is kept honest

- `docs/mapping.md`: how each Zig construct is written in Go, and every
  place the port has to differ.
- `docs/verification.md`: what was run to sign the port off, and how
  changes are tested since.
- `docs/post-port-optimizations.md`: the speed work, with measurements
  against upstream's plain (non-SIMD) build on the same machine.
- `docs/go-native-revisit.md`: what was made Go-shaped after the port, and
  what was kept as upstream has it, with the reasons.
- Upstream's own defects are copied on purpose and listed in the
  reference-tests repository (`docs/deliberate-fixes.md` there).

## Speed

Through a real terminal the port is 1.5 to 2.2 times slower than upstream's
plain (non-SIMD) build on the same machine; the measurements and what was
done are in `docs/post-port-optimizations.md`. The cheapest speed left is
the application's to take: building the final program with Go's
profile-guided optimisation gave 30% on plain text in a trial, with no code
change.

## Tests

    scripts/test.sh

runs `go vet` and every test, and compiles for 32-bit ARM. The tests
include the recorded cases, which live in the reference-tests repository
(the pinned upstream source, the Zig harness and about 7 GB of recorded
state; it is kept on the project's machine as `term51` and is not
published): point `LIBGHOSTTY_VT_CORPUS` at a checkout (default: a checkout
named `term51` beside the repository this library is part of). Without it
the corpus tests fail; they never skip. The default run samples the
recorded cases; `TERM51_FULL=1` runs all of them.
