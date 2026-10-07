#!/bin/sh
# Run every test on this machine, and check that the code compiles for
# 32-bit ARM (the BlackBerry 10 phone the port was first made for).
#
# The recorded cases are sampled by default; TERM51_FULL=1 runs all of them.
# Nothing is run under emulation: results on 32-bit were proven identical on
# the phone when the port was signed off (docs/verification.md), and a
# change is run there again only when it could behave differently on a
# 32-bit processor (arithmetic on plain int or uint, a changed integer
# width).
set -eu
cd "$(dirname "$0")/.."
GO=${GO:-/usr/local/go/bin/go}
echo "== no fused multiply-add (arm64, amd64 v3)"
GO="$GO" sh scripts/check-no-fma.sh
echo "== native"
"$GO" vet ./...
"$GO" test ./... -count=1 "$@"
echo "== linux/arm (32-bit, GOARM=7): compile only"
GOARCH=arm GOARM=7 "$GO" vet ./...
