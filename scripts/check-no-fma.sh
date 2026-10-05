#!/bin/sh
# Fail if the compiled terminal package contains a fused multiply-add.
#
# Zig rounds every float operation separately; Go may fuse x*y+z into one
# instruction (also across statements) unless the product is explicitly
# converted, and does so on arm64 and on amd64 with GOAMD64=v3. A fused
# instruction changes the last bit of results such as LAB.fromRgb's
# pow(x, 2.4) (docs/mapping.md, "Floating point"). Instruction selection can
# change with a Go upgrade even when the source does not, so this checks the
# machine code, not the source. 32-bit ARM is not checked: Go emits no
# fused instruction there.
set -eu
cd "$(dirname "$0")/.."
GO=${GO:-/usr/local/go/bin/go}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
status=0
#
# Both the package and its test binary are checked: a generic function is
# only compiled where it is instantiated, and some (RefCountedSetLayoutInit,
# for instance) are instantiated only by the tests until the files that use
# them are ported.
check() {
    name=$1
    pattern=$2
    shift 2
    env "$@" "$GO" build -o "$tmp/$name.a" ./terminal
    env "$@" "$GO" test -c -o "$tmp/$name.test" ./terminal
    if { "$GO" tool objdump -s 'terminal\.' "$tmp/$name.a"; "$GO" tool objdump -s 'terminal\.' "$tmp/$name.test"; } | grep -E "$pattern" > "$tmp/$name.txt"; then
        echo "fused multiply-add instructions in the $name build:"
        awk '{print "  " $1, $4}' "$tmp/$name.txt"
        status=1
    else
        echo "$name: no fused multiply-add"
    fi
}
check arm64 '\bF(N)?M(ADD|SUB)(S|D)\b' GOARCH=arm64
check amd64-v3 '\bVF(N)?M(ADD|SUB)' GOARCH=amd64 GOAMD64=v3
exit $status
