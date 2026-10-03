// Ported from src/terminal/apc.zig (Ghostty v1.3.1).
//
// Kitty graphics is compiled out of the library build
// (build_options.kitty_graphics is false), so every
// `if (comptime build_options.kitty_graphics)` branch takes its else side
// and the Kitty variants of State and Command are omitted. See
// docs/mapping.md.

package terminal

// APCHandler is Handler: the handler for APC sequences.
type APCHandler struct {
	state APCState
}

// APCStateTag is the tag of State.
type APCStateTag uint8

const (
	APCStateInactive APCStateTag = iota
	APCStateIgnore
	APCStateIdentify
)

// APCState is State: the possible states of the handler. Zig: a tagged
// union whose only non-void variant (kitty) is compiled out.
type APCState struct {
	tag APCStateTag
}

// APCCommand is Command: the possible APC commands. Zig: a union whose only
// variant (kitty) is compiled out, so no value of it can exist.
type APCCommand struct{}

// Deinit is Handler.deinit.
func (self *APCHandler) Deinit() {
	self.state.Deinit()
}

// Start is Handler.start.
func (self *APCHandler) Start() {
	self.state.Deinit()
	self.state = APCState{tag: APCStateIdentify}
}

// Feed is Handler.feed.
func (self *APCHandler) Feed(byte_ uint8) {
	switch self.state.tag {
	case APCStateInactive:
		panic("unreachable")

	// We're ignoring this APC command, likely because we don't recognize it.
	case APCStateIgnore:
		return

	// We aren't sure what command this is yet, it is determined by the
	// first byte.
	case APCStateIdentify:
		switch byte_ {
		// 'G' is the Kitty graphics protocol, compiled out.
		default:
			self.state = APCState{tag: APCStateIgnore}
		}
	}
}

// End is Handler.end.
func (self *APCHandler) End() Optional[APCCommand] {
	defer func() {
		self.state.Deinit()
		self.state = APCState{tag: APCStateInactive}
	}()

	switch self.state.tag {
	case APCStateInactive:
		panic("unreachable")
	case APCStateIgnore, APCStateIdentify:
		return None[APCCommand]()
	}
	panic("unreachable")
}

// Deinit is State.deinit.
func (self *APCState) Deinit() {
	switch self.tag {
	case APCStateInactive, APCStateIgnore, APCStateIdentify:
	}
}

// Deinit is Command.deinit. In the library build no APCCommand value can
// exist, so this can never be reached with a real command.
func (self *APCCommand) Deinit() {
	panic("unreachable")
}
