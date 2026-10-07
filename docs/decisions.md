# Decisions

Product and design decisions for the term51 app, newest first: what was
decided, by whom, and why. The project owner's rules are at the end. A
decision is added here when it is made; one that is reversed is marked, not
deleted.

| Date (UTC) | Decision | By | Why |
|---|---|---|---|
| 2026-10-07 | The input field is hidden: it stays in the page and keeps the focus, but is not seen and takes no room; the "type here" bar is gone and the terminal reaches the bottom of the page. | AlphaToad | The field exists only because the phone shows its on-screen keyboard rows when a text field has the focus (see the input field row); a bar nobody types into by sight only took rows from the terminal. |
| 2026-10-07 | The font size setting is in real points: pixels = points × the display's dots per inch / 72. | mitchellh | A size should look the same on any display. |
| 2026-10-07 | The terminal's fonts are read from the phone's own files (DejaVu Sans Mono in four styles; MHei, then Malgun, for what it lacks); the app ships none. A file that cannot be read ends the app with the reason in the log. | mitchellh | The files are part of the phone, and its Chinese, Japanese and Korean fonts come for free. |
| 2026-10-07 | The terminal is drawn with OpenGL ES 3.0 in a window of the app's own inside the Cascades page, from a table of cells and a sheet of character shapes built in Go; one call from Go into C per changed frame. It replaces the text grid: the app has one way to show the terminal and no setting to choose. | AlphaToad | His start order of 2026-10-07 (#proj-term51). |
| 2026-10-07 | A shell is started in two steps: the app starts itself as a session leader holding only the terminal, and that process makes the terminal its controlling terminal and becomes the shell. | mitchellh, anchron | Letting Go set the controlling terminal makes it start the shell with vfork, which copies every descriptor the app has open, the connections Cascades made to the phone's services included. The two steps let Go use spawn, which hands over only the terminal. |
| 2026-10-07 | The process ends in one place: on the main thread, after the Cascades event loop has returned. When the last shell goes, Go asks the UI to quit instead of exiting itself. | anchron | On this system `os.Exit` is the C library's `exit`, which runs the libraries' clean-up on the calling thread. Called from another thread it left a thread waiting on the input service for ever and a process that could not be killed. |
| 2026-10-07 | **Reversed the same day (see the OpenGL ES row).** The terminal stays drawn by the Cascades text grid. The grid's speed is measured on the phone in real use; only if it is too slow is it replaced (not joined) by a window in the page that Go draws, calling the window library directly without C. Cascades is not dropped. | mitchellh | It works on the phone and gives the on-screen symbol rows and system fonts for free; nothing yet shows it is too slow. A call into C costs about 0.59 microseconds on the phone by cgo or directly, so that cost does not decide it. |
| 2026-10-07 | Go hands the page one message per screen update, carrying every changed row; no redesign to cross into C++ less often. | anchron | It is already at most 30 crossings a second and none when the screen is still, about a microsecond each; the time in an update is Cascades laying out the rows, which does not depend on how they are handed over. |
| 2026-10-07 | The identity texts in the settings (name, version, answerback) may contain any characters; the earlier ban on control characters is removed. | anchron | An answerback can legitimately contain control characters, and the ban protected nothing. |
| 2026-10-07 | The settings file is plain text, one `name = value` per line with `#` comments. A name the app does not know is a warning and the rest of the file is used. | mitchellh, anchron | It is edited by hand on the phone, and an older app must survive a file written for a newer one. |
| 2026-10-07 | Output is shown at once when the terminal has been quiet, and at most 30 times a second while it keeps changing. | mitchellh, from anchron's review | The echo of a typed key must not wait, and a flood of output must not make the page lay text out again for every line. |
| 2026-10-07 | The key bar's Ctrl and Alt are sticky: each applies to the next key that is not itself a modifier, and is then released. | anchron | The Passport has no Ctrl key, and holding an on-screen key while typing on the keyboard is awkward. |
| 2026-10-07 | The app is built without profile-guided optimisation for now. | mitchellh, anchron | It needs a profile from a real session on the phone. A trial on the development machine gave +30% on plain text, so it is worth doing once there is one. |
| 2026-10-07 | The key-logging switch (`logInput`) is removed as soon as the Passport's keyboard has been checked with it. | mitchellh | A diagnostic is not a setting; and it would write what is typed to a file. |
| 2026-10-07 | The first start writes the settings file with every setting at its default and a comment on each. | anchron, mitchellh | Until there is a settings page a user needs something to edit. |
| 2026-10-07 | A settings file with a mistake in it: the app starts on the defaults and the log names the mistake. | anchron | A person editing by hand will make mistakes; the app should still open. |
| 2026-10-07 | Code for cases that cannot happen is removed across the app (size checks no caller can fail, a second way of opening the terminal device that could never run, guards in the page for problems nobody had seen). | mitchellh, anchron | The owner's rule; each removal named the reason the case cannot occur. |
| 2026-10-07 | **Reversed the same day (see the OpenGL ES row).** One way to show the terminal: the Cascades text grid. The pixel-drawn window, its C code and the setting that chose between them are removed. | AlphaToad | No fallbacks; the first run on the phone showed the text grid lines up. |
| 2026-10-07 | The page has an input field that holds the focus; characters come from it, other keys from its key listener, Enter from its submit. | mitchellh | The owner could not type `$`: without a text field the phone does not show its on-screen keyboard rows. |
| 2026-10-07 | Copy takes only a selection, at most its last 512 KB. Copy with nothing selected does nothing and says so. | mitchellh | "Select all" over a long scrollback sent megabytes to the page in one message. |
| 2026-10-07 | Closing a tab hangs the shell up and, if it has not gone after half a second, kills its process group; the app does not exit before its shells are gone. | mitchellh | A shell that ignores the hang-up made closing hang (shown by a test). |
| 2026-10-07 | The shell's exit is taken from waiting for the shell, not from the end of its output. | mitchellh | A background job can keep the terminal open after the shell has gone. |
| 2026-10-07 | Tabs: any number of shells, one shown; the app ends with the last one. | mitchellh | Several shells on a phone without several windows. |
| 2026-10-07 | The Passport's Alt key (the symbol layer) is not passed on as a terminal modifier when it changed the character; a terminal Alt and Ctrl come from the key bar, as sticky modifiers. | anchron | Alt+Q must type `#`, not ESC `#`. |
| 2026-10-07 | Ctrl+[ is sent as the library sends it, not as a bare ESC. | mitchellh | The key bar has Esc; the library stays as upstream. |
| 2026-10-07 | The answers to programs' queries are the app's own code (`internal/replies`), configurable in the settings: terminal model, name, version, answerback, and a switch per kind of query. `TERM` is `xterm-256color` unless set. | AlphaToad (configurable), mitchellh | Upstream leaves the answers to the application; the owner asked for them to be configurable. |
| 2026-10-07 | The library is frozen at `v0.1.0`; no further speed work on it. | AlphaToad | 1.5 to 2.2 times slower than upstream's plain build is accepted. |
| 2026-10-06 | All logic in Go; the only code that is not Go is the Cascades bridge (C++) and the call that sets the terminal's size. | mitchellh | The pattern of the club's working Go and Cascades app. |

## The owner's rules

- The library: as close to upstream as possible while making sense as a Go
  library.
- No fallbacks and no code for situations that cannot happen; handle what
  can.
- No setting unless a user needs to choose something.
- Decide by what gives the app's users the most value, and keep this list.
- Develop on feature branches; merge to `main` after the other developer's
  review.
- Test on the development machine; the phone for what only it can show.
- Nothing on the phone's internal storage beyond the installed app; no
  system changes.
