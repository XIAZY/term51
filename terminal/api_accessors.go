// Accessors for the types whose storage is internal.
//
// Upstream packs a cell and a row into one 64-bit value each and a mode set
// into bits, and reads them as fields of a packed struct. The Go port may
// store them differently (docs/post-port-optimizations.md), so their fields
// are not exported; these methods are the access, and they stay the same
// whatever the storage is. Each returns what the Zig field of the same name
// holds.

package terminal

// ContentTag is cell.content_tag: which kind of content the cell holds.
func (self PageCell) ContentTag() PageCellContentTag { return self.content_tag }

// StyleID is cell.style_id: the ID of the cell's style in its page's style
// set (the default style when the cell has no styling).
func (self PageCell) StyleID() StyleID { return self.style_id }

// Wide is cell.wide: narrow, wide, or one of the spacer cells of a wide
// character.
func (self PageCell) Wide() PageCellWide { return self.wide }

// Protected is cell.protected: whether the cell is protected from erasing
// (DECSCA).
func (self PageCell) Protected() bool { return self.protected }

// Hyperlink is cell.hyperlink: whether the cell is part of a hyperlink.
func (self PageCell) Hyperlink() bool { return self.hyperlink }

// SemanticContent is cell.semantic_content: output, input or prompt (OSC
// 133).
func (self PageCell) SemanticContent() PageCellSemanticContent { return self.semantic_content }

// ColorPalette is cell.content.color_palette: the palette index of a cell
// that is only a background colour. Meaningful when ContentTag says so.
func (self PageCell) ColorPalette() uint8 { return self.content.color_palette }

// ColorRGB is cell.content.color_rgb: the colour of a cell that is only a
// background colour. Meaningful when ContentTag says so.
func (self PageCell) ColorRGB() PageCellRGB { return self.content.color_rgb }

// Wrap is row.wrap: whether the row is soft-wrapped onto the next one.
func (self PageRow) Wrap() bool { return self.wrap }

// WrapContinuation is row.wrap_continuation: whether the row continues a
// soft-wrapped row.
func (self PageRow) WrapContinuation() bool { return self.wrap_continuation }

// Grapheme is row.grapheme: whether any cell of the row may have grapheme
// data (it can be true with none left).
func (self PageRow) Grapheme() bool { return self.grapheme }

// Styled is row.styled: whether any cell of the row may have a style.
func (self PageRow) Styled() bool { return self.styled }

// Hyperlink is row.hyperlink: whether any cell of the row may be part of a
// hyperlink.
func (self PageRow) Hyperlink() bool { return self.hyperlink }

// SemanticPrompt is row.semantic_prompt: the row's part in a shell prompt
// (OSC 133).
func (self PageRow) SemanticPrompt() PageRowSemanticPrompt { return self.semantic_prompt }

// KittyVirtualPlaceholder is row.kitty_virtual_placeholder.
func (self PageRow) KittyVirtualPlaceholder() bool { return self.kitty_virtual_placeholder }

// Dirty is row.dirty: whether the row changed since a renderer last cleared
// the flag.
func (self PageRow) Dirty() bool { return self.dirty }

// Get reports whether mode is set in the packed mode set.
func (self *ModesModePacked) Get(mode ModesMode) bool {
	return self.get(modesModeEntryIndex(mode))
}

// Set sets or resets mode in the packed mode set.
func (self *ModesModePacked) Set(mode ModesMode, value bool) {
	self.set(modesModeEntryIndex(mode), value)
}
