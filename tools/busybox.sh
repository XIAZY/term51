#!/bin/sh
# Fetches the BusyBox that goes into the package: the file for BlackBerry 10
# of the busybox-qnx release named in tools/busybox.lock, checked against
# the sha256 written there, to build/busybox/busybox. A file that is there
# already and has that checksum is kept, so only the first build downloads.
# tools/build.sh calls this before it packages.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
lock=$repo/tools/busybox.lock
value() { sed -n "s/^$1 = //p" "$lock"; }
tag=$(value tag)
file=$(value file)
sum=$(value sha256)
out=$repo/build/busybox/busybox

matches() { [ -f "$1" ] && [ "$(sha256sum "$1" | cut -d' ' -f1)" = "$sum" ]; }

if ! matches "$out"; then
    mkdir -p "$(dirname "$out")"
    curl -fsSL -o "$out.part" "https://github.com/XIAZY/busybox-qnx/releases/download/$tag/$file"
    matches "$out.part" || {
        echo "error: $file of $tag does not have the sha256 in tools/busybox.lock" >&2
        rm -f "$out.part"
        exit 1
    }
    mv "$out.part" "$out"
fi
chmod 755 "$out"
echo "busybox: $file of $tag"
