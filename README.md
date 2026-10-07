# libghostty-vt-go

A Go port of libghostty-vt, the terminal emulation core of
[Ghostty](https://github.com/ghostty-org/ghostty), from tag v1.3.1 (commit
332b2aef). MIT licensed, like the original.

It is a library, not a terminal: it parses what programs write, keeps the
screen, scrollback, modes and colours, reflows on resize, and encodes keys
and pastes. Drawing, the shell and its pseudo-terminal, and the answers to
programs' queries belong to the application, as they do with upstream.

**The rule of the port:** the same features and behaviour as upstream, in
the form a Go user expects. Every upstream test is translated, and about
190,000 recorded cases compare the terminal's full state, byte for byte,
with what the pinned upstream library produced. Pure Go, no cgo, no
dependencies; it builds for any platform Go does.

## Use

```go
import (
    "github.com/XIAZY/libghostty-vt-go/vt"
    "github.com/XIAZY/libghostty-vt-go/vt/input"
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
- Upstream's own defects are copied on purpose and listed in the `term51`
  repository (`docs/deliberate-fixes.md`).

## Tests

    scripts/test.sh

runs `go vet` and every test, and compiles for 32-bit ARM. The tests
include the recorded cases, which live in the `term51` repository: point
`LIBGHOSTTY_VT_CORPUS` at a checkout (default: `../term51`). The default run
samples the recorded cases; `TERM51_FULL=1` runs all of them.
