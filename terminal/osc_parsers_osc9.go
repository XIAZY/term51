// Port of src/terminal/osc/parsers/osc9.zig (Ghostty v1.3.1).
//
// File naming: `osc/parsers/osc9.zig` becomes
// `osc_parsers_osc9.go` in package terminal, and its
// declarations take the prefix `OSCParsersOsc9`. The file has no C ABI
// declarations.
//
// Zig's labeled block `conemu: { ... }` and its `break :conemu` are a Go
// block followed by the label `conemu` and `goto conemu` (see
// docs/mapping.md).
//
// std.fmt.parseUnsigned has no Go standard-library equivalent that accepts
// the same inputs (Zig accepts '_' between digits); it is ported as
// oscParsersOsc9ParseUnsigned below.

package terminal

// OSCParsersOsc9Parse is osc.parsers.osc9.parse: parse OSC 9, which
// could be an iTerm2 notification or a ConEmu extension.
//
// Zig: `parse(parser: *Parser, _: ?u8) ?*Command`. The result is nil for
// Zig's null; otherwise it points at the parser's own command.
func OSCParsersOsc9Parse(parser *OSCParser, _ Optional[uint8]) *OSCCommand {
	writer := parser.writer
	if writer == nil {
		parser.state = OSCParserStateInvalid
		return nil
	}

	// Check first to see if this is a ConEmu OSC
	// https://conemu.github.io/en/AnsiEscapeCodes.html#ConEmu_specific_OSC
	{
		data := writer.buffered()
		if len(data) == 0 {
			goto conemu
		}
		switch data[0] {
		// Check for OSC 9;1 9;10 9;11 9;12
		case '1':
			if len(data) < 2 {
				goto conemu
			}
			switch data[1] {
			// OSC 9;1 sleep
			case ';':
				var duration_ms uint16
				if num, ok := oscParsersOsc9ParseUnsigned(data[2:], 16); ok {
					duration_ms = uint16(min(num, 10_000))
				} else {
					duration_ms = 100
				}
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuSleep,
					ConemuSleep: OSCCommandConemuSleepPayload{
						DurationMs: duration_ms,
					},
				}
				return &parser.command
			// OSC 9;10 xterm keyboard and output emulation
			case '0':
				if len(data) == 2 {
					parser.command = OSCCommand{
						Tag: OSCCommandKeyConemuXtermEmulation,
						ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{
							Keyboard: Some(true),
							Output:   Some(true),
						},
					}
					return &parser.command
				}
				if len(data) < 4 {
					goto conemu
				}
				if data[2] != ';' {
					goto conemu
				}
				switch data[3] {
				case '0':
					parser.command = OSCCommand{
						Tag: OSCCommandKeyConemuXtermEmulation,
						ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{
							Keyboard: Some(false),
							Output:   Some(false),
						},
					}
					return &parser.command
				case '1':
					parser.command = OSCCommand{
						Tag: OSCCommandKeyConemuXtermEmulation,
						ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{
							Keyboard: Some(true),
							Output:   Some(true),
						},
					}
					return &parser.command
				case '2':
					parser.command = OSCCommand{
						Tag: OSCCommandKeyConemuXtermEmulation,
						ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{
							Keyboard: None[bool](),
							Output:   Some(false),
						},
					}
					return &parser.command
				case '3':
					parser.command = OSCCommand{
						Tag: OSCCommandKeyConemuXtermEmulation,
						ConemuXtermEmulation: OSCCommandConemuXtermEmulationPayload{
							Keyboard: None[bool](),
							Output:   Some(true),
						},
					}
					return &parser.command
				default:
					goto conemu
				}
			// OSC 9;11 comment
			case '1':
				if len(data) < 3 {
					goto conemu
				}
				if data[2] != ';' {
					goto conemu
				}
				if writer.writeByte(0) != nil {
					parser.state = OSCParserStateInvalid
					return nil
				}
				data = writer.buffered()
				parser.command = OSCCommand{
					Tag:           OSCCommandKeyConemuComment,
					ConemuComment: data[3 : len(data)-1],
				}
				return &parser.command
			// OSC 9;12 mark prompt start
			case '2':
				parser.command = OSCCommand{
					Tag:            OSCCommandKeySemanticPrompt,
					SemanticPrompt: OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt),
				}
				return &parser.command
			default:
				goto conemu
			}
		// OSC 9;2 show message box
		case '2':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if writer.writeByte(0) != nil {
				parser.state = OSCParserStateInvalid
				return nil
			}
			data = writer.buffered()
			parser.command = OSCCommand{
				Tag:                  OSCCommandKeyConemuShowMessageBox,
				ConemuShowMessageBox: data[2 : len(data)-1],
			}
			return &parser.command
		// OSC 9;3 change tab title
		case '3':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if len(data) == 2 {
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuChangeTabTitle,
					ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{
						Tag: OSCCommandConemuChangeTabTitlePayloadReset,
					},
				}
				return &parser.command
			}
			if writer.writeByte(0) != nil {
				parser.state = OSCParserStateInvalid
				return nil
			}
			data = writer.buffered()
			parser.command = OSCCommand{
				Tag: OSCCommandKeyConemuChangeTabTitle,
				ConemuChangeTabTitle: OSCCommandConemuChangeTabTitlePayload{
					Tag:   OSCCommandConemuChangeTabTitlePayloadValue,
					Value: data[2 : len(data)-1],
				},
			}
			return &parser.command
		// OSC 9;4 progress report
		case '4':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if len(data) < 3 {
				goto conemu
			}
			switch data[2] {
			case '0':
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuProgressReport,
					ConemuProgressReport: OSCCommandProgressReport{
						State: OSCCommandProgressReportStateRemove,
					},
				}
			case '1':
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuProgressReport,
					ConemuProgressReport: OSCCommandProgressReport{
						State:    OSCCommandProgressReportStateSet,
						Progress: Some[uint8](0),
					},
				}
			case '2':
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuProgressReport,
					ConemuProgressReport: OSCCommandProgressReport{
						State: OSCCommandProgressReportStateError,
					},
				}
			case '3':
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuProgressReport,
					ConemuProgressReport: OSCCommandProgressReport{
						State: OSCCommandProgressReportStateIndeterminate,
					},
				}
			case '4':
				parser.command = OSCCommand{
					Tag: OSCCommandKeyConemuProgressReport,
					ConemuProgressReport: OSCCommandProgressReport{
						State: OSCCommandProgressReportStatePause,
					},
				}
			default:
				goto conemu
			}
			switch parser.command.ConemuProgressReport.State {
			case OSCCommandProgressReportStateRemove, OSCCommandProgressReportStateIndeterminate:
			case OSCCommandProgressReportStateSet, OSCCommandProgressReportStateError, OSCCommandProgressReportStatePause:
				// progress: (Zig's `break :progress` is the `break` out of
				// this switch case)
				if len(data) < 4 {
					break
				}
				if data[3] != ';' {
					break
				}
				// parse the progress value
				//
				// Zig: std.fmt.parseUnsigned(usize, ...). usize is 64 bits
				// here whatever the target: the recorded corpus comes from
				// a 64-bit build.
				if v, ok := oscParsersOsc9ParseUnsigned(data[4:], 64); ok {
					parser.command.ConemuProgressReport.Progress = Some(uint8(max(0, min(v, 100))))
				} else {
					parser.command.ConemuProgressReport.Progress = None[uint8]()
				}
			}
			return &parser.command
		// OSC 9;5 wait for input
		case '5':
			parser.command = OSCCommand{Tag: OSCCommandKeyConemuWaitInput}
			return &parser.command
		// OSC 9;6 guimacro
		case '6':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if writer.writeByte(0) != nil {
				parser.state = OSCParserStateInvalid
				return nil
			}
			data = writer.buffered()
			parser.command = OSCCommand{
				Tag:            OSCCommandKeyConemuGuimacro,
				ConemuGuimacro: data[2 : len(data)-1],
			}
			return &parser.command
		// OSC 9;7 run process
		case '7':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if writer.writeByte(0) != nil {
				parser.state = OSCParserStateInvalid
				return nil
			}
			data = writer.buffered()
			parser.command = OSCCommand{
				Tag:              OSCCommandKeyConemuRunProcess,
				ConemuRunProcess: data[2 : len(data)-1],
			}
			return &parser.command
		// OSC 9;8 output environment variable
		case '8':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if writer.writeByte(0) != nil {
				parser.state = OSCParserStateInvalid
				return nil
			}
			data = writer.buffered()
			parser.command = OSCCommand{
				Tag:                             OSCCommandKeyConemuOutputEnvironmentVariable,
				ConemuOutputEnvironmentVariable: data[2 : len(data)-1],
			}
			return &parser.command
		// OSC 9;9 current working directory
		case '9':
			if len(data) < 2 {
				goto conemu
			}
			if data[1] != ';' {
				goto conemu
			}
			if writer.writeByte(0) != nil {
				parser.state = OSCParserStateInvalid
				return nil
			}
			data = writer.buffered()
			parser.command = OSCCommand{
				Tag: OSCCommandKeyReportPwd,
				ReportPwd: OSCCommandReportPwdPayload{
					Value: data[2 : len(data)-1],
				},
			}
			return &parser.command
		default:
			goto conemu
		}
	}
conemu:

	// If it's not a ConEmu OSC, it's an iTerm2 notification

	if writer.writeByte(0) != nil {
		parser.state = OSCParserStateInvalid
		return nil
	}
	data := writer.buffered()
	parser.command = OSCCommand{
		Tag: OSCCommandKeyShowDesktopNotification,
		ShowDesktopNotification: OSCCommandShowDesktopNotificationPayload{
			Title: []uint8{0}[:0], // Zig: "", a [:0]const u8 literal
			Body:  data[0 : len(data)-1],
		},
	}
	return &parser.command
}

// oscParsersOsc9ParseUnsigned is std.fmt.parseUnsigned(T, buf, 10) for an
// unsigned T of the given bit width (at most 64), as osc9.zig calls it.
//
// Not a function of osc9.zig: it is the port of the standard-library
// function (std/fmt.zig parseIntWithSign with sign .pos, base 10). Every
// caller in osc9.zig discards the error (`catch`, `else |_|`), so only
// success is reported: ok is false where Zig returns error.InvalidCharacter
// or error.Overflow. Like Zig, it accepts leading zeros and '_' between
// digits, and rejects a sign, an empty string and a leading or trailing
// '_'.
func oscParsersOsc9ParseUnsigned(buf []uint8, bits uint) (uint64, bool) {
	if len(buf) == 0 {
		return 0, false
	}
	if buf[0] == '_' || buf[len(buf)-1] == '_' {
		return 0, false
	}
	maxValue := uint64(1)<<bits - 1
	if bits == 64 {
		maxValue = ^uint64(0)
	}
	var accumulate uint64
	for _, c := range buf {
		if c == '_' {
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		digit := uint64(c - '0')
		if accumulate != 0 {
			if accumulate > maxValue/10 {
				return 0, false
			}
			accumulate *= 10
		}
		if accumulate > maxValue-digit {
			return 0, false
		}
		accumulate += digit
	}
	return accumulate, true
}
