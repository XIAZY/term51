# The first run on the phone

Everything in this repository has been built and tested on the development
machine, with a real shell. Nothing has run on the phone yet. These are the
things only the phone can answer, in the order to look at them. Each names
the one place that changes if the answer is not the assumed one.

## Before installing

- Installing needs the project owner's go-ahead (the phone's internal
  storage). Build with `GOQNX=<go-qnx> tools/build.sh`; install with
  `tools/install.sh <ssh-host>`.
- The front end is chosen by the file `data/frontend` in the app's sandbox
  (`text` or `pixels`; text when absent). `data/rawkeys`, when it exists,
  makes the pixel front end read raw key events. The log is
  `data/term51.log`.

## The shell

1. **A pseudo-terminal opens.** BlackBerry 10's C library has no
   `posix_openpt`; `internal/pty` scans the BSD devices `/dev/pty[p-s]?`.
   That scan has never opened a real device. (`internal/pty/pty.go`,
   `openBSD`.)
2. **The shell's exit is seen** when a background job still holds the
   terminal (`sleep 1000 &`, then `exit`). (`internal/session`, `wait`.)
3. **Can the runtime poll a pseudo-terminal?** If so, opening the master
   non-blocking lets `Close` interrupt the reader. One experiment.
   (`internal/pty`, `internal/session`.)

## Keys

4. **Log a dozen keys on both paths** (letters, digits and symbols with
   Alt, Shift, Enter, Backspace, Space, the symbol key) and compare with
   what `internal/keys` assumes: special keys as `KEYCODE_*` values, the
   symbol layer as "sym `#`, cap `q`, Alt set". Only `FromCascades` and
   `FromScreen` change.
5. **Which path delivers,** and whether a press arrives on both. The pixel
   front end serves one (`app.Config.RawKeys`).
6. **The key bar's buttons do not keep the focus** (an Enter typed after
   tapping Ctrl must reach the terminal).

## Copy and paste

- **The clipboard:** Copy (the selection, or the whole screen) and Paste
  from the page's menu go through `bb::system::Clipboard` in the bridge.
- **A long press selects the word under the finger** (text front end): the
  cell is computed from the touch position and the measured cell size.
  Selection by dragging is not there yet; a drag scrolls.

## The text front end (`assets/text.qml`, `internal/textgrid`)

7. **The font is monospace.** The page measures 80 `M` against 80 `i` and
   Go logs when they differ. The family name is `"Monospace"`.
8. **A background colour inside a span is drawn, and no-break spaces keep
   their width.** The approach rests on both.
9. **The grid holds:** wide characters take two cells; the borders of `mc`
   or `htop` have no gaps between rows; a full row does not end in "…".
10. **Speed:** typing echo, and `cat` of a large file (the labels are
    updated at most 30 times a second).

## The pixel front end (`assets/pixels.qml`, `internal/render`)

11. **The native window shows inside the page,** in the right place and
    size, and follows a resize. (`internal/ui/window.c`.)
12. **Pixel format:** the buffer is written as `0x00RRGGBB`; check the
    colour order and that the zero top byte is not transparency.
13. **Buffers:** the window is assumed to hand out two, in turn, and to keep
    their contents between frames (only changed rows are redrawn in each).
    (`internal/view`, `Frame`; `window.c`, `t51_win_buffer`.)
14. **Speed:** one row and a full frame, measured on the phone (0.18 ms and
    6.8 ms on the development machine).
15. **A font for Chinese, Japanese and emoji:** the built-in font has none
    (they show as boxes); a fallback to one of the phone's fonts is needed.

## Then decide

Which front end becomes the default: the text grid if 7 to 10 hold, the
pixels otherwise. And whether to build with profile-guided optimisation
(+30% on plain text in a trial on the development machine).
