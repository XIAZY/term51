# The public API

Rule (project owner, 2026-10-06): the same features and behaviour as upstream
libghostty-vt, in the form a Go user expects; inside, the code stays
upstream's unless it makes no sense in Go.

Upstream's public API is `src/lib_vt.zig` (Ghostty v1.3.1). This file records
how the Go module presents it.

- Package `vt` and one package per namespace (`vt/osc`, `vt/sgr`, ...) are
  generated from package `terminal` by `scripts/genvt` (aliases, constants
  and thin wrappers; nothing is copied).
- The fields of the types below are exported by `scripts/exportfields`
  with the list `scripts/exportfields/types.txt`.
- Functions that produce output take an `io.Writer` (`terminal/api_writer.go`).
- The cell, the row and the packed mode set are read through methods
  (`terminal/api_accessors.go`): upstream packs them into 64-bit values and
  bits, and the port's storage for them may change, so their fields are not
  a promise.

## 1. Whose fields are exported

Zig has no private fields, so "public upstream" is the language default, not
a design choice. In Go every exported field is a promise, so fields are
exported by need: the types a user reads or fills in. Field names change
from `snake_case` to Go names by tool (`cursor.x` -> `Cursor.X`).

### Exported

| Group | Types (current Go names in package `terminal`) |
|---|---|
| Options a user fills in | `TerminalOptions`, `TerminalColors`, `ScreenOptions`, `ScreenResize`, `PageListResize`, `PageListClone`, `ScreenCursorCopyOptions`, `ScreenSelectionString`, `ScreenSelectLine`, `ScreenDumpStringOptions`, `FormatterOptions`, `FormatterTerminalFormatterExtra`, `FormatterScreenFormatterExtra`, `InputKeyEncodeOptions`, `InputPasteOptions` |
| Terminal state | `Terminal`, `TerminalFlags`, `TerminalDirty`, `TerminalScrollingRegion`, `ScreenSet`, `Screen`, `ScreenDirty`, `ScreenCursor`, `ScreenSavedCursor`, `ScreenCharsetState`, `ScreenSemanticPrompt`, `ScreenPromptClickMove` |
| Modes and keyboard state | `ModesModeState`, `ModesModeTag`, `KittyKeyFlags`, `DeviceStatusTag` |
| The render snapshot | `RenderState`, `RenderStateColors`, `RenderStateCursor`, `RenderStateCursorViewport`, `RenderStateRow`, `RenderStateCell`, `RenderStateHighlight`, `RenderStateStringMapParam` |
| Cells, styles, colours | `PageCellRGB`, `Style`, `StyleFlags`, `StyleFg`, `ConfigColor`, `ColorRGB`, `ColorDynamicRGB`, `ColorDynamicPalette`, `Hyperlink` |
| Positions and selection | `PointCoordinate`, `PageListPin`, `PageListCell`, `PageListScrollbar`, `Selection`, `SelectionBoundsUntracked`, `SelectionBoundsTracked`, `HighlightUntracked`, `HighlightTracked`, `HighlightFlattened`, `HighlightFlattenedChunk`, `StringMap`, `SearchScreenSelectedMatch` |
| What a stream handler receives | every `StreamAction...` payload, every `OSCCommand...` payload, `OSCParsersColorColoredTarget`, `OSCParsersContextSignalCommand`, `OSCParsersKittyClipboardProtocolOSC`, `OSCParsersKittyTextSizingOSC`, `OSCParsersSemanticPromptCommand`, `KittyColorOSC`, `KittyColorOSCRequestSetPayload`, `DCSCommand`, `DCSCommandXTGETTCAP`, `SGRAttributeUnknown` (their shape is settled with the speed work first) |
| Key and paste input | `InputKeyKeyEvent`, `InputKeyModsMods`, `InputKeyModsModsKeys`, `InputKeyModsModsSide` |

### Not exported (internals; reachable through methods)

| Group | Types |
|---|---|
| Storage that may be repacked (read through `api_accessors.go`) | `PageCell`, `PageCellContent`, `PageRow`, `ModesModePacked` |
| Page memory and its bookkeeping | `Page`, `PageLayout`, `PageSize`, `PageCapacity`, `PageCapacityAdjustment`, `PageList` (its own fields), `PageListList`, `PageListNode`, `PageListMemoryPool`, `BitmapAllocator...`, `HashMap...`, `RefCountedSet...`, `HyperlinkPageEntry`, `HyperlinkSetContext` |
| Iterators | `PageListPromptIterator`, `PageListCellIterator`, `PageListRowIterator`, `PageListPageIterator`, `PageListPageIteratorChunk`, `ScreenLineIterator`, `OSCParsersColorList...`, `StaticBitSet256Iterator`, `DatastructCircBufIterator` |
| Parsers and decoders | `Parser`, `OSCParser`, `SGRParser`, `UTF8Decoder`, `Stream`, `APCHandler`, `APCState`, `DCSHandler`, `DCSState`, `Transition` |
| Formatter and search machinery | `Formatter...Formatter`, `FormatterCodepointMap`, `FormatterPinMap`, `FormatterPageFormatter...`, `SearchActive`, `SearchPagelist`, `SearchScreen`, `SearchSlidingWindow`, `SearchViewport` |
| Tables and helpers | `Tabstops`, `KittyKeyFlagStack`, `InputFunctionKeysEntry`, `InputFunctionKeysKeyEntryArray`, `InputKittyEntry`, `InputKeyModsAliasEntry`, `InputKeyModsRemapSet`, `InputKeyModsRemapSetMask`, `UnicodeLutTables`, `UnicodeProperties`, `X11ColorColorMap`, `StaticBitSet24`, `StaticBitSet256`, `DatastructCircBuf`, `RenderStateCellSet`, `StreamReadonlyHandler` |

Fields whose Go name would collide with a method (`Style.underline_color`),
padding fields, and fields whose type is internal (`ScreenSet.all`,
`ScreenCharsetState.charsets`, `RenderState.selection_cache`,
`DCSCommandXTGETTCAP.data`) stay unexported; the type's methods are the
access.

## 2. Wrong arguments

Zig rules many wrong arguments out at compile time (an enum cannot hold an
undeclared value, a `u21` cannot hold a number above 0x1FFFFF) and asserts
on others; a release build of upstream does not check those assertions. In
Go:

- **An error, where the value can come from the outside world at run time.**
  A size of zero columns or rows returns `ErrZeroColCount` or
  `ErrZeroRowCount` from `TerminalInit`, `ScreenInit`, `PageListInit`,
  `Terminal.Resize` and `Screen.Resize`, before anything changes (a window
  being laid out can report a zero size).
- **A panic, where only a programming mistake produces the value:** a nil
  terminal, screen or writer; a value of an enumeration type that is not one
  of its declared constants (`EraseDisplay(99)`, `Mode(60000)`); a code
  point above 0x1FFFFF passed to `Print`; an index outside a fixed table
  (`SwitchScreen(99)`). These are the Go counterparts of values a Zig
  program cannot construct.

Everything else a caller can pass is accepted as upstream accepts it: zero
and very large counts, positions outside the screen, empty input.

## 3. Allocators and clean-up

Go allocates and frees by itself, so allocator parameters are gone. Where
upstream's behaviour depends on whether an allocator was given (the OSC
parser refuses commands that need a growing buffer when it has none), the
choice is a `bool`: `OSCParserInit(false)` is upstream's `init(null)`,
`StreamInit` builds a stream whose OSC parser cannot grow and
`StreamInitAlloc` one that can; `Terminal.VtStream` uses the second, as
upstream does. In package `vt` the growing one is `NewStream`, because that
is the name a Go user reaches for and the fixed one silently refuses the
colour and clipboard commands; the fixed one is `NewStreamFixed`.

`Deinit` methods remain and are optional for memory. Call them where they
do more than free: `Selection.Deinit` and `HighlightTracked.Deinit` stop
tracking their pins in the screen, and `Stream.Deinit` calls the handler's
`Deinit`.

