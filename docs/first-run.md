# On the phone

The app shows the terminal one way: drawn with OpenGL ES 3.0 in a window of
its own that sits in the page (`internal/gpu`, `internal/cells`,
`assets/main.qml`). Everything below the page
is tested on the development machine with real shells; the page itself can
only be seen on the phone.

Build with `GOQNX=<go-qnx> tools/build.sh`; install with
`tools/install.sh <ssh-host>` (the app must be closed: the phone does not
replace an open app). The settings are `~/.config/term51/settings.conf`,
where `~` is the app's home, the `data` folder of its sandbox
(`internal/layout`, `internal/settings`); the log is `logs/term51.log` in the
sandbox.

## Checking a page change before it is installed

The page (`assets/*.qml`) is compiled by the phone, when the app starts: a
mistake in it stops the app at once, and nothing on the development machine
can find it. So every page change is loaded on the phone by hand first. It
needs no install and does not disturb an installed copy that is running:

1. Copy `build/term51` to `/dev/shmem/<name>` (the one place a program may
   be started from; it has no directories), and the page files to a folder
   on the SD card laid out like the app: `<folder>/app/native/assets/` with
   `images/` in it, and an empty `<folder>/data/`. Copy the files level by
   level; `scp -r` fails on the phone.
2. Start it as `devuser` from that folder:

       on -d -u devuser sh -c 'cd <folder> && HOME=<folder>/data LD_LIBRARY_PATH=/usr/lib/qt4/lib \
         QML_IMPORT_PATH=/usr/lib/qt4/imports QT_PLUGIN_PATH=/usr/lib/qt4/plugins \
         CASCADES_THEME=dark GRAPHICS_ROOT=/base/usr/lib/graphics/qc \
         /dev/shmem/<name> > <folder>/out.txt 2>&1'

   Without `QML_IMPORT_PATH` it stops at the first `import`. Without `HOME`
   the shells start in root's, which `devuser` cannot enter: the first shell
   ends at once (exit status 127) and the app with it.
3. A page with a mistake prints `ui: the page does not load:` with the file,
   line and reason, and the process ends. A page that loads prints nothing
   and the process stays (so run the ssh command with a time limit). It
   gets no window this way: this shows that the page compiles, not how it
   looks.
4. Stop it by its process number (`pidin`, `kill`), and remove the folder
   and the program.

The same error text is in the system log of an installed copy that fails
(`slog2info`, the lines with `QmlDocument`), and since 13a910f in the app's
`logs/log`.

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

- **The first native page did not start** (2026-10-07). The app menu's
  "Close tab" was a `DeleteActionItem`, which an app menu does not accept
  (`Cannot assign object to list`), and one such mistake stops the whole
  page from loading. Changed: a plain action with an icon of ours; the
  bridge prints why a page does not load; and pages are checked on the phone
  before they are installed (above).

- **The system's menu opens without its bar** (2026-10-07, 5c1046a).
  `Page.openActionMenu()` opens the menu from the right for a page's
  overflow actions also while the action bar is hidden
  (`actionBarVisibility: ChromeVisibility.Hidden`); the SDK's headers do not
  say so. It needs 10.3.1; the Passport runs 10.3.3. The app used it while it
  had a bar of its own; with the system's bar, which has its own three dots,
  it no longer does.

- **A gap under the bottom bar** (2026-10-07, d1d9025). The hidden input
  field, told to have no height (`maxHeight: 0`), kept its 120 pixels at the
  end of the page's column. Changed: the field lies under the rest of the
  page in a dock layout and is no part of the column.
- **Margins across a row's direction are not applied:** a divider with a
  top and bottom margin in a left-to-right stack ran the bar's whole height.
  Changed: padding on a container around it.

- **An app cannot quietly take a picture of the display** (2026-10-07).
  `bb::system::Screenshot::captureDisplay` puts a request on the screen for
  every picture, with no "always allow"; until it is answered the file is
  empty, and the log has "error 5". Tried as a way for the developers to see
  the page, and removed, with the remote control that went with it.

- **What the system's action bar holds** (2026-10-07, 5fb459e on the phone,
  and the SDK's headers). Between the tabs button and the three dots it
  shows three actions; more go into the dots' menu. Every action on the bar
  is listed in that menu as well ("The action will also be placed in the
  action menu"). An action's icon and its title are both shown, so an icon
  that is a word reads twice. Besides actions the bar takes one text field
  (`TextInputActionItem`, since 10.3: the browser's address field) and no
  control of an app's own. The field does not count among the three: the
  phone's own text-message app shows a field and three buttons. The bar is
  always at the bottom of the page; nothing in the public SDK moves it to
  the top.
- **The phone's terminal database is small** (2026-10-08): `/usr/lib/terminfo`
  has `xterm`, `xterms`, `vt100`, `vt102`, `vt52`, `ansi` and QNX's own
  (`qansi`, `qnx` and their variants), and neither `xterm-256color` nor
  `vt220`. The app's default had been `xterm-256color`; it is `xterm` now,
  and `xterm-256color` is still offered, for programs on other machines
  over ssh, where the other machine's database counts.

- **Enter brought the on-screen rows up** (2026-10-08). With nothing
  focused, Enter on the physical keyboard gave the hidden input field the
  focus, and a focused text field is what makes the phone show its rows; no
  other key did that. Changed: the field's focus policy is "none" (the
  SDK: "only focusable programmatically") and it is no target for key
  navigation, so only the page's own `requestFocus()` on a tap focuses it.

- **A changed default does not reach a phone that has a settings file**
  (2026-10-08). The file holds every setting, written out in full at the
  first start and at every change on the settings sheet, so the value in it
  is the default of the build that wrote it, and a file always wins over a
  default. Seen with the terminal type: after the default became `xterm`,
  the owner's file still said `xterm-256color`, because switching the bell
  off earlier had saved everything. Kept as it is for now (one phone, one
  file); whoever changes a default next has to change the file on the phone
  as well, or say so to its user.

- **A stopped context menu leaves the finger with the control**
  (2026-10-08, 29a793a on the phone). The system tells a control of a press
  and hold by being about to show its context menu
  (`ContextMenuHandler.onPopulating`). Calling `abort()` on that event stops
  the menu, and the finger's later moves and its lift still reach the
  control's `onTouch`. That is how the arrow pad takes over a press and hold
  without a timer of the app's own; the menu is then opened from code when
  wanted (`ForeignWindowControl.showContextMenu()`).

## Still to look at

1. The input field: symbols, Enter, Backspace, and that no character is
   typed twice (characters come from the field, other keys from key
   events).
2. What the keyboard delivers for Alt, Shift and the symbol key, against
   what `internal/keys` assumes.
3. The key bar: sticky Ctrl and Alt, Esc, Tab; the arrow pad; an Enter typed
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
