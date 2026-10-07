#!/bin/sh
# Builds build/io.github.xiazy.term51.bar. Runs inside the BB10 builder image
# (blackberry10-toolchain) with the qnx/arm Go toolchain (go-qnx) on PATH;
# tools/build.sh calls it.
set -eu
cd "$(dirname "$0")/.."

export GOOS=qnx GOARCH=arm GOARM=7
export CGO_ENABLED=1 CC=bb10-cc CXX=bb10-c++
# A developer's go.work names host paths; the build uses go.mod's replace.
export GOWORK=off GOFLAGS=-mod=mod

mkdir -p build
/opt/bb10-qt4/bin/moc -nw -o internal/ui/moc_bridge.cpp internal/ui/bridge.h
# go build skips linking when the old binary looks up to date, even if the
# external linker (bb10-cc) changed, so always relink.
rm -f build/term51
go build -trimpath -buildvcs=false -ldflags="-s -w" -o build/term51 .
blackberry-nativepackager -devMode -package build/io.github.xiazy.term51.bar \
    bar-descriptor.xml -configuration Device-Debug 2>&1 | grep -v -e "^WARNING" -e JAVA_TOOL || true
test -f build/io.github.xiazy.term51.bar
echo "packaged: build/io.github.xiazy.term51.bar"
