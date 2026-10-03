// Ported from src/terminal/dcs.zig (Ghostty v1.3.1).
//
// tmux control mode is compiled out of the library build
// (build_options.tmux_control_mode is false); the tmux variants of State
// and Command are omitted and the `if (comptime
// build_options.tmux_control_mode)` branches take their else side. See
// docs/mapping.md.

package terminal

import "errors"

// ErrOutOfMemory is Zig's error.OutOfMemory.
var ErrOutOfMemory = errors.New("OutOfMemory")

// DCSHandler is Handler.
type DCSHandler struct {
	state DCSState

	// maximum number of bytes that can be stored for a command. This
	// prevents malicious input from causing OOM. This is a generous limit
	// but still stops it from being unbounded.
	max_bytes uint
}

// DCSHandlerInit returns a Handler with Zig's field defaults.
func DCSHandlerInit() DCSHandler {
	return DCSHandler{state: DCSState{tag: DCSStateInactive}, max_bytes: 1024 * 1024}
}

// Deinit is Handler.deinit.
func (self *DCSHandler) Deinit() {
	self.discard()
}

// Hook is Handler.hook.
func (self *DCSHandler) Hook(dcs ParserActionDCS) Optional[DCSCommand] {
	assert(self.state.tag == DCSStateInactive)

	// Initialize our state to ignore in case of error
	self.state = DCSState{tag: DCSStateIgnore}

	// Try to parse the hook.
	hk_, err := self.tryHook(dcs)
	if err != nil {
		warn("terminal_dcs", "error initializing DCS hook, will ignore hook err=%v", err)
		return None[DCSCommand]()
	}
	if !hk_.Valid {
		warn("terminal_dcs", "unknown DCS hook: %v", dcs)
		return None[DCSCommand]()
	}
	hk := hk_.Value

	self.state = hk.state
	return hk.command
}

// dcsHook is Handler.Hook (the private result struct of tryHook).
type dcsHook struct {
	state   DCSState
	command Optional[DCSCommand]
}

// tryHook is Handler.tryHook.
//
// Zig takes `self: Handler` by value; the receiver is only read.
func (self DCSHandler) tryHook(dcs ParserActionDCS) (Optional[dcsHook], error) {
	switch len(dcs.Intermediates) {
	case 0:
		switch dcs.Final {
		// Tmux control mode: compiled out of the library build.
		case 'p':
			return None[dcsHook](), nil
		default:
			return None[dcsHook](), nil
		}
	case 1:
		switch dcs.Intermediates[0] {
		case '+':
			switch dcs.Final {
			// XTGETTCAP
			// https://github.com/mintty/mintty/wiki/CtrlSeqs#unix-terminfo-and-termcap-queries
			case 'q':
				return Some(dcsHook{
					state: DCSState{tag: DCSStateXtgettcap, xtgettcap: &dcsWriterAllocating{}},
				}), nil
			default:
				return None[dcsHook](), nil
			}
		case '$':
			switch dcs.Final {
			// DECRQSS
			case 'q':
				return Some(dcsHook{state: DCSState{tag: DCSStateDecrqss}}), nil
			default:
				return None[dcsHook](), nil
			}
		default:
			return None[dcsHook](), nil
		}
	default:
		return None[dcsHook](), nil
	}
}

// Put is Handler.put.
func (self *DCSHandler) Put(byte_ uint8) Optional[DCSCommand] {
	cmd, err := self.tryPut(byte_)
	if err != nil {
		warn("terminal_dcs", "error putting byte into DCS handler err=%v", err)
		self.discard()
		self.state = DCSState{tag: DCSStateIgnore}
		return None[DCSCommand]()
	}
	return cmd
}

// tryPut is Handler.tryPut.
func (self *DCSHandler) tryPut(byte_ uint8) (Optional[DCSCommand], error) {
	switch self.state.tag {
	case DCSStateInactive, DCSStateIgnore:
	case DCSStateXtgettcap:
		list := self.state.xtgettcap
		if uint(len(list.written())) >= self.max_bytes {
			return None[DCSCommand](), ErrOutOfMemory
		}
		list.writeByte(byte_)
	case DCSStateDecrqss:
		buffer := &self.state.decrqss
		if buffer.len >= uint8(len(buffer.data)) {
			return None[DCSCommand](), ErrOutOfMemory
		}
		buffer.data[buffer.len] = byte_
		buffer.len += 1
	}

	return None[DCSCommand](), nil
}

// Unhook is Handler.unhook.
func (self *DCSHandler) Unhook() Optional[DCSCommand] {
	// Note: we do NOT call deinit here on purpose because some commands
	// transfer memory ownership. If state needs cleanup, the switch
	// prong below should handle it.
	defer func() { self.state = DCSState{tag: DCSStateInactive} }()

	switch self.state.tag {
	case DCSStateInactive, DCSStateIgnore:
		return None[DCSCommand]()

	case DCSStateXtgettcap:
		list := self.state.xtgettcap
		items := list.written()
		for i, b := range items {
			// std.ascii.toUpper
			if b >= 'a' && b <= 'z' {
				items[i] = b - 'a' + 'A'
			}
		}
		return Some(DCSCommand{tag: DCSCommandXtgettcap, xtgettcap: DCSCommandXTGETTCAP{data: list}})

	case DCSStateDecrqss:
		buffer := self.state.decrqss
		var d DCSCommandDECRQSS
		switch buffer.len {
		case 0:
			d = DCSCommandDECRQSSNone
		case 1:
			switch buffer.data[0] {
			case 'm':
				d = DCSCommandDECRQSSSgr
			case 'r':
				d = DCSCommandDECRQSSDecstbm
			case 's':
				d = DCSCommandDECRQSSDecslrm
			default:
				d = DCSCommandDECRQSSNone
			}
		case 2:
			switch buffer.data[0] {
			case ' ':
				switch buffer.data[1] {
				case 'q':
					d = DCSCommandDECRQSSDecscusr
				default:
					d = DCSCommandDECRQSSNone
				}
			default:
				d = DCSCommandDECRQSSNone
			}
		default:
			panic("unreachable")
		}
		return Some(DCSCommand{tag: DCSCommandDecrqss, decrqss: d})
	}
	panic("unreachable")
}

// discard is Handler.discard.
func (self *DCSHandler) discard() {
	self.state.Deinit()
	self.state = DCSState{tag: DCSStateInactive}
}

// DCSCommandTag is the tag of Command.
type DCSCommandTag uint8

const (
	DCSCommandXtgettcap DCSCommandTag = iota
	DCSCommandDecrqss
)

// DCSCommand is Command. Zig: a tagged union; only the field named by tag
// is meaningful. The tmux variant is compiled out.
type DCSCommand struct {
	tag       DCSCommandTag
	xtgettcap DCSCommandXTGETTCAP
	decrqss   DCSCommandDECRQSS
}

// Deinit is Command.deinit.
func (self *DCSCommand) Deinit() {
	switch self.tag {
	case DCSCommandXtgettcap:
		self.xtgettcap.data.deinit()
	case DCSCommandDecrqss:
	}
}

// DCSCommandXTGETTCAP is Command.XTGETTCAP.
type DCSCommandXTGETTCAP struct {
	data *dcsWriterAllocating
	i    uint
}

// Next returns the next terminfo key being requested and null when there
// are no more keys. The returned value is NOT hex-decoded because we expect
// to use a comptime lookup table.
//
// Zig: XTGETTCAP.next.
func (self *DCSCommandXTGETTCAP) Next() Optional[[]uint8] {
	items := self.data.written()
	if self.i >= uint(len(items)) {
		return None[[]uint8]()
	}
	rem := items[self.i:]
	idx := uint(len(rem))
	for j, b := range rem {
		if b == ';' {
			idx = uint(j)
			break
		}
	}

	// Note that if we're at the end, idx + 1 is len + 1 so we're over
	// the end but that's okay because our check above is >= so we'll
	// never read.
	self.i += idx + 1

	return Some(rem[0:idx])
}

// DCSCommandDECRQSS is Command.DECRQSS: supported DECRQSS settings.
type DCSCommandDECRQSS uint8

const (
	DCSCommandDECRQSSNone DCSCommandDECRQSS = iota
	DCSCommandDECRQSSSgr
	DCSCommandDECRQSSDecscusr
	DCSCommandDECRQSSDecstbm
	DCSCommandDECRQSSDecslrm
)

// DCSStateTag is the tag of State.
type DCSStateTag uint8

const (
	DCSStateInactive DCSStateTag = iota
	DCSStateIgnore
	DCSStateXtgettcap
	DCSStateDecrqss
)

// DCSState is State. Zig: a tagged union; only the field named by tag is
// meaningful. The tmux variant is compiled out.
type DCSState struct {
	tag       DCSStateTag
	xtgettcap *dcsWriterAllocating
	decrqss   dcsDecrqssBuffer
}

// dcsDecrqssBuffer is the anonymous struct payload of State.decrqss.
type dcsDecrqssBuffer struct {
	data [2]uint8
	// Zig: u2.
	len uint8
}

// Deinit is State.deinit.
func (self *DCSState) Deinit() {
	switch self.tag {
	case DCSStateInactive, DCSStateIgnore:
	case DCSStateXtgettcap:
		self.xtgettcap.deinit()
	case DCSStateDecrqss:
	}
}

// dcsWriterAllocating stands in for std.Io.Writer.Allocating, of which
// dcs.zig uses only written() and writeByte(). See docs/mapping.md.
type dcsWriterAllocating struct {
	buf []uint8
}

func (self *dcsWriterAllocating) written() []uint8 {
	return self.buf
}

func (self *dcsWriterAllocating) writeByte(b uint8) {
	self.buf = append(self.buf, b)
}

func (self *dcsWriterAllocating) deinit() {
	self.buf = nil
}
