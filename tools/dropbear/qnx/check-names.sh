#!/bin/sh
# Copyright (c) 2026 Zhongyang Xia
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the
# "Software"), to deal in the Software without restriction, including
# without limitation the rights to use, copy, modify, merge, publish,
# distribute, sublicense, and/or sell copies of the Software, and to
# permit persons to whom the Software is furnished to do so, subject to
# the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
# OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
# IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
# CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
# TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
# SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

# Checks that a program asks the loader only for names that the system it
# is meant for has.
#
#   qnx/check-names.sh PROGRAM ALLOWED...
#
# Each ALLOWED is a shared library, or a text file with one name a line.
# The program's needs are its undefined dynamic symbols (weak ones aside,
# which the loader may leave unresolved) and the
# data symbols it copies from a library (COPY relocations), which a list of
# undefined symbols does not show. Any name outside the allowed ones is
# printed, and the exit status is 1: the program would not load.
#
# A program linked against BlackBerry 10's libraries can pick up names
# that QNX 6.5 lacks without anyone noticing; this is where it shows.
# READELF names the readelf to use (default: readelf).
set -eu

readelf=${READELF:-readelf}
program=$1
shift

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# The dynamic symbols of an ELF file: "UND name" or "DEF name" a line.
symbols() {
    # The name is the last column and the section the one before it; the
    # type column can be more than one word ("<OS specific>: 10").
    "$readelf" --dyn-syms -W "$1" | awk '
        $1 ~ /^[0-9]+:$/ && NF >= 8 {
            name = $NF; sub(/@.*/, "", name)
            # A weak undefined symbol may stay unresolved at load time:
            # crtbegin.o of gcc has three (_Jv_RegisterClasses and the
            # frame-info pair).
            print ($(NF-1) != "UND" ? "DEF" : $(NF-3) == "WEAK" ? "WEAK" : "UND"), name
        }'
}

for allowed in "$@"; do
    if [ "$(head -c 4 "$allowed" | od -An -c | tr -d ' ')" = '177ELF' ]; then
        symbols "$allowed" | awk '$1 == "DEF" { print $2 }'
    else
        cat "$allowed"
    fi
done | sort -u > "$work/allowed"

{
    symbols "$program" | awk '$1 == "UND" { print $2 }'
    "$readelf" -r -W "$program" | awk '$3 ~ /_COPY$/ { name = $5; sub(/@.*/, "", name); print name }'
} | sort -u > "$work/needed"

comm -23 "$work/needed" "$work/allowed" > "$work/missing"
if [ -s "$work/missing" ]; then
    echo "error: $program needs names that are not allowed:" >&2
    sed 's/^/    /' "$work/missing" >&2
    exit 1
fi
echo "$program: $(wc -l < "$work/needed" | tr -d ' ') names needed, all allowed"
