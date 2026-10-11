#!/bin/sh
# Fetches the PowerShell that goes into the package: the archive for
# BlackBerry 10 of the powershell-qnx release named in tools/powershell.lock,
# checked against the sha256 written there, and unpacked as it is to
# build/powershell/. The archive is kept under build/, so only the first
# build downloads, and it is unpacked again only when the lock names
# another one. tools/build.sh calls this before it packages.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
lock=$repo/tools/powershell.lock
value() { sed -n "s/^$1 = //p" "$lock"; }
tag=$(value tag)
file=$(value file)
sum=$(value sha256)
archive=$repo/build/$file
tree=$repo/build/powershell
stamp=$repo/build/powershell.sha256

matches() { [ -f "$1" ] && [ "$(sha256sum "$1" | cut -d' ' -f1)" = "$sum" ]; }

if ! matches "$archive"; then
    mkdir -p "$repo/build"
    curl -fsSL -o "$archive.part" "https://github.com/XIAZY/powershell-qnx/releases/download/$tag/$file"
    matches "$archive.part" || {
        echo "error: $file of $tag does not have the sha256 in tools/powershell.lock" >&2
        rm -f "$archive.part"
        exit 1
    }
    mv "$archive.part" "$archive"
fi
if [ "$(cat "$stamp" 2>/dev/null)" != "$sum" ] || [ ! -x "$tree/pwsh/pwsh" ]; then
    rm -rf "$tree" "$stamp"
    mkdir -p "$tree"
    tar -xzf "$archive" -C "$tree"
    echo "$sum" > "$stamp"
fi
echo "powershell: $file of $tag"
