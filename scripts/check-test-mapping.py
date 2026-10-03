#!/usr/bin/env python3
"""Check that the Go tests map one to one onto the Zig tests.

  scripts/check-test-mapping.py <zig src dir> terminal/Parser.zig ...

For each Zig file given (relative to the Zig source directory) every `test`
in it must appear exactly once, in the same order, as a `// Zig: test ...`
comment directly above a Go test function in the corresponding Go test
files (<File>_test.go and <File>_*_test.go).
"""

import glob
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def zig_tests(path):
    out = []
    for i, line in enumerate(open(path).read().split("\n"), 1):
        m = re.match(r'^\s*test (?:"((?:[^"\\]|\\.)*)"|([\w.@"]+)) \{\s*$', line)
        if m:
            out.append('"%s"' % m.group(1) if m.group(1) is not None else m.group(2))
        elif re.match(r"^\s*test \{\s*$", line):
            out.append("(unnamed, %s:%d)" % (os.path.basename(path), i))
    return out


def go_tests(paths):
    out = []
    for p in paths:
        lines = open(p).read().split("\n")
        for i, line in enumerate(lines):
            m = re.match(r"^// Zig: test (.*)$", line)
            if not m:
                continue
            j = i + 1
            while j < len(lines) and lines[j].startswith("//"):
                j += 1
            if j >= len(lines) or not re.match(r"^func Test\w+\(t \*testing\.T\) \{", lines[j]):
                print("%s:%d: Zig comment is not followed by a test function" % (p, i + 1))
                sys.exit(1)
            out.append((m.group(1).strip(), os.path.basename(p)))
    return out


def main():
    zig_dir = sys.argv[1]
    ok = True
    total = 0
    for rel in sys.argv[2:]:
        base = os.path.splitext(os.path.basename(rel))[0]
        pkg = os.path.join(ROOT, os.path.dirname(rel))
        go_files = sorted(glob.glob(os.path.join(pkg, base + "_test.go")) + glob.glob(os.path.join(pkg, base + "_*_test.go")))
        want = zig_tests(os.path.join(zig_dir, rel))
        got = go_tests(go_files)
        got_names = [g[0] for g in got]
        total += len(want)
        missing = [w for w in want if got_names.count(w) == 0]
        dupes = sorted({g for g in got_names if got_names.count(g) > 1})
        extra = [g for g in got_names if g not in want]
        # order is checked within each Go file
        order_ok = True
        for gf in {g[1] for g in got}:
            seq = [g[0] for g in got if g[1] == gf and g[0] in want]
            idx = [want.index(s) for s in seq]
            if idx != sorted(idx):
                order_ok = False
        tagged = sum(1 for g in got if g[1] != base + "_test.go")
        status = "ok" if not (missing or dupes or extra) and order_ok else "MISMATCH"
        print("%-28s zig=%d go=%d (%d in tagged files) %s" % (rel, len(want), len(got), tagged, status))
        for label, items in (("missing in Go", missing), ("duplicated in Go", dupes), ("not in Zig", extra)):
            for it in items:
                print("   %s: %s" % (label, it))
                ok = False
        if not order_ok:
            print("   order differs from the Zig file")
            ok = False
    print("total Zig tests: %d" % total)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
