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
// (stream.go): `Vt(action StreamActionKey, value *StreamAction) error` and
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
		if err := self.terminal.Print(value.Print.Cp); err != nil {
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
		self.terminal.CursorUp(uint(value.CursorUp.Value))
	case StreamActionKeyCursorDown:
		self.terminal.CursorDown(uint(value.CursorDown.Value))
	case StreamActionKeyCursorLeft:
		self.terminal.CursorLeft(uint(value.CursorLeft.Value))
	case StreamActionKeyCursorRight:
		self.terminal.CursorRight(uint(value.CursorRight.Value))
	case StreamActionKeyCursorPos:
		self.terminal.SetCursorPos(uint(value.CursorPos.Row), uint(value.CursorPos.Col))
	case StreamActionKeyCursorCol:
		// Zig: `cursor.y + 1` is a u16 addition; y < rows <= maxInt(u16),
		// so it cannot overflow.
		self.terminal.SetCursorPos(uint(self.terminal.Screens.Active.Cursor.Y+1), uint(value.CursorCol.Value))
	case StreamActionKeyCursorRow:
		self.terminal.SetCursorPos(uint(value.CursorRow.Value), uint(self.terminal.Screens.Active.Cursor.X+1))
	case StreamActionKeyCursorColRelative:
		// Zig: `cursor.x + 1 +| value.value` is `(x + 1) +| value`, all u16.
		self.terminal.SetCursorPos(
			uint(self.terminal.Screens.Active.Cursor.Y+1),
			uint(saturatingAddU16(self.terminal.Screens.Active.Cursor.X+1, value.CursorColRelative.Value)),
		)
	case StreamActionKeyCursorRowRelative:
		self.terminal.SetCursorPos(
			uint(saturatingAddU16(self.terminal.Screens.Active.Cursor.Y+1, value.CursorRowRelative.Value)),
			uint(self.terminal.Screens.Active.Cursor.X+1),
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
		self.terminal.Modes.Set(ModesModeCursorBlinking, blink)
		self.terminal.Screens.Active.Cursor.CursorStyle = style
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
		if err := self.setMode(value.SetMode.Mode, true); err != nil {
			return err
		}
	case StreamActionKeyResetMode:
		if err := self.setMode(value.ResetMode.Mode, false); err != nil {
			return err
		}
	case StreamActionKeySaveMode:
		self.terminal.Modes.Save(value.SaveMode.Mode)
	case StreamActionKeyRestoreMode:
		v := self.terminal.Modes.Restore(value.RestoreMode.Mode)
		if err := self.setMode(value.RestoreMode.Mode, v); err != nil {
			return err
		}
	case StreamActionKeyTopAndBottomMargin:
		self.terminal.SetTopAndBottomMargin(uint(value.TopAndBottomMargin.TopLeft), uint(value.TopAndBottomMargin.BottomRight))
	case StreamActionKeyLeftAndRightMargin:
		self.terminal.SetLeftAndRightMargin(uint(value.LeftAndRightMargin.TopLeft), uint(value.LeftAndRightMargin.BottomRight))
	case StreamActionKeyLeftAndRightMarginAmbiguous:
		if self.terminal.Modes.Get(ModesModeEnableLeftAndRightMargin) {
			self.terminal.SetLeftAndRightMargin(0, 0)
		} else {
			self.terminal.SaveCursor()
		}
	case StreamActionKeySaveCursor:
		self.terminal.SaveCursor()
	case StreamActionKeyRestoreCursor:
		self.terminal.RestoreCursor()
	case StreamActionKeyInvokeCharset:
		self.terminal.InvokeCharset(value.InvokeCharset.Bank, value.InvokeCharset.Charset, value.InvokeCharset.Locking)
	case StreamActionKeyConfigureCharset:
		self.terminal.ConfigureCharset(value.ConfigureCharset.Slot, value.ConfigureCharset.Charset)
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
			self.terminal.Flags.MouseShiftCapture = TerminalFlagsMouseShiftCaptureTrue
		} else {
			self.terminal.Flags.MouseShiftCapture = TerminalFlagsMouseShiftCaptureFalse
		}
	case StreamActionKeyKittyKeyboardPush:
		self.terminal.Screens.Active.KittyKeyboard.Push(value.KittyKeyboardPush.Flags)
	case StreamActionKeyKittyKeyboardPop:
		// Zig: @intCast(value), u16 to usize, which always fits.
		self.terminal.Screens.Active.KittyKeyboard.Pop(uint(value.KittyKeyboardPop))
	case StreamActionKeyKittyKeyboardSet:
		self.terminal.Screens.Active.KittyKeyboard.Set(KittyKeySetModeSet, value.KittyKeyboardSet.Flags)
	case StreamActionKeyKittyKeyboardSetOr:
		self.terminal.Screens.Active.KittyKeyboard.Set(KittyKeySetModeOr, value.KittyKeyboardSetOr.Flags)
	case StreamActionKeyKittyKeyboardSetNot:
		self.terminal.Screens.Active.KittyKeyboard.Set(KittyKeySetModeNot, value.KittyKeyboardSetNot.Flags)
	case StreamActionKeyModifyKeyFormat:
		self.terminal.Flags.ModifyOtherKeys2 = false
		switch value.ModifyKeyFormat {
		case ANSIModifyKeyFormatOtherKeysNumeric:
			self.terminal.Flags.ModifyOtherKeys2 = true
		default:
		}
	case StreamActionKeyActiveStatusDisplay:
		self.terminal.StatusDisplay = value.ActiveStatusDisplay
	case StreamActionKeyDecaln:
		if err := self.terminal.Decaln(); err != nil {
			return err
		}
	case StreamActionKeyFullReset:
		self.terminal.FullReset()
	case StreamActionKeyStartHyperlink:
		if err := self.terminal.Screens.Active.StartHyperlink(value.StartHyperlink.URI, value.StartHyperlink.ID); err != nil {
			return err
		}
	case StreamActionKeyEndHyperlink:
		self.terminal.Screens.Active.EndHyperlink()
	case StreamActionKeySemanticPrompt:
		if err := self.terminal.SemanticPrompt(value.SemanticPrompt); err != nil {
			return err
		}
	case StreamActionKeyMouseShape:
		self.terminal.MouseShape = value.MouseShape
	case StreamActionKeyColorOperation:
		if err := self.colorOperation(value.ColorOperation.Op, &value.ColorOperation.Requests); err != nil {
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
		x := self.terminal.Screens.Active.Cursor.X
		self.terminal.HorizontalTab()
		if x == self.terminal.Screens.Active.Cursor.X {
			break
		}
	}
}

// horizontalTabBack is stream_readonly.Handler.horizontalTabBack.
//
// Zig: inline fn horizontalTabBack(self: *Handler, count: u16) void.
func (self *StreamReadonlyHandler) horizontalTabBack(count uint16) {
	for range count {
		x := self.terminal.Screens.Active.Cursor.X
		self.terminal.HorizontalTabBack()
		if x == self.terminal.Screens.Active.Cursor.X {
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
	self.terminal.Modes.Set(mode, enabled)

	// Some modes require additional processing
	switch mode {
	case ModesModeAutorepeat,
		ModesModeReverseColors:

	case ModesModeOrigin:
		self.terminal.SetCursorPos(1, 1)

	case ModesModeEnableLeftAndRightMargin:
		if !enabled {
			self.terminal.ScrollingRegion.Left = 0
			// Zig: `cols - 1` on u16 traps when cols is 0; Terminal.init
			// asserts cols > 0.
			self.terminal.ScrollingRegion.Right = self.terminal.Cols - 1
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
			self.terminal.Flags.MouseEvent = TerminalMouseEventsX10
		} else {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsNone
		}
	case ModesModeMouseEventNormal:
		if enabled {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsNormal
		} else {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsNone
		}
	case ModesModeMouseEventButton:
		if enabled {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsButton
		} else {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsNone
		}
	case ModesModeMouseEventAny:
		if enabled {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsAny
		} else {
			self.terminal.Flags.MouseEvent = TerminalMouseEventsNone
		}

	case ModesModeMouseFormatUtf8:
		if enabled {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatUtf8
		} else {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatX10
		}
	case ModesModeMouseFormatSgr:
		if enabled {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatSgr
		} else {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatX10
		}
	case ModesModeMouseFormatUrxvt:
		if enabled {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatUrxvt
		} else {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatX10
		}
	case ModesModeMouseFormatSgrPixels:
		if enabled {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatSgrPixels
		} else {
			self.terminal.Flags.MouseFormat = TerminalMouseFormatX10
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
			switch set.Target.Tag {
			case OSCParsersColorTargetPalette:
				i := set.Target.Palette
				self.terminal.Flags.Dirty.Palette = true
				self.terminal.Colors.Palette.Set(i, set.Color)
			case OSCParsersColorTargetDynamic:
				switch set.Target.Dynamic {
				case ColorDynamicForeground:
					self.terminal.Colors.Foreground.Set(set.Color)
				case ColorDynamicBackground:
					self.terminal.Colors.Background.Set(set.Color)
				case ColorDynamicCursor:
					self.terminal.Colors.Cursor.Set(set.Color)
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
				self.terminal.Flags.Dirty.Palette = true
				self.terminal.Colors.Palette.Reset(i)
			case OSCParsersColorTargetDynamic:
				switch target.Dynamic {
				case ColorDynamicForeground:
					self.terminal.Colors.Foreground.Reset()
				case ColorDynamicBackground:
					self.terminal.Colors.Background.Reset()
				case ColorDynamicCursor:
					self.terminal.Colors.Cursor.Reset()
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
			mask := &self.terminal.Colors.Palette.Mask
			mask_it := mask.iterator()
			for i := mask_it.Next(); i.Valid; i = mask_it.Next() {
				self.terminal.Flags.Dirty.Palette = true
				// Zig: @intCast(i), usize to u8; i < 256.
				self.terminal.Colors.Palette.Reset(uint8(i.Value))
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
	for _, item := range request.List {
		switch item.Tag {
		case KittyColorOSCRequestSet:
			v := item.Set
			switch v.Key.Tag {
			case KittyColorKindPalette:
				palette := v.Key.Palette
				self.terminal.Flags.Dirty.Palette = true
				self.terminal.Colors.Palette.Set(palette, v.Color)
			case KittyColorKindSpecial:
				switch v.Key.Special {
				case KittyColorSpecialForeground:
					self.terminal.Colors.Foreground.Set(v.Color)
				case KittyColorSpecialBackground:
					self.terminal.Colors.Background.Set(v.Color)
				case KittyColorSpecialCursor:
					self.terminal.Colors.Cursor.Set(v.Color)
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
				self.terminal.Flags.Dirty.Palette = true
				self.terminal.Colors.Palette.Reset(palette)
			case KittyColorKindSpecial:
				switch key.Special {
				case KittyColorSpecialForeground:
					self.terminal.Colors.Foreground.Reset()
				case KittyColorSpecialBackground:
					self.terminal.Colors.Background.Reset()
				case KittyColorSpecialCursor:
					self.terminal.Colors.Cursor.Reset()
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
