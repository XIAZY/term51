# Optimisations deferred until the port passes

The port is literal on purpose, and some of the literal translations are
slower in Go than the original is in Zig. Nothing here may be changed until
every translated test and every recorded case passes; then each item is its
own change, measured before and after.

| # | Where | What is slow | Why it is that way now |
|---|---|---|---|
| 1 | `Parser.Next` | Returns `[3]Optional[ParserAction]` per input byte. `ParserAction` is a struct holding every payload, and will carry a whole OSC command by value once `osc.zig` is ported, so three of them are copied for every byte. | Zig returns `[3]?Action`, a small tagged union; the struct-with-tag mapping makes it large. |
| 2 | `Parser.Next` | A deferred closure runs per byte to set the next state. | Zig's `defer self.state = next_state` costs nothing; Go's `defer` with a closure does. |
