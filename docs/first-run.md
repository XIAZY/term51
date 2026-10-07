# On the phone

The app shows the terminal one way: drawn with OpenGL ES 3.0 in a window of
its own that sits in the page (`internal/gpu`, `internal/cells`,
`assets/main.qml`). Everything below the page
is tested on the development machine with real shells; the page itself can
only be seen on the phone.

Build with `GOQNX=<go-qnx> tools/build.sh`; install with
`tools/install.sh <ssh-host>` (the app must be closed: the phone does not
replace an open app). The settings are `data/settings.conf` in the app's
sandbox (`internal/settings`); the log is `data/term51.log`.

## Seen to work (2026-10-07, first run)

- The app starts and its log has no errors.
- A pseudo-terminal opens and the shell starts (the BSD devices
  `/dev/ptyp?`; BlackBerry 10's C library has no `posix_openpt`).
- The text lines up in a fixed-width font, and there is a prompt.
- The physical keys type.

## Found, and changed

- **Symbols could not be typed:** without an input field on the page the
  phone does not show its on-screen keyboard rows. The page now has one; it
  holds the focus, and what is typed into it goes to the shell.

- **The app could not be closed** (2026-10-07). After the owner closed it,
  the process stayed behind with one thread, `is_client_reader`, waiting for
  a reply from the phone's input service (`imfwk`); it could not be killed,
  the installer refused to install over it, and only a restart removed it.
  The cause was how the shell was started: asking Go for a controlling
  terminal made it start the shell with vfork, which copies every descriptor
  the app has open, so the shell held copies of the connections Cascades
  had made, the one to the input service among them. Changed: the shell is
  started in two steps so that Go uses spawn, which hands over only the
  terminal (`internal/pty`, Start). Checked on the phone with main b8e630b:
  with the app open the shell holds only its terminal (`/dev/ttyp0` on
  descriptors 0, 1, 2 and one of its own) against 47 entries in the app,
  and after closing the app no process and no shell are left.
- **Found on the way, also changed:** the process called the C library's
  `exit` from two threads (`os.Exit` is that on this system). It now ends in
  one place, on the main thread after the Cascades event loop has returned
  (`main.go`). This was not the cause of the hang above.
- **Half of each cell was missing in the right third of the window**
  (2026-10-07, the first GPU build, 69ed2e3). With a 60-column test table
  every cell in columns 40 to 59 was shown without its upper-left triangle,
  the first of the two that make a cell; columns 0 to 39 were whole. The
  cells' corners were then computed from `gl_VertexID`, with no per-vertex
  attribute at all. Changed: the corners come from a per-vertex attribute, a
  fixed buffer of four corners (`internal/gpu`). With that (9b02a81) the
  screen is whole to the right edge. The loss could not be reproduced by a
  stand-alone program with the same drawing code that reads its pixels
  back, so it is recorded as gone with the change, not as explained.
- **After a resize only the old part of the picture was shown.** The window
  system keeps the part of the buffer it shows (the source size) at the
  first buffers' size. Changed: it is set with every new set of buffers.
- **The page's first size is not its last.** The page reports its place for
  the terminal in pixels, and correctly, but the first report comes before
  the page has settled (948 high, then 804, then 828), and Cascades slides
  the window to a new size over several frames, so the window's own size
  lags what the page has said.

## Still to look at

1. The input field: symbols, Enter, Backspace, and that no character is
   typed twice (characters come from the field, other keys from key
   events).
2. What the keyboard delivers for Alt, Shift and the symbol key, against
   what `internal/keys` assumes.
3. The key bar: sticky Ctrl and Alt, Esc, Tab, the arrows; an Enter typed
   after tapping a button reaches the shell.
4. Colours: a background colour inside a row, the cursor as an inverted
   cell.
5. Full-screen programs: wide characters take two cells; the borders of
   `mc` or `htop` have no gaps between rows.
6. Speed: typing echo, and `cat` of a large file (the labels are updated at
   most 30 times a second).
7. The shell's exit is seen when a background job still holds the terminal
   (`sleep 1000 &`, then `exit`).
8. Copy and paste from the menu, a long press selecting a word, and tabs.
