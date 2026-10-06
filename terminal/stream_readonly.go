// Ported from src/terminal/stream_readonly.zig (Ghostty v1.3.1).
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
	return StreamReadonlyHandler{
		terminal: terminal,
	}
}

// Deinit is stream_readonly.Handler.deinit.
//
// Currently does nothing but may in the future so callers should
// call this.
func (self *StreamReadonlyHandler) Deinit() {
	// Currently does nothing but may in the future so callers should
	// call this.
	_ = self
}

// Vt is stream_readonly.Handler.vt.
//
// Zig: pub fn vt(self: *Handler, comptime action: Action.Tag,
// value: Action.Value(action)) !void. `value.Tag` equals `action`; the
// payload is the field of `value` named by it. The inferred error set is
// that of the Terminal functions it calls with `try`: print, printRepeat,
// linefeed, index, scrollUp, decaln, semanticPrompt, Screen.startHyperlink
// and the private setMode, colorOperation and kittyColorOperation below.
// The error of `setAttribute` is discarded (`catch {}`).
func (self *StreamReadonlyHandler) Vt(action StreamActionKey, value *StreamAction) error {
	switch action {
	case StreamActionKeyPrint:
		if err := self.terminal.Print(value.Print.cp); err != nil {
			return err
		}
	case StreamActionKeyPrintRepeat:
		if err := self.terminal.PrintRepeat(value.PrintRepeat); err != nil {
			return err
		}
	case StreamActionKeyBackspace:
		self.terminal.Backspace()
	case StreamActionKeyCarriageReturn:
		self.terminal.CarriageReturn()
	case StreamActionKeyLinefeed:
		if err := self.terminal.Linefeed(); err != nil {
			return err
		}
	case StreamActionKeyIndex:
		if err := self.terminal.Index(); err != nil {
			return err
		}
	case StreamActionKeyNextLine:
		if err := self.terminal.Index(); err != nil {
			return err
		}
		self.terminal.CarriageReturn()
	case StreamActionKeyReverseIndex:
		self.terminal.ReverseIndex()
	case StreamActionKeyCursorUp:
		self.terminal.CursorUp(uint(value.CursorUp.value))
	case StreamActionKeyCursorDown:
		self.terminal.CursorDown(uint(value.CursorDown.value))
	case StreamActionKeyCursorLeft:
		self.terminal.CursorLeft(uint(value.CursorLeft.value))
	case StreamActionKeyCursorRight:
		self.terminal.CursorRight(uint(value.CursorRight.value))
	case StreamActionKeyCursorPos:
		self.terminal.SetCursorPos(uint(value.CursorPos.row), uint(value.CursorPos.col))
	case StreamActionKeyCursorCol:
		// Zig: `cursor.y + 1` is a u16 addition; y < rows <= maxInt(u16),
		// so it cannot overflow.
		self.terminal.SetCursorPos(uint(self.terminal.screens.active.cursor.y+1), uint(value.CursorCol.value))
	case StreamActionKeyCursorRow:
		self.terminal.SetCursorPos(uint(value.CursorRow.value), uint(self.terminal.screens.active.cursor.x+1))
	case StreamActionKeyCursorColRelative:
		// Zig: `cursor.x + 1 +| value.value` is `(x + 1) +| value`, all u16.
		self.terminal.SetCursorPos(
			uint(self.terminal.screens.active.cursor.y+1),
			uint(saturatingAddU16(self.terminal.screens.active.cursor.x+1, value.CursorColRelative.value)),
		)
	case StreamActionKeyCursorRowRelative:
		self.terminal.SetCursorPos(
			uint(saturatingAddU16(self.terminal.screens.active.cursor.y+1, value.CursorRowRelative.value)),
			uint(self.terminal.screens.active.cursor.x+1),
		)
	case StreamActionKeyCursorStyle:
		var blink bool
		switch value.CursorStyle {
		case ANSICursorStyleDefault, ANSICursorStyleSteadyBlock, ANSICursorStyleSteadyBar, ANSICursorStyleSteadyUnderline:
			blink = false
		case ANSICursorStyleBlinkingBlock, ANSICursorStyleBlinkingBar, ANSICursorStyleBlinkingUnderline:
			blink = true
		default:
			panic("unreachable")
		}
		var style ScreenCursorStyle
		switch value.CursorStyle {
		case ANSICursorStyleDefault, ANSICursorStyleBlinkingBlock, ANSICursorStyleSteadyBlock:
			style = ScreenCursorStyleBlock
		case ANSICursorStyleBlinkingBar, ANSICursorStyleSteadyBar:
			style = ScreenCursorStyleBar
		case ANSICursorStyleBlinkingUnderline, ANSICursorStyleSteadyUnderline:
			style = ScreenCursorStyleUnderline
		default:
			panic("unreachable")
		}
		self.terminal.modes.Set(ModesModeCursorBlinking, blink)
		self.terminal.screens.active.cursor.cursor_style = style
	case StreamActionKeyEraseDisplayBelow:
		self.terminal.EraseDisplay(CSIEraseDisplayBelow, value.EraseDisplayBelow)
	case StreamActionKeyEraseDisplayAbove:
		self.terminal.EraseDisplay(CSIEraseDisplayAbove, value.EraseDisplayAbove)
	case StreamActionKeyEraseDisplayComplete:
		self.terminal.EraseDisplay(CSIEraseDisplayComplete, value.EraseDisplayComplete)
	case StreamActionKeyEraseDisplayScrollback:
		self.terminal.EraseDisplay(CSIEraseDisplayScrollback, value.EraseDisplayScrollback)
	case StreamActionKeyEraseDisplayScrollComplete:
		self.terminal.EraseDisplay(CSIEraseDisplayScrollComplete, value.EraseDisplayScrollComplete)
	case StreamActionKeyEraseLineRight:
		self.terminal.EraseLine(CSIEraseLineRight, value.EraseLineRight)
	case StreamActionKeyEraseLineLeft:
		self.terminal.EraseLine(CSIEraseLineLeft, value.EraseLineLeft)
	case StreamActionKeyEraseLineComplete:
		self.terminal.EraseLine(CSIEraseLineComplete, value.EraseLineComplete)
	case StreamActionKeyEraseLineRightUnlessPendingWrap:
		self.terminal.EraseLine(CSIEraseLineRightUnlessPendingWrap, value.EraseLineRightUnlessPendingWrap)
	case StreamActionKeyDeleteChars:
		self.terminal.DeleteChars(value.DeleteChars)
	case StreamActionKeyEraseChars:
		self.terminal.EraseChars(value.EraseChars)
	case StreamActionKeyInsertLines:
		self.terminal.InsertLines(value.InsertLines)
	case StreamActionKeyInsertBlanks:
		self.terminal.InsertBlanks(value.InsertBlanks)
	case StreamActionKeyDeleteLines:
		self.terminal.DeleteLines(value.DeleteLines)
	case StreamActionKeyScrollUp:
		if err := self.terminal.ScrollUp(value.ScrollUp); err != nil {
			return err
		}
	case StreamActionKeyScrollDown:
		self.terminal.ScrollDown(value.ScrollDown)
	case StreamActionKeyHorizontalTab:
		self.horizontalTab(value.HorizontalTab)
	case StreamActionKeyHorizontalTabBack:
		self.horizontalTabBack(value.HorizontalTabBack)
	case StreamActionKeyTabClearCurrent:
		self.terminal.TabClear(CSITabClearCurrent)
	case StreamActionKeyTabClearAll:
		self.terminal.TabClear(CSITabClearAll)
	case StreamActionKeyTabSet:
		self.terminal.TabSet()
	case StreamActionKeyTabReset:
		self.terminal.TabReset()
	case StreamActionKeySetMode:
		if err := self.setMode(value.SetMode.mode, true); err != nil {
			return err
		}
	case StreamActionKeyResetMode:
		if err := self.setMode(value.ResetMode.mode, false); err != nil {
			return err
		}
	case StreamActionKeySaveMode:
		self.terminal.modes.Save(value.SaveMode.mode)
	case StreamActionKeyRestoreMode:
		v := self.terminal.modes.Restore(value.RestoreMode.mode)
		if err := self.setMode(value.RestoreMode.mode, v); err != nil {
			return err
		}
	case StreamActionKeyTopAndBottomMargin:
		self.terminal.SetTopAndBottomMargin(uint(value.TopAndBottomMargin.top_left), uint(value.TopAndBottomMargin.bottom_right))
	case StreamActionKeyLeftAndRightMargin:
		self.terminal.SetLeftAndRightMargin(uint(value.LeftAndRightMargin.top_left), uint(value.LeftAndRightMargin.bottom_right))
	case StreamActionKeyLeftAndRightMarginAmbiguous:
		if self.terminal.modes.Get(ModesModeEnableLeftAndRightMargin) {
			self.terminal.SetLeftAndRightMargin(0, 0)
		} else {
			self.terminal.SaveCursor()
		}
	case StreamActionKeySaveCursor:
		self.terminal.SaveCursor()
	case StreamActionKeyRestoreCursor:
		self.terminal.RestoreCursor()
	case StreamActionKeyInvokeCharset:
		self.terminal.InvokeCharset(value.InvokeCharset.bank, value.InvokeCharset.charset, value.InvokeCharset.locking)
	case StreamActionKeyConfigureCharset:
		self.terminal.ConfigureCharset(value.ConfigureCharset.slot, value.ConfigureCharset.charset)
	case StreamActionKeySetAttribute:
		switch value.SetAttribute.Tag {
		case SGRAttributeTagUnknown:
		default:
			// Zig: `catch {}`.
			_ = self.terminal.SetAttribute(value.SetAttribute)
		}
	case StreamActionKeyProtectedModeOff:
		self.terminal.SetProtectedMode(ANSIProtectedModeOff)
	case StreamActionKeyProtectedModeIso:
		self.terminal.SetProtectedMode(ANSIProtectedModeIso)
	case StreamActionKeyProtectedModeDec:
		self.terminal.SetProtectedMode(ANSIProtectedModeDec)
	case StreamActionKeyMouseShiftCapture:
		if value.MouseShiftCapture {
			self.terminal.flags.mouse_shift_capture = TerminalFlagsMouseShiftCaptureTrue
		} else {
			self.terminal.flags.mouse_shift_capture = TerminalFlagsMouseShiftCaptureFalse
		}
	case StreamActionKeyKittyKeyboardPush:
		self.terminal.screens.active.kitty_keyboard.Push(value.KittyKeyboardPush.flags)
	case StreamActionKeyKittyKeyboardPop:
		// Zig: @intCast(value), u16 to usize, which always fits.
		self.terminal.screens.active.kitty_keyboard.Pop(uint(value.KittyKeyboardPop))
	case StreamActionKeyKittyKeyboardSet:
		self.terminal.screens.active.kitty_keyboard.Set(KittyKeySetModeSet, value.KittyKeyboardSet.flags)
	case StreamActionKeyKittyKeyboardSetOr:
		self.terminal.screens.active.kitty_keyboard.Set(KittyKeySetModeOr, value.KittyKeyboardSetOr.flags)
	case StreamActionKeyKittyKeyboardSetNot:
		self.terminal.screens.active.kitty_keyboard.Set(KittyKeySetModeNot, value.KittyKeyboardSetNot.flags)
	case StreamActionKeyModifyKeyFormat:
		self.terminal.flags.modify_other_keys_2 = false
		switch value.ModifyKeyFormat {
		case ANSIModifyKeyFormatOtherKeysNumeric:
			self.terminal.flags.modify_other_keys_2 = true
		default:
		}
	case StreamActionKeyActiveStatusDisplay:
		self.terminal.status_display = value.ActiveStatusDisplay
	case StreamActionKeyDecaln:
		if err := self.terminal.Decaln(); err != nil {
			return err
		}
	case StreamActionKeyFullReset:
		self.terminal.FullReset()
	case StreamActionKeyStartHyperlink:
		if err := self.terminal.screens.active.StartHyperlink(value.StartHyperlink.uri, value.StartHyperlink.id); err != nil {
			return err
		}
	case StreamActionKeyEndHyperlink:
		self.terminal.screens.active.EndHyperlink()
	case StreamActionKeySemanticPrompt:
		if err := self.terminal.SemanticPrompt(value.SemanticPrompt); err != nil {
			return err
		}
	case StreamActionKeyMouseShape:
		self.terminal.mouse_shape = value.MouseShape
	case StreamActionKeyColorOperation:
		if err := self.colorOperation(value.ColorOperation.op, &value.ColorOperation.requests); err != nil {
			return err
		}
	case StreamActionKeyKittyColorReport:
		if err := self.kittyColorOperation(value.KittyColorReport); err != nil {
			return err
		}

	// No supported DCS commands have any terminal-modifying effects,
	// but they may in the future. For now we just ignore it.
	case StreamActionKeyDcsHook,
		StreamActionKeyDcsPut,
		StreamActionKeyDcsUnhook:

	// APC can modify terminal state (Kitty graphics) but we don't
	// currently support it in the readonly stream.
	case StreamActionKeyApcStart,
		StreamActionKeyApcEnd,
		StreamActionKeyApcPut:

	// Have no terminal-modifying effect
	case StreamActionKeyBell,
		StreamActionKeyEnquiry,
		StreamActionKeyRequestMode,
		StreamActionKeyRequestModeUnknown,
		StreamActionKeySizeReport,
		StreamActionKeyXtversion,
		StreamActionKeyDeviceAttributes,
		StreamActionKeyDeviceStatus,
		StreamActionKeyKittyKeyboardQuery,
		StreamActionKeyWindowTitle,
		StreamActionKeyReportPwd,
		StreamActionKeyShowDesktopNotification,
		StreamActionKeyProgressReport,
		StreamActionKeyClipboardContents,
		StreamActionKeyTitlePush,
		StreamActionKeyTitlePop:

	default:
		// Zig: the switch is exhaustive over Action.Tag.
		panic("unreachable")
	}
	return nil
}

// horizontalTab is stream_readonly.Handler.horizontalTab.
//
// Zig: inline fn horizontalTab(self: *Handler, count: u16) void.
func (self *StreamReadonlyHandler) horizontalTab(count uint16) {
	for range count {
		x := self.terminal.screens.active.cursor.x
		self.terminal.HorizontalTab()
		if x == self.terminal.screens.active.cursor.x {
			break
		}
	}
}

// horizontalTabBack is stream_readonly.Handler.horizontalTabBack.
//
// Zig: inline fn horizontalTabBack(self: *Handler, count: u16) void.
func (self *StreamReadonlyHandler) horizontalTabBack(count uint16) {
	for range count {
		x := self.terminal.screens.active.cursor.x
		self.terminal.HorizontalTabBack()
		if x == self.terminal.screens.active.cursor.x {
			break
		}
	}
}

// setMode is stream_readonly.Handler.setMode.
//
// Zig: fn setMode(self: *Handler, mode: modes.Mode, enabled: bool) !void.
// The inferred error set is that of Terminal.switchScreenMode and
// Terminal.deccolm.
func (self *StreamReadonlyHandler) setMode(mode ModesMode, enabled bool) error {
	// Set the mode on the terminal
	self.terminal.modes.Set(mode, enabled)

	// Some modes require additional processing
	switch mode {
	case ModesModeAutorepeat,
		ModesModeReverseColors:

	case ModesModeOrigin:
		self.terminal.SetCursorPos(1, 1)

	case ModesModeEnableLeftAndRightMargin:
		if !enabled {
			self.terminal.scrolling_region.left = 0
			// Zig: `cols - 1` on u16 traps when cols is 0; Terminal.init
			// asserts cols > 0.
			self.terminal.scrolling_region.right = self.terminal.cols - 1
		}

	case ModesModeAltScreenLegacy:
		if err := self.terminal.SwitchScreenMode(TerminalSwitchScreenMode47, enabled); err != nil {
			return err
		}
	case ModesModeAltScreen:
		if err := self.terminal.SwitchScreenMode(TerminalSwitchScreenMode1047, enabled); err != nil {
			return err
		}
	case ModesModeAltScreenSaveCursorClearEnter:
		if err := self.terminal.SwitchScreenMode(TerminalSwitchScreenMode1049, enabled); err != nil {
			return err
		}

	case ModesModeSaveCursor:
		if enabled {
			self.terminal.SaveCursor()
		} else {
			self.terminal.RestoreCursor()
		}

	case ModesModeEnableMode3:

	case ModesMode132Column:
		// Zig: deccolm(self.terminal.screens.active.alloc, ...). The Go
		// Deccolm has no allocator parameter.
		var m TerminalDeccolmMode
		if enabled {
			m = TerminalDeccolmMode132Cols
		} else {
			m = TerminalDeccolmMode80Cols
		}
		if err := self.terminal.Deccolm(m); err != nil {
			return err
		}

	case ModesModeSynchronizedOutput,
		ModesModeLinefeed,
		ModesModeInBandSizeReports,
		ModesModeFocusEvent:

	case ModesModeMouseEventX10:
		if enabled {
			self.terminal.flags.mouse_event = TerminalMouseEventsX10
		} else {
			self.terminal.flags.mouse_event = TerminalMouseEventsNone
		}
	case ModesModeMouseEventNormal:
		if enabled {
			self.terminal.flags.mouse_event = TerminalMouseEventsNormal
		} else {
			self.terminal.flags.mouse_event = TerminalMouseEventsNone
		}
	case ModesModeMouseEventButton:
		if enabled {
			self.terminal.flags.mouse_event = TerminalMouseEventsButton
		} else {
			self.terminal.flags.mouse_event = TerminalMouseEventsNone
		}
	case ModesModeMouseEventAny:
		if enabled {
			self.terminal.flags.mouse_event = TerminalMouseEventsAny
		} else {
			self.terminal.flags.mouse_event = TerminalMouseEventsNone
		}

	case ModesModeMouseFormatUtf8:
		if enabled {
			self.terminal.flags.mouse_format = TerminalMouseFormatUtf8
		} else {
			self.terminal.flags.mouse_format = TerminalMouseFormatX10
		}
	case ModesModeMouseFormatSgr:
		if enabled {
			self.terminal.flags.mouse_format = TerminalMouseFormatSgr
		} else {
			self.terminal.flags.mouse_format = TerminalMouseFormatX10
		}
	case ModesModeMouseFormatUrxvt:
		if enabled {
			self.terminal.flags.mouse_format = TerminalMouseFormatUrxvt
		} else {
			self.terminal.flags.mouse_format = TerminalMouseFormatX10
		}
	case ModesModeMouseFormatSgrPixels:
		if enabled {
			self.terminal.flags.mouse_format = TerminalMouseFormatSgrPixels
		} else {
			self.terminal.flags.mouse_format = TerminalMouseFormatX10
		}

	default:
	}
	return nil
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
	_ = op
	if requests.count() == 0 {
		return nil
	}

	it := requests.constIterator(0)
	for req := it.next(); req != nil; req = it.next() {
		switch req.Tag {
		case OSCParsersColorRequestSet:
			set := req.Set
			switch set.target.Tag {
			case OSCParsersColorTargetPalette:
				i := set.target.Palette
				self.terminal.flags.dirty.palette = true
				self.terminal.colors.palette.Set(i, set.color)
			case OSCParsersColorTargetDynamic:
				switch set.target.Dynamic {
				case ColorDynamicForeground:
					self.terminal.colors.foreground.Set(set.color)
				case ColorDynamicBackground:
					self.terminal.colors.background.Set(set.color)
				case ColorDynamicCursor:
					self.terminal.colors.cursor.Set(set.color)
				case ColorDynamicPointerForeground,
					ColorDynamicPointerBackground,
					ColorDynamicTektronixForeground,
					ColorDynamicTektronixBackground,
					ColorDynamicHighlightBackground,
					ColorDynamicTektronixCursor,
					ColorDynamicHighlightForeground:
				default:
					panic("unreachable")
				}
			case OSCParsersColorTargetSpecial:
			default:
				panic("unreachable")
			}

		case OSCParsersColorRequestReset:
			target := req.Reset
			switch target.Tag {
			case OSCParsersColorTargetPalette:
				i := target.Palette
				self.terminal.flags.dirty.palette = true
				self.terminal.colors.palette.Reset(i)
			case OSCParsersColorTargetDynamic:
				switch target.Dynamic {
				case ColorDynamicForeground:
					self.terminal.colors.foreground.Reset()
				case ColorDynamicBackground:
					self.terminal.colors.background.Reset()
				case ColorDynamicCursor:
					self.terminal.colors.cursor.Reset()
				case ColorDynamicPointerForeground,
					ColorDynamicPointerBackground,
					ColorDynamicTektronixForeground,
					ColorDynamicTektronixBackground,
					ColorDynamicHighlightBackground,
					ColorDynamicTektronixCursor,
					ColorDynamicHighlightForeground:
				default:
					panic("unreachable")
				}
			case OSCParsersColorTargetSpecial:
			default:
				panic("unreachable")
			}

		case OSCParsersColorRequestResetPalette:
			mask := &self.terminal.colors.palette.mask
			mask_it := mask.iterator()
			for i := mask_it.Next(); i.Valid; i = mask_it.Next() {
				self.terminal.flags.dirty.palette = true
				// Zig: @intCast(i), usize to u8; i < 256.
				self.terminal.colors.palette.Reset(uint8(i.Value))
			}
			*mask = StaticBitSet256InitEmpty()

		case OSCParsersColorRequestQuery,
			OSCParsersColorRequestResetSpecial:

		default:
			panic("unreachable")
		}
	}
	return nil
}

// kittyColorOperation is stream_readonly.Handler.kittyColorOperation.
//
// Zig: fn kittyColorOperation(self: *Handler, request: kitty_color.OSC)
// !void. Nothing in the body can fail; the inferred error set is empty and
// the Go result is always nil.
func (self *StreamReadonlyHandler) kittyColorOperation(request KittyColorOSC) error {
	for _, item := range request.list {
		switch item.Tag {
		case KittyColorOSCRequestSet:
			v := item.Set
			switch v.key.Tag {
			case KittyColorKindPalette:
				palette := v.key.Palette
				self.terminal.flags.dirty.palette = true
				self.terminal.colors.palette.Set(palette, v.color)
			case KittyColorKindSpecial:
				switch v.key.Special {
				case KittyColorSpecialForeground:
					self.terminal.colors.foreground.Set(v.color)
				case KittyColorSpecialBackground:
					self.terminal.colors.background.Set(v.color)
				case KittyColorSpecialCursor:
					self.terminal.colors.cursor.Set(v.color)
				default:
				}
			default:
				panic("unreachable")
			}
		case KittyColorOSCRequestReset:
			key := item.Reset
			switch key.Tag {
			case KittyColorKindPalette:
				palette := key.Palette
				self.terminal.flags.dirty.palette = true
				self.terminal.colors.palette.Reset(palette)
			case KittyColorKindSpecial:
				switch key.Special {
				case KittyColorSpecialForeground:
					self.terminal.colors.foreground.Reset()
				case KittyColorSpecialBackground:
					self.terminal.colors.background.Reset()
				case KittyColorSpecialCursor:
					self.terminal.colors.cursor.Reset()
				default:
				}
			default:
				panic("unreachable")
			}
		case KittyColorOSCRequestQuery:
		default:
			panic("unreachable")
		}
	}
	return nil
}
