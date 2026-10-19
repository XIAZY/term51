# Dropbear's SSH client for QNX 6.5 and BlackBerry 10

This is Dropbear 2026.94 with the changes it needs to build for and run on
QNX: QNX 6.5.0 on x86, and BlackBerry 10 (QNX 8 on 32-bit ARM). **Only the
client programs are built**; there is no server in it.

The result is one program of about 250 KB, `dropbearmulti`, which is four
tools by the name it is called by:

| Name | What it is |
|---|---|
| `dbclient` | the SSH client; link it as `ssh` where a program expects that name |
| `scp` | copies files over SSH; it runs `dbclient`, which must be on the `PATH` under that name (or name it with `-S`) |
| `dropbearkey` | makes a key pair, and prints the public half of one (`-y`) |
| `dropbearconvert` | converts a key between Dropbear's file format and OpenSSH's |

    ln -s dropbearmulti dbclient    # and scp, dropbearkey, dropbearconvert
    ./dropbearkey -t rsa -s 2048 -f mykey
    ./dropbearkey -y -f mykey       # the line for the server's authorized_keys
    ./dbclient -i mykey user@host
    PATH=$PWD:$PATH ./scp -i mykey file user@host:

It is linked against the system's own libraries (`libc` and `libsocket`,
and `libm` and `libbps` on the phone) and uses Dropbear's own copy of the
cryptography.

Every change to Dropbear's source is in its compatibility layer
(`src/compat.c`, `src/compat.h`, `src/includes.h`). In term51 they are
`tools/dropbear/qnx.patch`, whose head says what each one is for; this
folder is `tools/dropbear/qnx/`.

## Building

In term51, `tools/dropbear.sh` does all of the following: it exports
upstream Dropbear at the commit the submodule `third_party/dropbear` is
pinned to, applies `tools/dropbear/qnx.patch`, adds this folder as `qnx/`,
and runs `qnx/build.sh bb10` in the image below. By hand, in such a tree:

    qnx/build.sh bb10       # BlackBerry 10, 32-bit ARM (the default)
    qnx/build.sh qnx65      # QNX 6.5 on x86

Each leaves the stripped program in `./dropbearmulti` and builds in a
folder of its own under `build-qnx/`, made anew every time. They need on
the `PATH`:

- for `bb10`: `bb10-cc`, the C compiler driver for BlackBerry 10, and the
  GNU binutils for `arm-unknown-nto-qnx6.5.0eabi`;
- for `qnx65`: `qnx65-cc`, the driver for QNX 6.5 (clang and lld, with QNX
  SDP 6.5.0's own headers and libraries), and LLVM's binutils (`llvm-ar`,
  `llvm-readelf` and the rest);
- `make`.

The container image `ghcr.io/xiazy/blackberry10-toolchain` has all of
them. From a clean checkout, with the version of the image that the
released files are built with:

    docker run --rm -v "$PWD:/src" -w /src --user "$(id -u):$(id -g)" \
        ghcr.io/xiazy/blackberry10-toolchain@sha256:60ce49510a37ed92368cca010bbaabf97b12d209b9fab179e8491b266bb2578f \
        qnx/build.sh qnx65

The same source built with the same image gives the same file, byte for
byte; term51's `tools/dropbear.lock` holds the BlackBerry 10 file's
checksum, and its build workflow builds that file on every push.

After each build `qnx/check-names.sh` confirms that the program asks the
loader only for names its system's own libraries have.

## The settings, and why

`qnx/options.h` is what differs from Dropbear's defaults:

- **The SHA-1 algorithms are on**: RSA signatures with SHA-1, the
  `diffie-hellman-group14-sha1` key exchange and `hmac-sha1`. Dropbear has
  them off by default now. QNX 6.5's own sshd is OpenSSH 5.2 of 2009 and
  knows nothing newer, so without them the client cannot log in to a QNX
  6.5 machine at all. Dropbear offers the modern algorithms first; the old
  ones are used only with a server that has nothing better.
- **`scp` finds `dbclient` on the `PATH`** instead of at Dropbear's fixed
  `/usr/bin/dbclient`, a place these systems do not have.
- **Server options off** that need `crypt()` and `setresgid()`, which QNX
  lacks, and which Dropbear checks even when no server is built.

And from `qnx/build.sh`:

- **No compression** (`--disable-zlib`): QNX 6.5's SDK has no zlib.
- **No syslog**: BlackBerry 10 has no `<syslog.h>`. Messages go to the
  terminal, as a client's do anyway.

## Checking a build

    sh check.sh /path/to/dropbearmulti                 # on the device
    CHECK_KEY=key sh check.sh /path/to/dropbearmulti user@host

`qnx/check.sh` uses only the shell and the programs under test. Without a
host it checks what needs no server: the four names, key generation (RSA,
Ed25519, ECDSA), the public half, conversion to an OpenSSH key file and
back. With a host and a key that the server already accepts it also runs
a command there, tries the OpenSSH-style options, and copies a file there
and back with `scp`. It never adds a key anywhere, and keeps its own home
folder, so nothing is written to `~/.ssh`.

## What to know when using it

- **The first connection to a host asks whether to trust it**, as `ssh`
  does, and remembers the answer in `~/.ssh/known_hosts`. `-y` accepts a
  new host without asking; `-y -y` does not check at all.
- **Options as `ssh` takes them:** `-l user`, `-p port`, `-i keyfile`,
  `-s` (the command is a subsystem), `-t`/`-T`, `-N`, `-f`, and `-o` with
  `BatchMode`, `BindAddress`, `Compression`, `DisableTrivialAuth`,
  `ExitOnForwardFailure`, `ForwardAgent`, `GatewayPorts`, `IdentityFile`,
  `PasswordAuthentication`, `Port`, `ProxyCommand`, `ServerAliveInterval`,
  `StrictHostKeyChecking`, `UseSyslog`. Any other `-o` is passed over with
  a warning, not refused. It reads no configuration file.
- **Keys are Dropbear's own file format.** `dropbearconvert openssh
  dropbear id_rsa mykey` brings an OpenSSH key over.
- On QNX 6.5 files are opened without `O_NOFOLLOW`, which that system does
  not have.

## Licences

Dropbear is under an MIT-style licence; `LICENSE` has it, together with
those of the parts taken from elsewhere: `scp` and a few helpers from
OpenSSH (BSD), and others named there. Its cryptography is libtomcrypt and
libtommath, which their own licence files (`libtomcrypt/LICENSE`,
`libtommath/LICENSE`) release into the public domain (libtomcrypt
alternatively under the WTFPL). `qnx/check-names.sh`
is under the MIT licence; its notice is in the file.

The programs also contain what the toolchain links in: for BlackBerry 10 a
few helper functions of LLVM's compiler-rt (Apache License 2.0 with LLVM
exceptions); for QNX 6.5 gcc's start-up objects and helper functions
(`crtbegin`, `libgcc`; GPL with the GCC runtime library exception).
