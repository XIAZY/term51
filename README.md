# libghostty-vt-go

A Go port of libghostty-vt, the terminal emulation core of
[Ghostty](https://github.com/ghostty-org/ghostty), ported from tag v1.3.1
(commit 332b2aef). MIT licensed, like the original.

**Status: early. Only the first parser files are ported.** See
`docs/port-status.md`.

The port is deliberately literal: one Go file per Zig file, one Go function
per Zig function, same names and control flow, so that it can be checked
against upstream side by side and verified with upstream's own tests.
`docs/mapping.md` records every place where Go has to differ.

## Tests

    scripts/test.sh

runs everything natively and as a 32-bit ARM build. The tests include a
recorded corpus: inputs with the exact output the pinned upstream library
produced. The corpus lives in the `term51` repository; point
`LIBGHOSTTY_VT_CORPUS` at a checkout (default: `../term51`).
