#!/bin/sh
# Builds Dropbear's client programs for QNX from a clean checkout:
# ./dropbearmulti, one stripped program that is dbclient, scp, dropbearkey
# and dropbearconvert, by the name it is called by. No server is built.
#
#   qnx/build.sh [bb10|qnx65]
#
# bb10 (the default) is BlackBerry 10, 32-bit ARM; qnx65 is QNX 6.5 on x86.
# Both are built from the same source and options (qnx/options.h),
# each against its own system's SDK, with the toolchain that must be on the
# PATH: the C compiler driver for the target (bb10-cc or qnx65-cc) and
# binutils (GNU for ARM, LLVM's for x86); besides make. See qnx/README.md.
set -eu

target=${1:-bb10}
cd "$(dirname "$0")/.."
src=$(pwd)
qnx=$src/qnx

case $target in
bb10)
    host=arm-unknown-nto-qnx8.0.0eabi
    prefix=arm-unknown-nto-qnx6.5.0eabi-
    cc=bb10-cc
    ar=${prefix}ar ranlib=${prefix}ranlib strip=${prefix}strip readelf=${prefix}readelf
    sdk=${QNX_TARGET:-/opt/bb10-sdk/target/qnx6}/armle-v7
    libs="$sdk/lib/libc.so.3 $sdk/lib/libm.so.2 $sdk/lib/libsocket.so.3 $sdk/usr/lib/libbps.so.3"
    ;;
qnx65)
    host=i486-pc-nto-qnx6.5.0
    # The driver's own defaults, said here so that the file never needs
    # more than a 486 whatever the driver comes to default to.
    cc="qnx65-cc -march=i486 -mno-sse"
    ar=llvm-ar ranlib=llvm-ranlib strip=llvm-strip readelf=llvm-readelf
    sdk=${QNX65_SDK:-/opt/qnx65-sdk}/qnx6/x86
    libs="$sdk/lib/libc.so.3 $sdk/lib/libm.so.2 $sdk/lib/libsocket.so.3"
    ;;
*)
    echo "usage: qnx/build.sh [bb10|qnx65]" >&2
    exit 2
    ;;
esac

for tool in ${cc%% *} $ar $strip $readelf make; do
    command -v "$tool" >/dev/null || { echo "error: $tool is not on the PATH (see qnx/README.md)" >&2; exit 1; }
done

# A build folder of its own for each target, made anew every time.
build=$src/build-qnx/$target
rm -rf "$build"
mkdir -p "$build"
cd "$build"
cp "$qnx/options.h" localoptions.h

# No zlib (QNX 6.5's SDK has none), no syslog (BlackBerry 10 has no
# <syslog.h>), no login records, and Dropbear's own copy of the
# libtomcrypt and libtommath it needs. The hardening flags are left to the
# compiler drivers, which know what each system's loader takes. The socket
# functions are in libsocket on QNX.
"$src/configure" --host=$host CC="$cc" AR=$ar RANLIB=$ranlib STRIP=$strip LIBS=-lsocket \
    --disable-zlib --disable-syslog --disable-harden --enable-bundled-libtom --disable-shadow \
    --disable-utmp --disable-utmpx --disable-wtmp --disable-wtmpx --disable-lastlog \
    --disable-loginfunc --disable-pututline > configure.log 2>&1 || { tail -20 configure.log >&2; exit 1; }
jobs=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2)
make PROGRAMS="dbclient dropbearkey dropbearconvert scp" MULTI=1 SCPPROGRESS=1 -j"$jobs"
$strip -o "$src/dropbearmulti" dropbearmulti
cd "$src"

# The program may ask the loader only for names that its system's own
# libraries have (qnx/check-names.sh).
# shellcheck disable=SC2086
READELF=$readelf "$qnx/check-names.sh" dropbearmulti $libs
ls -l dropbearmulti
