#!/bin/sh
# Builds the Dropbear SSH client that goes into the package: the file for
# BlackBerry 10, from upstream Dropbear at the commit the submodule
# third_party/dropbear is pinned to, with term51's changes for QNX
# (tools/dropbear/qnx.patch, and the build settings and script in
# tools/dropbear/qnx/), in the toolchain image named in tools/dropbear.lock,
# checked against the sha256 written there, to build/dropbear/dropbearmulti.
# A file that is there already and has that checksum is kept, so only the
# first build compiles. tools/build.sh calls this before it packages.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
lock=$repo/tools/dropbear.lock
value() { sed -n "s/^$1 = //p" "$lock"; }
image=$(value image)
sum=$(value sha256)
out=$repo/build/dropbear/dropbearmulti

matches() { [ -f "$1" ] && [ "$(sha256sum "$1" | cut -d' ' -f1)" = "$sum" ]; }

if ! matches "$out"; then
    # A clone made without --recursive has the folder but not its files.
    if [ -z "$(ls -A "$repo/third_party/dropbear")" ]; then
        git -C "$repo" submodule update --init third_party/dropbear
    fi
    # Upstream at the pinned commit, and nothing of the submodule's working
    # folder; then the changes and the settings.
    pinned=$(git -C "$repo" rev-parse HEAD:third_party/dropbear)
    work=$repo/build/dropbear-src
    rm -rf "$work"
    mkdir -p "$work" "$(dirname "$out")"
    git -C "$repo/third_party/dropbear" archive "$pinned" | tar -x -C "$work"
    # The copy is inside this repository's build folder: git apply takes
    # the patch's paths from the repository's top, so it is told where.
    git -C "$repo" apply --check --directory=build/dropbear-src tools/dropbear/qnx.patch
    git -C "$repo" apply --directory=build/dropbear-src tools/dropbear/qnx.patch
    cp -R "$repo/tools/dropbear/qnx" "$work/qnx"
    docker run --rm --user "$(id -u):$(id -g)" -e HOME=/src/build-qnx -v "$work:/src" -w /src "$image" \
        qnx/build.sh bb10 > "$repo/build/dropbear-build.log" 2>&1 || {
        tail -20 "$repo/build/dropbear-build.log" >&2
        exit 1
    }
    matches "$work/dropbearmulti" || {
        echo "error: the Dropbear built from $(echo "$pinned" | cut -c1-12) with tools/dropbear/qnx.patch does not have the sha256 in tools/dropbear.lock" >&2
        exit 1
    }
    cp "$work/dropbearmulti" "$out"
    rm -rf "$work"
fi
chmod 755 "$out"
echo "dropbear: upstream $(git -C "$repo" rev-parse --short HEAD:third_party/dropbear) with tools/dropbear/qnx.patch"
