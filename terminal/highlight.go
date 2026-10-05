// Port of src/terminal/highlight.zig (Ghostty v1.3.1).
//
// Highlights are any contiguous sequences of cells that should
// be called out in some way, most commonly for text selection but
// also search results or any other purpose.
//
// Within the terminal package, a highlight is a generic concept
// that represents a range of cells.
//
// NOTE: The plan is for highlights to ultimately replace Selection
// completely. Selection is deeply tied to various parts of the Ghostty
// internals so this may take some time.
//
// File naming: highlight.zig is a namespace file, so its declarations take
// the prefix `Highlight`: `highlight.Untracked` is `HighlightUntracked`,
// `highlight.Tracked` is `HighlightTracked`, `highlight.Flattened` is
// `HighlightFlattened`, `highlight.Flattened.Chunk` is
// `HighlightFlattenedChunk`, `highlight.Flattened.empty` is
// `HighlightFlattenedEmpty`. `init` functions are `<Type>Init`:
// `HighlightTrackedInit`, `HighlightTrackedInitAssume`.
//
// Declarations appear in the same order as in highlight.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory. A pin is a
// `PageListPin` and a tracked pin a `*PageListPin` (PageList.go). Allocator
// parameters are dropped; `deinit` stubs are kept.
//
// `Flattened.chunks` is a `std.MultiArrayList(Chunk)`, a struct-of-arrays
// list. Go: `[]HighlightFlattenedChunk`. Zig's `chunks.slice().items(.node)[i]`
// is `chunks[i].node`, `chunks.len` is `len(chunks)`, `.empty` is the nil
// slice, `chunks.clone(alloc)` is a copy of the slice.
//
// # Omitted
//
// Aliases and imports with nothing to declare: `PageChunk`
// (`PageList.PageIterator.Chunk`: use `PageListPageIteratorChunk`), `Pin`
// (use `PageListPin`), `Screen`, `size`, `Allocator`.
//
// # Upstream defect, recorded
//
// `Flattened.init` (line 141) cannot compile in this version: it builds a
// `std.MultiArrayList(PageChunk)`, appends values with a `serial` field
// that `PageChunk` does not have, and returns a struct literal with a field
// `end_x` that `Flattened` does not have (the field is `bot_x`). Zig only
// analyses a function that is referenced and nothing references this one,
// so the library builds. There is no upstream behaviour to port and no
// upstream test, so it is not ported (docs/mapping.md, "Page-layer
// bookkeeping": `PageList.diagram` and `HighlightFlattenedInit` are not
// ported); a comment marks its place, as for `PageList.diagram`. The tests
// build flattened highlights with their own helper.

package terminal

// HighlightUntracked is highlight.Untracked.
//
// An untracked highlight is a highlight that stores its highlighted
// area as a top-left and bottom-right screen pin. Since it is untracked,
// the pins are only valid for the current terminal state and may not
// be safe to use after any terminal modifications.
//
// For rectangle highlights/selections, the downstream consumer of this
// code is expected to interpret the pins in whatever shape they want.
// For example, a rectangular selection would interpret the pins as
// setting the x bounds for each row between start.y and end.y.
//
// To simplify all operations, start MUST be before or equal to end.
type HighlightUntracked struct {
	start PageListPin
	end   PageListPin
}

// Track is highlight.Untracked.track.
//
// Zig: pub fn track(self: *const Untracked, screen: *Screen)
// Allocator.Error!Tracked. The error is ErrOutOfMemory (dcs.go).
func (self *HighlightUntracked) Track(screen *Screen) (HighlightTracked, error) {
	return HighlightTrackedInit(
		screen,
		self.start,
		self.end,
	)
}

// Eql is highlight.Untracked.eql.
func (self HighlightUntracked) Eql(other HighlightUntracked) bool {
	return self.start.Eql(other.start) && self.end.Eql(other.end)
}

// HighlightTracked is highlight.Tracked.
//
// A tracked highlight is a highlight that stores its highlighted
// area as tracked pins within a screen.
//
// A tracked highlight ensures that the pins remain valid even as
// the terminal state changes. Because of this, tracked highlights
// have more operations available to them.
//
// There is more overhead to creating and maintaining tracked highlights.
// If you're manipulating highlights that are untracked and you're sure
// that the terminal state won't change, you can use the `initAssume`
// function.
type HighlightTracked struct {
	start *PageListPin
	end   *PageListPin
}

// HighlightTrackedInit is highlight.Tracked.init.
//
// Zig: pub fn init(screen: *Screen, start: Pin, end: Pin)
// Allocator.Error!Tracked. The error is ErrOutOfMemory (dcs.go).
func HighlightTrackedInit(
	screen *Screen,
	start PageListPin,
	end PageListPin,
) (_ HighlightTracked, err error) {
	start_tracked, err := screen.pages.TrackPin(start)
	if err != nil {
		return HighlightTracked{}, err
	}
	defer func() {
		// errdefer
		if err != nil {
			screen.pages.UntrackPin(start_tracked)
		}
	}()
	end_tracked, err := screen.pages.TrackPin(end)
	if err != nil {
		return HighlightTracked{}, err
	}
	// Zig: errdefer screen.pages.untrackPin(end_tracked); nothing after it
	// can fail, so it never runs.
	return HighlightTracked{
		start: start_tracked,
		end:   end_tracked,
	}, nil
}

// HighlightTrackedInitAssume is highlight.Tracked.initAssume.
//
// Initializes a tracked highlight by assuming that the provided
// pins are already tracked. This allows callers to perform tracked
// operations without the overhead of tracking the pins, if the
// caller can guarantee that the pins are already tracked or that
// the terminal state will not change.
//
// Do not call deinit on highlights created with this function.
func HighlightTrackedInitAssume(start *PageListPin, end *PageListPin) HighlightTracked {
	return HighlightTracked{
		start: start,
		end:   end,
	}
}

// Deinit is highlight.Tracked.deinit.
//
// Kept: it untracks both pins in the screen's page list.
func (self HighlightTracked) Deinit(screen *Screen) {
	screen.pages.UntrackPin(self.start)
	screen.pages.UntrackPin(self.end)
}

// HighlightFlattened is highlight.Flattened.
//
// A flattened highlight is a highlight that stores its highlighted
// area as a list of page chunks. This representation allows for
// traversing the entire highlighted area without needing to read any
// terminal state or dereference any page nodes (which may have been
// pruned).
type HighlightFlattened struct {
	// The page chunks that make up this highlight. This handles the
	// y bounds since chunks[0].start is the first highlighted row
	// and chunks[len - 1].end is the last highlighted row (exclsive).
	//
	// Zig: std.MultiArrayList(Chunk). See the file header.
	chunks []HighlightFlattenedChunk

	// The x bounds of the highlight. `bot_x` may be less than `top_x`
	// for typical left-to-right highlights: can start the selection right
	// of the end on a higher row.
	top_x SizeCellCountInt
	bot_x SizeCellCountInt
}

// HighlightFlattenedChunk is highlight.Flattened.Chunk.
//
// A flattened chunk is almost identical to a PageList.Chunk but
// we also flatten the serial number. This lets the flattened
// highlight more robust for comparisons and validity checks with
// the PageList.
type HighlightFlattenedChunk struct {
	node *PageListNode
	// Zig: u64.
	serial uint64
	start  SizeCellCountInt
	end    SizeCellCountInt
}

// HighlightFlattenedEmpty is highlight.Flattened.empty.
var HighlightFlattenedEmpty = HighlightFlattened{
	chunks: nil,
	top_x:  0,
	bot_x:  0,
}

// highlight.Flattened.init (line 141) is omitted: upstream's body does not
// compile and is never referenced; see the file header.

// Deinit is highlight.Flattened.deinit.
//
// Zig: pub fn deinit(self: *Flattened, alloc: Allocator) void. The
// allocator parameter is dropped. It only frees; the stub is kept for the
// callers.
func (self *HighlightFlattened) Deinit() {
	// Zig: self.chunks.deinit(alloc), which frees the list and leaves
	// `chunks` undefined. The garbage collector frees in Go; the list is
	// dropped (nil).
	self.chunks = nil
}

// Clone is highlight.Flattened.clone.
//
// Zig: pub fn clone(self: *const Flattened, alloc: Allocator)
// Allocator.Error!Flattened. The allocator parameter is dropped. The error
// is ErrOutOfMemory (dcs.go).
func (self *HighlightFlattened) Clone() (HighlightFlattened, error) {
	// Zig: try self.chunks.clone(alloc): a list with its own storage holding
	// the same elements (std's clone of an empty list is an empty list).
	chunks := make([]HighlightFlattenedChunk, len(self.chunks))
	copy(chunks, self.chunks)
	return HighlightFlattened{
		chunks: chunks,
		top_x:  self.top_x,
		bot_x:  self.bot_x,
	}, nil
}

// StartPin is highlight.Flattened.startPin.
func (self HighlightFlattened) StartPin() PageListPin {
	slice := self.chunks
	return PageListPin{
		node: slice[0].node,
		x:    self.top_x,
		y:    slice[0].start,
	}
}

// EndPin is highlight.Flattened.endPin.
func (self HighlightFlattened) EndPin() PageListPin {
	slice := self.chunks
	end := slice[len(slice)-1].end
	assert(end > 0) // Zig: `end - 1` traps on u16 underflow.
	return PageListPin{
		node: slice[len(slice)-1].node,
		x:    self.bot_x,
		y:    end - 1,
	}
}

// Untracked is highlight.Flattened.untracked.
//
// Convert to an Untracked highlight.
func (self HighlightFlattened) Untracked() HighlightUntracked {
	// Note: we don't use startPin/endPin here because it is slightly
	// faster to reuse the slices.
	//
	// Zig: `slice.items(.node)`, `.items(.start)`, `.items(.end)` are the
	// fields of the elements of the one Go slice (file header).
	slice := self.chunks
	start := PageListPin{
		node: slice[0].node,
		x:    self.top_x,
		y:    slice[0].start,
	}
	end := slice[len(slice)-1].end
	assert(end > 0) // Zig: `ends[ends.len - 1] - 1` traps on u16 underflow.
	return HighlightUntracked{
		start: start,
		end: PageListPin{
			node: slice[len(slice)-1].node,
			x:    self.bot_x,
			y:    end - 1,
		},
	}
}
