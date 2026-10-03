// Ported from src/terminal/csi.zig (Ghostty v1.3.1).
//
// The C ABI selection (lib_target / lib.Enum) is not ported; see
// docs/mapping.md.

package terminal

// CSIEraseDisplay is EraseDisplay: modes for the ED CSI command.
type CSIEraseDisplay uint8

const (
	CSIEraseDisplayBelow      CSIEraseDisplay = 0
	CSIEraseDisplayAbove      CSIEraseDisplay = 1
	CSIEraseDisplayComplete   CSIEraseDisplay = 2
	CSIEraseDisplayScrollback CSIEraseDisplay = 3

	// This is an extension added by Kitty to move the viewport into the
	// scrollback and then erase the display.
	CSIEraseDisplayScrollComplete CSIEraseDisplay = 22
)

// CSIEraseLine is EraseLine: modes for the EL CSI command.
//
// Zig: non-exhaustive so that @enumFromInt never fails since the inputs
// are user-generated. A Go uint8 type has the same property.
type CSIEraseLine uint8

const (
	CSIEraseLineRight                  CSIEraseLine = 0
	CSIEraseLineLeft                   CSIEraseLine = 1
	CSIEraseLineComplete               CSIEraseLine = 2
	CSIEraseLineRightUnlessPendingWrap CSIEraseLine = 4
)

// CSITabClear is TabClear: modes for the TBC (tab clear) command.
//
// Zig: non-exhaustive, as CSIEraseLine.
type CSITabClear uint8

const (
	CSITabClearCurrent CSITabClear = 0
	CSITabClearAll     CSITabClear = 3
)

// CSISizeReportStyle is SizeReportStyle: style formats for terminal size
// reports. Zig: lib.Enum(lib_target, ...); the Zig-target variant is a
// plain enum.
type CSISizeReportStyle uint8

const (
	// XTWINOPS
	CSISizeReportStyleCsi14T CSISizeReportStyle = iota
	CSISizeReportStyleCsi16T
	CSISizeReportStyleCsi18T
	CSISizeReportStyleCsi21T
)

// CSITitlePushPop is TitlePushPop: XTWINOPS CSI 22/23.
type CSITitlePushPop struct {
	Op    CSITitlePushPopOp
	Index uint16
}

// CSITitlePushPopOp is TitlePushPop.Op.
type CSITitlePushPopOp uint8

const (
	CSITitlePushPopOpPush CSITitlePushPopOp = iota
	CSITitlePushPopOpPop
)
