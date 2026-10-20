#!/bin/sh
# Fetches Go with the QNX port for the machine this runs on: the file of the
# go-qnx release named in tools/go-qnx.lock, checked against the sha256
# written there, unpacked to build/go-qnx/<sha256>/go, whose path it prints.
# A folder that is there already is kept, so only the first build
# downloads. tools/build.sh uses it when GOQNX is not set.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
lock=$repo/tools/go-qnx.lock
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*)
    echo "error: tools/go-qnx.lock has no Go for $(uname -m)" >&2
    exit 1
    ;;
esac
tag=$(sed -n "s/^tag = //p" "$lock")
line=$(sed -n "s/^$arch = //p" "$lock")
file=${line% *}
sum=${line#* }
dir=$repo/build/go-qnx/$sum

if [ ! -x "$dir/go/bin/go" ]; then
    rm -rf "$dir"
    mkdir -p "$dir"
    curl -fsSL -o "$dir/$file" "https://github.com/XIAZY/go-qnx/releases/download/$tag/$file"
    [ "$(sha256sum "$dir/$file" | cut -d' ' -f1)" = "$sum" ] || {
        echo "error: $file of $tag does not have the sha256 in tools/go-qnx.lock" >&2
        rm -rf "$dir"
        exit 1
    }
    tar -xzf "$dir/$file" -C "$dir"
    rm "$dir/$file"
fi
echo "$dir/go"
