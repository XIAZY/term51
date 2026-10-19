// The public, io.Writer forms of the functions that produce output.
//
// Upstream's functions take a *std.Io.Writer. Inside the package the port
// keeps a copy of that writer (oscWriter), because output depends on its
// exact behaviour: a fixed buffer takes what fits and then fails. A Go
// caller has an io.Writer instead, so each public function here runs the
// internal one into a growing buffer and then writes that buffer to w.
// What was produced before an error is still written, as a streaming Zig
// writer would have done.

package terminal

import "io"

// writeVia runs f with a growing writer and writes what it produced to w.
// The error of f comes first; a write error is returned otherwise.
func writeVia(w io.Writer, f func(writer *oscWriter) error) error {
	aw := oscWriterAllocatingInit()
	err := f(&aw.writer)
	if b := aw.writer.buffered(); len(b) > 0 {
		if _, werr := w.Write(b); werr != nil && err == nil {
			err = werr
		}
	}
	return err
}

// Format writes the terminal as the formatter's options say (plain text, VT
// sequences or HTML) to w. It is TerminalFormatter.format.
func (self FormatterTerminalFormatter) Format(w io.Writer) error {
	return writeVia(w, self.format)
}

// Format writes the screen to w. It is ScreenFormatter.format.
func (self FormatterScreenFormatter) Format(w io.Writer) error {
	return writeVia(w, self.format)
}

// Format writes the page list to w. It is PageListFormatter.format.
func (self FormatterPageListFormatter) Format(w io.Writer) error {
	return writeVia(w, self.format)
}

// Format writes the page to w. It is PageFormatter.format.
func (self FormatterPageFormatter) Format(w io.Writer) error {
	return writeVia(w, self.format)
}

// FormatWithState writes the page to w and returns the trailing state to
// hand to the formatter of the next page. It is
// PageFormatter.formatWithState.
func (self FormatterPageFormatter) FormatWithState(w io.Writer) (FormatterPageFormatterTrailingState, error) {
	var state FormatterPageFormatterTrailingState
	err := writeVia(w, func(writer *oscWriter) (err error) {
		state, err = self.formatWithState(writer)
		return err
	})
	return state, err
}

// String writes the render state's text to w. It is RenderState.string.
func (self *RenderState) String(w io.Writer, map_ Optional[RenderStateStringMapParam]) error {
	return writeVia(w, func(writer *oscWriter) error { return self.stringW(writer, map_) })
}

// DumpString writes the screen's text to w. It is Screen.dumpString.
func (self *Screen) DumpString(w io.Writer, opts ScreenDumpStringOptions) error {
	return writeVia(w, func(writer *oscWriter) error { return self.dumpString(writer, opts) })
}

// WriteCommandLine writes the command line of a semantic prompt command to
// w. It is semantic_prompt.Command.writeCommandLine.
func (self OSCParsersSemanticPromptCommand) WriteCommandLine(w io.Writer) error {
	return writeVia(w, self.writeCommandLine)
}

// InputKeyEncodeEncode writes the bytes a key event sends to the program
// running in the terminal to w. It is key_encode.encode.
func InputKeyEncodeEncode(w io.Writer, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error {
	return writeVia(w, func(writer *oscWriter) error { return inputKeyEncodeEncode(writer, event, opts) })
}
