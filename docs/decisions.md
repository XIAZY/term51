# Decisions

Product and design decisions for the term51 app, newest first: what was
decided, by whom, and why. The project owner's rules are at the end. A
decision is added here when it is made; one that is reversed is marked, not
deleted.

| Date (UTC) | Decision | By | Why |
|---|---|---|---|
| 2026-10-07 | The key-logging switch (`logInput`) is removed as soon as the Passport's keyboard has been checked with it. | mitchellh | A diagnostic is not a setting; and it would write what is typed to a file. |
| 2026-10-07 | The first start writes the settings file with every setting at its default and a comment on each. | anchron, mitchellh | Until there is a settings page a user needs something to edit. |
| 2026-10-07 | A settings file with a mistake in it: the app starts on the defaults and the log names the mistake. | anchron | A person editing by hand will make mistakes; the app should still open. |
| 2026-10-07 | Code for cases that cannot happen is removed across the app (size checks no caller can fail, a second way of opening the terminal device that could never run, guards in the page for problems nobody had seen). | mitchellh, anchron | The owner's rule; each removal named the reason the case cannot occur. |
| 2026-10-07 | One way to show the terminal: the Cascades text grid. The pixel-drawn window, its C code and the setting that chose between them are removed. | AlphaToad | No fallbacks; the first run on the phone showed the text grid lines up. |
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
