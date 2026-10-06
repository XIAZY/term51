# The public API (proposal)

Rule (project owner, 2026-10-06): the same features and behaviour as upstream
libghostty-vt, in the form a Go user expects; inside, the code stays
upstream's unless it makes no sense in Go.

Upstream's public API is `src/lib_vt.zig` (Ghostty v1.3.1). This file records
how the Go module presents it. Status: **proposal, nothing renamed yet**.

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
| Modes and keyboard state | `ModesModeState`, `ModesModePacked`, `ModesModeTag`, `KittyKeyFlags`, `DeviceStatusTag` |
| The render snapshot | `RenderState`, `RenderStateColors`, `RenderStateCursor`, `RenderStateCursorViewport`, `RenderStateRow`, `RenderStateCell`, `RenderStateHighlight`, `RenderStateStringMapParam` |
| Cells, styles, colours | `PageRow`, `PageCell`, `PageCellContent`, `PageCellRGB`, `Style`, `StyleFg`, `ConfigColor`, `ColorRGB`, `ColorDynamicRGB`, `ColorDynamicPalette`, `Hyperlink` |
| Positions and selection | `PointCoordinate`, `PageListPin`, `PageListCell`, `PageListScrollbar`, `Selection`, `SelectionBoundsUntracked`, `SelectionBoundsTracked`, `HighlightUntracked`, `HighlightTracked`, `HighlightFlattened`, `HighlightFlattenedChunk`, `StringMap`, `SearchScreenSelectedMatch` |
| What a stream handler receives | every `StreamAction...` payload, every `OSCCommand...` payload, `OSCParsersColorColoredTarget`, `OSCParsersContextSignalCommand`, `OSCParsersKittyClipboardProtocolOSC`, `OSCParsersKittyTextSizingOSC`, `OSCParsersSemanticPromptCommand`, `KittyColorOSC`, `KittyColorOSCRequestSetPayload`, `DCSCommand`, `DCSCommandXTGETTCAP`, `SGRAttributeUnknown` (their shape is settled with the speed work first) |
| Key and paste input | `InputKeyKeyEvent`, `InputKeyModsMods`, `InputKeyModsModsKeys`, `InputKeyModsModsSide` |

### Not exported (internals; reachable through methods)

| Group | Types |
|---|---|
| Page memory and its bookkeeping | `Page`, `PageLayout`, `PageSize`, `PageCapacity`, `PageCapacityAdjustment`, `PageList` (its own fields), `PageListList`, `PageListNode`, `PageListMemoryPool`, `BitmapAllocator...`, `HashMap...`, `RefCountedSet...`, `HyperlinkPageEntry`, `HyperlinkSetContext` |
| Iterators | `PageListPromptIterator`, `PageListCellIterator`, `PageListRowIterator`, `PageListPageIterator`, `PageListPageIteratorChunk`, `ScreenLineIterator`, `OSCParsersColorList...`, `StaticBitSet256Iterator`, `DatastructCircBufIterator` |
| Parsers and decoders | `Parser`, `OSCParser`, `SGRParser`, `UTF8Decoder`, `Stream`, `APCHandler`, `APCState`, `DCSHandler`, `DCSState`, `Transition` |
| Formatter and search machinery | `Formatter...Formatter`, `FormatterCodepointMap`, `FormatterPinMap`, `FormatterPageFormatter...`, `SearchActive`, `SearchPagelist`, `SearchScreen`, `SearchSlidingWindow`, `SearchViewport` |
| Tables and helpers | `Tabstops`, `KittyKeyFlagStack`, `InputFunctionKeysEntry`, `InputFunctionKeysKeyEntryArray`, `InputKittyEntry`, `InputKeyModsAliasEntry`, `InputKeyModsRemapSet`, `InputKeyModsRemapSetMask`, `UnicodeLutTables`, `UnicodeProperties`, `X11ColorColorMap`, `StaticBitSet24`, `StaticBitSet256`, `DatastructCircBuf`, `RenderStateCellSet`, `StreamReadonlyHandler` |

Fields whose Go name would collide with a method (`Style.underline_color`,
`PageList.tracked_pins`, ...) and padding fields stay unexported; the method
is the access.
