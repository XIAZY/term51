#!/bin/sh
# Run every test natively and as a 32-bit ARM build (the target phone is
# 32-bit ARM). The 32-bit run uses qemu user-mode emulation.
set -eu
cd "$(dirname "$0")/.."
GO=${GO:-/usr/local/go/bin/go}
echo "== native"
"$GO" vet ./...
"$GO" test ./... -count=1 "$@"
echo "== linux/arm (32-bit, GOARM=7)"
GOARCH=arm GOARM=7 "$GO" vet ./...
GOARCH=arm GOARM=7 "$GO" test -exec "${QEMU_ARM:-/usr/bin/qemu-arm-static}" ./... -count=1 "$@"
