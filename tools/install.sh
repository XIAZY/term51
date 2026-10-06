#!/bin/sh
# Installs term51 on a phone rooted with bb10mt, using the phone's own
# package installer over SSH: no Development Mode password, no Java and no
# BlackBerry SDK on the host, only ssh and scp.
#
# usage: tools/install.sh <ssh-host> [package.bar]
#   <ssh-host> is anything ssh accepts (root@<ip>, or a Host from
#   ~/.ssh/config) and must reach a root shell on the phone. The package
#   defaults to the one tools/build.sh writes.
#
# The package is copied to the phone's SD card (REMOTE_DIR), not its internal
# storage, and removed after a successful install.
set -eu

host=${1:?usage: $0 <ssh-host> [package.bar]}
bar=${2:-$(cd "$(dirname "$0")/.." && pwd)/build/io.github.xiazy.term51.bar}
[ -f "$bar" ] || { echo "error: no such package: $bar" >&2; exit 1; }

remote_dir=${REMOTE_DIR:-/accounts/1000/removable/sdcard/term51}
remote="$remote_dir/$(basename "$bar")"
scp -q "$bar" "$host:$remote"

# sud_install_package_2 comes from the phone's /base/scripts/sudtools.sh. The
# phone's sh lacks many utilities, so run it under ksh, and judge success by
# its log line (spelt that way) rather than its exit status. The first install
# after a reboot can take minutes. The BAR is only removed once the installer
# is done with it; deleting it under a slow install breaks that install.
output=$(ssh "$host" "/bin/ksh -c '. /base/scripts/sudtools.sh; sud_install_package_2 -T 600 -D -p \"$remote\"'" 2>&1) || true
if printf '%s\n' "$output" | grep -q "Sucessfully installed"; then
    ssh "$host" "rm -f \"$remote\""
    echo "installed: $(basename "$bar") on $host"
else
    printf '%s\n' "$output" >&2
    echo "error: installation failed; $remote is left on the phone" >&2
    exit 1
fi
