#!/bin/sh
# Development build: runs tools/build-inside.sh in the BB10 builder image with
# a go-qnx toolchain (GOQNX, built with src/make.bash). The terminal library
# is part of this repository (libghostty-vt-go/, where go.mod's replace
# directive points), so nothing else is mounted.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
goqnx=${GOQNX:?set GOQNX to a go-qnx checkout built with src/make.bash}
image=${BB10_BUILDER_IMAGE:-ghcr.io/xiazy/blackberry10-toolchain:latest}
[ -x "$goqnx/bin/go" ] || { echo "error: no Go toolchain in $goqnx (run src/make.bash there)" >&2; exit 1; }
mkdir -p "$repo/build"
# The BusyBox that the package carries (tools/busybox.lock).
"$repo/tools/busybox.sh"
# And the Dropbear SSH client (tools/dropbear.lock).
"$repo/tools/dropbear.sh"

exec docker run --rm --user "$(id -u):$(id -g)" \
    -e HOME=/tmp/home -e GOCACHE=/src/build/go-cache -e GOMODCACHE=/src/build/go-mod \
    -e PATH=/opt/go-qnx/bin:/opt/bb10-toolchain/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    -v "$goqnx:/opt/go-qnx:ro" -v "$repo:/src" -w /src "$image" tools/build-inside.sh
