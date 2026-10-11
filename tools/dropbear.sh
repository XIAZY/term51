#!/bin/sh
# Builds the Dropbear SSH client that goes into the package: the file for
# BlackBerry 10 from the repository dropbear-qnx at the commit named in
# tools/dropbear.lock, in the toolchain image named there, checked against
# the sha256 written there, to build/dropbear/dropbearmulti. A file that is
# there already and has that checksum is kept, so only the first build
# compiles. tools/build.sh calls this before it packages.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
common=$(cd "$(git -C "$repo" rev-parse --git-common-dir)/.." && pwd)
lock=$repo/tools/dropbear.lock
value() { sed -n "s/^$1 = //p" "$lock"; }
commit=$(value commit)
image=$(value image)
sum=$(value sha256)
out=$repo/build/dropbear/dropbearmulti
source=$(cd "${DROPBEAR_QNX:-$common/../dropbear-qnx}" && pwd)

matches() { [ -f "$1" ] && [ "$(sha256sum "$1" | cut -d' ' -f1)" = "$sum" ]; }

if ! matches "$out"; then
    # That commit's files, and nothing of the repository's working tree.
    work=$repo/build/dropbear-src
    rm -rf "$work"
    mkdir -p "$work" "$(dirname "$out")"
    git -C "$source" archive "$commit" | tar -x -C "$work"
    docker run --rm --user "$(id -u):$(id -g)" -e HOME=/src/build-qnx -v "$work:/src" -w /src "$image" \
        qnx/build.sh bb10 > "$repo/build/dropbear-build.log" 2>&1 || {
        tail -20 "$repo/build/dropbear-build.log" >&2
        exit 1
    }
    matches "$work/dropbearmulti" || {
        echo "error: the Dropbear built from $commit does not have the sha256 in tools/dropbear.lock" >&2
        exit 1
    }
    cp "$work/dropbearmulti" "$out"
    rm -rf "$work"
fi
chmod 755 "$out"
echo "dropbear: $(echo "$commit" | cut -c1-9) of dropbear-qnx"
