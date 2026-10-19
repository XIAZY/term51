#!/bin/sh
# A run check of the QNX build of Dropbear's client programs, for the
# device it was built for.
#
#   sh check.sh /path/to/dropbearmulti [USER@HOST [PORT]]
#
# Run it in a directory that can be written to and can hold symbolic links;
# it works in a subdirectory "w" there and removes it. The programs' home is
# that directory too, so nothing is written to the user's ~/.ssh.
#
# Without a host it checks what needs no server: the four programs by their
# names, key generation, the public half, conversion to an OpenSSH key file
# and back. With USER@HOST it also logs in to that server with the key in
# CHECK_KEY (a Dropbear key file the server accepts; the script never adds
# a key anywhere): a command, the OpenSSH-style options, scp both ways.
# RSA keys are made with CHECK_BITS bits (2048); an emulated machine may
# want fewer to be done in reasonable time.
#
# Only the shell and the programs under test are used, since a QNX system
# may have little else. Each line is "ok", "FAIL" with what came out, or
# "skip".
M=$1
TARGET=${2:-}
PORT=${3:-22}
BITS=${CHECK_BITS:-2048}
D=$(pwd)/w
pass=0; fail=0; skip=0
ok() { pass=$((pass+1)); echo "ok   $1"; }
bad() { fail=$((fail+1)); echo "FAIL $1: $2"; }
has() { case $1 in *"$2"*) return 0;; esac; return 1; } # whether $1 contains $2

rm -rf "$D"; mkdir "$D" || exit 1
cd "$D" || exit 1
HOME=$D; export HOME
for p in dbclient scp dropbearkey dropbearconvert; do ln -s "$M" $p || exit 1; done

out=$(./dbclient -V 2>&1)
if has "$out" "Dropbear v"; then ok "dbclient: $out"; else bad dbclient "$out"; fi
out=$(./scp 2>&1)
if has "$out" "usage: scp"; then ok scp; else bad scp "$out"; fi
out=$(./dropbearconvert 2>&1)
if has "$out" "Usage"; then ok dropbearconvert; else bad dropbearconvert "$out"; fi

for kind in rsa ed25519 ecdsa; do
    if [ $kind = rsa ]; then out=$(./dropbearkey -t rsa -s "$BITS" -f k_$kind 2>&1)
    else out=$(./dropbearkey -t $kind -f k_$kind 2>&1); fi
    pub=$(./dropbearkey -y -f k_$kind 2>&1)
    if has "$out" "Fingerprint" && has "$pub" "Public key portion"; then ok "dropbearkey $kind"
    else bad "dropbearkey $kind" "$out"; fi
done

# To an OpenSSH key file and back: the public half must stay the same.
want=$(./dropbearkey -y -f k_rsa 2>&1)
./dropbearconvert dropbear openssh k_rsa k_rsa.pem >/dev/null 2>&1
read -r first < k_rsa.pem
./dropbearconvert openssh dropbear k_rsa.pem k_rsa.back >/dev/null 2>&1
got=$(./dropbearkey -y -f k_rsa.back 2>&1)
if has "$first" "PRIVATE KEY" && [ "$want" = "$got" ]; then ok "dropbearconvert there and back"
else bad "dropbearconvert there and back" "$first"; fi

# With no way to log in, the client must stop with a refusal, not hang.
out=$(./dbclient -y -o BatchMode=yes -i k_ed25519 -p 1 nobody@127.0.0.1 true 2>&1)
if has "$out" "onnect" || has "$out" "xited"; then ok "dbclient to a closed port stops"
else bad "dbclient to a closed port" "$out"; fi

if [ -n "$TARGET" ] && [ -n "${CHECK_KEY:-}" ]; then
    user=${TARGET%@*}; host=${TARGET#*@}
    out=$(./dbclient -y -i "$CHECK_KEY" -p "$PORT" "$TARGET" 'echo logged-in' 2>/dev/null)
    if [ "$out" = logged-in ]; then ok "net: a command on $host"; else bad "net: a command" "$out"; fi
    out=$(./dbclient -y -l "$user" -p "$PORT" -i "$CHECK_KEY" -o BatchMode=yes -o NoSuchOption=1 "$host" 'echo options' 2>/dev/null)
    if [ "$out" = options ]; then ok "net: -l, -p, -o, and an unknown -o is passed over"; else bad "net: options" "$out"; fi
    echo "payload $$" > up.txt
    remote=/tmp/dropbear-check-$$.txt
    ./scp -P "$PORT" -S "$D/dbclient" -i "$CHECK_KEY" up.txt "$TARGET:$remote" >/dev/null 2>&1
    ./scp -P "$PORT" -S "$D/dbclient" -i "$CHECK_KEY" "$TARGET:$remote" back.txt >/dev/null 2>&1
    ./dbclient -y -i "$CHECK_KEY" -p "$PORT" "$TARGET" "rm -f $remote" >/dev/null 2>&1
    if [ -f back.txt ] && [ "$(cat up.txt)" = "$(cat back.txt)" ]; then ok "net: scp there and back"
    else bad "net: scp there and back" "no file came back"; fi
else
    for n in "a command" "options" "scp there and back"; do skip=$((skip+1)); echo "skip net: $n: run with USER@HOST and CHECK_KEY"; done
fi

cd ..; rm -rf "$D"
echo "passed $pass, failed $fail, skipped $skip"
[ $fail = 0 ]
