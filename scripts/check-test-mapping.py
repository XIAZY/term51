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

CROSS_FILES = {"Screen": ["rollback_test.go"], "PageList": ["rollback_test.go"], "Terminal": ["rollback_test.go"]}


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


def not_translated():
    """scripts/not-translated.txt: '<zig file> | <title> | <reason>' lines."""
    out = {}
    p = os.path.join(ROOT, "scripts", "not-translated.txt")
    if os.path.exists(p):
        for line in open(p):
            if line.strip() and not line.startswith("#"):
                f, title, reason = [x.strip() for x in line.split("|", 2)]
                out.setdefault(f, {})[title] = reason
    return out


def main():
    zig_dir = sys.argv[1]
    skipped = not_translated()
    ok = True
    total = 0
    for rel in sys.argv[2:]:
        base = os.path.splitext(os.path.basename(rel))[0]
        pkg = os.path.join(ROOT, os.path.dirname(rel))
        # A Zig file in a subdirectory (terminal/kitty/color.zig) lives in the
        # same Go package as its parent, named <subdir>_<file> (kitty_color).
        parts = rel.split("/")
        if len(parts) > 2:
            base = "_".join(parts[1:-1] + [os.path.splitext(parts[-1])[0]])
            pkg = os.path.join(ROOT, parts[0])
        # src/input/<name>.zig lives in the same Go package, files input_<name>.
        if len(parts) == 2 and parts[0] == "input":
            base = "input_" + os.path.splitext(parts[1])[0]
            pkg = os.path.join(ROOT, "terminal")
        # src/datastruct/<name>.zig, src/os/<name>.zig and src/unicode/<name>.zig:
        # Go files datastruct_<name>, os_<name> and unicode_<name>.
        if len(parts) == 2 and parts[0] in ("datastruct", "os", "unicode"):
            base = parts[0] + "_" + os.path.splitext(parts[1])[0]
            pkg = os.path.join(ROOT, "terminal")
        extra = []
        for f in glob.glob(os.path.join(pkg, base + "_*_test.go")):
            # osc_encoding_test.go belongs to osc_encoding.go, not to osc.go:
            # a test file whose stem names an existing Go file is not an
            # extra test file of this one (Parser_osc_test.go is: there is no
            # Parser_osc.go).
            stem = os.path.basename(f)[: -len("_test.go")]
            # Likewise a file whose stem starts with the name of a longer
            # existing Go file (input_key_encode_a_test.go is not an extra
            # file of input_key.go).
            longer = any(
                os.path.basename(g)[:-3].startswith(base + "_") and (stem == os.path.basename(g)[:-3] or stem.startswith(os.path.basename(g)[:-3] + "_"))
                for g in glob.glob(os.path.join(pkg, base + "_*.go")) if not g.endswith("_test.go")
            )
            if not os.path.exists(os.path.join(pkg, stem + ".go")) and not longer:
                extra.append(f)
        # Cross-cutting test files (not named after one Zig file): the rollback
        # tests cover functions of Screen.zig, PageList.zig and Terminal.zig.
        for cross in CROSS_FILES.get(base, []):
            extra.append(os.path.join(pkg, cross))
        go_files = sorted(glob.glob(os.path.join(pkg, base + "_test.go")) + extra)
        want = zig_tests(os.path.join(zig_dir, rel))
        got = go_tests(go_files)
        # A cross-cutting file holds tests of several Zig files: keep only the
        # ones that belong to this Zig file.
        cross_names = set(CROSS_FILES.get(base, []))
        got = [g for g in got if g[1] not in cross_names or g[0] in want]
        got_names = [g[0] for g in got]
        total += len(want)
        listed = skipped.get(rel, {})
        missing = [w for w in want if got_names.count(w) == 0 and w.strip('"') not in listed]
        n_listed = sum(1 for w in want if w.strip('"') in listed and got_names.count(w) == 0)
        dupes = sorted({g for g in got_names if got_names.count(g) > max(1, want.count(g))})
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
        print("%-28s zig=%d go=%d (%d in other test files, %d listed as not translated) %s" % (rel, len(want), len(got), tagged, n_listed, status))
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
