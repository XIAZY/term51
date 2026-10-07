# On the phone

The app shows the terminal one way: a column of Cascades text labels, one
per row (`assets/main.qml`, `internal/textgrid`). Everything below the page
is tested on the development machine with real shells; the page itself can
only be seen on the phone.

Build with `GOQNX=<go-qnx> tools/build.sh`; install with
`tools/install.sh <ssh-host>` (the app must be closed: the phone does not
replace an open app). The settings are `data/settings.conf` in the app's
sandbox (`internal/settings`); `logInput = true` there writes every key and
text command to `data/term51.log`.

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

## Still to look at

1. The input field: symbols, Enter, Backspace, and that no character is
   typed twice (characters come from the field, other keys from key
   events).
2. What the keyboard delivers for Alt, Shift and the symbol key
   (`logInput`), against what `internal/keys` assumes.
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
