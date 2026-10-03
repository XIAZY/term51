// WIREFRAME for src/terminal/stream_readonly.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: stream_readonly.zig is a namespace file, so its declarations
// take the prefix `StreamReadonly`: `stream_readonly.Stream` is
// `StreamReadonlyStream`, `stream_readonly.Handler` is
// `StreamReadonlyHandler` and its constructor `Handler.init` is
// `StreamReadonlyHandlerInit`. `pub fn` methods are exported (`Deinit`,
// `Vt`); private methods keep their Zig name (`horizontalTab`,
// `horizontalTabBack`, `setMode`, `colorOperation`, `kittyColorOperation`).
//
// Declarations appear in the same order as in stream_readonly.zig.
//
// # The handler and the stream
//
// Zig: `pub const Stream = stream.Stream(Handler);`, a stream that holds a
// `Handler` by value. In Go `Stream[H]` holds its handler by pointer
// (docs/mapping.md, "Naming rules settled with the small-file wireframes"),
// and the methods of the handler have pointer receivers as in Zig
// (`self: *Handler`), so the instantiation is
// `Stream[*StreamReadonlyHandler]`. The line
//
//	var _ StreamHandler = (*StreamReadonlyHandler)(nil)
//
// below makes the compiler check that the handler satisfies StreamHandler
// (stream.go): `Vt(action StreamActionKey, value StreamAction) error` and
// `Deinit()`.
//
// `Handler.vt` is `fn(self: *Handler, comptime action: Action.Tag, value:
// Action.Value(action)) !void`. As stream.go explains, Go has no comptime
// parameter and no type that depends on a value: `Vt` takes the action key
// and the whole StreamAction union, of which the field named by the key is
// the payload Zig passes.
//
// The handler has no `vtRaw`, so it does not implement StreamHandlerVtRaw.
//
// # Declarations that are not Zig declarations
//
//   - the compile-time assertion `var _ StreamHandler = ...`.
//
// # Omitted
//
// Aliases and imports with nothing to declare: `std`, `testing`, `stream`,
// `Action` (use StreamAction), `Screen`, `modes`, `osc_color` (the
// OSCParsersColor... names), `kitty_color` (the KittyColor... names),
// `Terminal`.
// No C ABI declaration exists in this file. Nothing in it is under
// `build_options.kitty_graphics`: the APC actions, which could carry Kitty
// graphics, are ignored by `vt` in every build.
//
// # Zig defaults a Go literal does not give
//
// None. `Handler` has one field and no default.

package terminal

// StreamReadonlyStream is stream_readonly.Stream.
//
// This is a Stream implementation that processes actions against
// a Terminal and updates the Terminal state. It is called "readonly" because
// it only processes actions that modify terminal state, while ignoring
// any actions that require a response (like queries).
//
// If you're implementing a terminal emulator that only needs to render
// output and doesn't need to respond (since it maybe isn't running the
// actual program), this is the stream type to use. For example, this is
// ideal for replay tooling, CI logs, PaaS builder output, etc.
//
// Zig: pub const Stream = stream.Stream(Handler). See the file header for
// why the Go handler type is a pointer.
type StreamReadonlyStream = Stream[*StreamReadonlyHandler]

// StreamReadonlyHandler is stream_readonly.Handler.
//
// See Stream, which is just the stream wrapper around this.
//
// This isn't attached directly to Terminal because there is additional
// state and options we plan to add in the future, such as APC/DCS which
// don't make sense to me to add to the Terminal directly. Instead, you
// can call `vtHandler` on Terminal to initialize this handler.
type StreamReadonlyHandler struct {
	// The terminal state to modify.
	terminal *Terminal
}

// The handler satisfies what stream.Stream requires of its Handler type.
var _ StreamHandler = (*StreamReadonlyHandler)(nil)

// StreamReadonlyHandlerInit is stream_readonly.Handler.init.
//
// Zig: pub fn init(terminal: *Terminal) Handler.
func StreamReadonlyHandlerInit(terminal *Terminal) StreamReadonlyHandler {
	panic(notImplemented)
}

// Deinit is stream_readonly.Handler.deinit.
//
// Currently does nothing but may in the future so callers should
// call this.
func (self *StreamReadonlyHandler) Deinit() { panic(notImplemented) }

// Vt is stream_readonly.Handler.vt.
//
// Zig: pub fn vt(self: *Handler, comptime action: Action.Tag,
// value: Action.Value(action)) !void. `value.Tag` equals `action`; the
// payload is the field of `value` named by it. The inferred error set is
// that of the Terminal functions it calls with `try`: print, printRepeat,
// linefeed, index, scrollUp, decaln, semanticPrompt, Screen.startHyperlink
// and the private setMode, colorOperation and kittyColorOperation below.
// The error of `setAttribute` is discarded (`catch {}`).
func (self *StreamReadonlyHandler) Vt(action StreamActionKey, value StreamAction) error {
	panic(notImplemented)
}

// horizontalTab is stream_readonly.Handler.horizontalTab.
//
// Zig: inline fn horizontalTab(self: *Handler, count: u16) void.
func (self *StreamReadonlyHandler) horizontalTab(count uint16) { panic(notImplemented) }

// horizontalTabBack is stream_readonly.Handler.horizontalTabBack.
//
// Zig: inline fn horizontalTabBack(self: *Handler, count: u16) void.
func (self *StreamReadonlyHandler) horizontalTabBack(count uint16) { panic(notImplemented) }

// setMode is stream_readonly.Handler.setMode.
//
// Zig: fn setMode(self: *Handler, mode: modes.Mode, enabled: bool) !void.
// The inferred error set is that of Terminal.switchScreenMode and
// Terminal.deccolm.
func (self *StreamReadonlyHandler) setMode(mode ModesMode, enabled bool) error {
	panic(notImplemented)
}

// colorOperation is stream_readonly.Handler.colorOperation.
//
// Zig: fn colorOperation(self: *Handler, op: osc_color.Operation,
// requests: *const osc_color.List) !void. Nothing in the body can fail; the
// inferred error set is empty and the Go result is always nil.
func (self *StreamReadonlyHandler) colorOperation(
	op OSCParsersColorOperation,
	requests *OSCParsersColorList,
) error {
	panic(notImplemented)
}

// kittyColorOperation is stream_readonly.Handler.kittyColorOperation.
//
// Zig: fn kittyColorOperation(self: *Handler, request: kitty_color.OSC)
// !void. Nothing in the body can fail; the inferred error set is empty and
// the Go result is always nil.
func (self *StreamReadonlyHandler) kittyColorOperation(request KittyColorOSC) error {
	panic(notImplemented)
}
