// Port of src/terminal/search/sliding_window.zig (Ghostty v1.3.1).
//
// File naming: `search/sliding_window.zig` is `search_sliding_window.go`,
// prefix `SearchSlidingWindow` (lowerCamel `searchSlidingWindow` for
// private declarations). The file's main type, `SlidingWindow`, has the name
// of the file and so is not doubled: `SearchSlidingWindow` (the same rule
// that names the main type of every search file by its bare file prefix;
// see search.go). What is nested in it takes that prefix. Its private
// declarations are `searchSlidingWindowDirection` (values
// `searchSlidingWindowDirectionForward`, `...Reverse`),
// `searchSlidingWindowDataBuf`, `searchSlidingWindowMetaBuf` and
// `searchSlidingWindowMeta`. `init` is `SearchSlidingWindowInit`. `pub fn`
// methods are exported (`Next`, `Append`, `ClearAndRetainCapacity`,
// `Meta.Deinit`); private methods keep their Zig name (`highlight`,
// `testChangeNeedle`, `assertIntegrity`).
//
// The file is not re-exported by search.zig ("Non-public APIs"), but
// `SlidingWindow` is `pub` in its file and so exported here.
//
// Declarations appear in the same order as in sliding_window.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory.
//
//   - The field `alloc: Allocator` is dropped, and with it every allocator
//     argument (`DataBuf.init`, `Meta.deinit`, ...). `Deinit` stubs are
//     kept; the tests call `w.Deinit()`.
//   - `DataBuf` is `CircBuf(u8, 0)`: `DatastructCircBuf[uint8]`
//     (datastruct_circ_buf.go) made with default 0. `MetaBuf` is
//     `CircBuf(Meta, undefined)`: `DatastructCircBuf[searchSlidingWindowMeta]`
//     made with the zero Meta. Both are Go type aliases of the instantiated
//     generic type.
//   - `chunk_buf` is `std.MultiArrayList(FlattenedHighlight.Chunk)`: a
//     `[]HighlightFlattenedChunk`, as `HighlightFlattened.chunks` is
//     (highlight.go). `items(.node)[i]` is `chunk_buf[i].node`, `.len` is
//     `len(chunk_buf)`, `clearRetainingCapacity` is `chunk_buf[:0]`,
//     `ensureTotalCapacity` grows the slice's capacity. `highlight` ends with
//     `result.chunks = self.chunk_buf`, so a returned HighlightFlattened
//     shares the window's chunk slice exactly as in Zig: it is valid only
//     until the next `Next` or `Append`; callers that keep it `Clone` it.
//   - `needle` and `overlap_buf` are owned `[]uint8`; `alloc.dupe` is a Go
//     copy.
//   - `Meta.cell_map` is `std.ArrayList(point.Coordinate)`: a
//     `[]PointCoordinate`. `append` hands `&meta.cell_map` to the page
//     formatter as `FormatterPageFormatterPointMap{map_: &meta.cell_map}`
//     (formatter.go).
//   - `append` encodes into a `std.Io.Writer.Allocating`: the stand-in is
//     `oscWriterAllocating` (osc.go), as in formatter.go;
//     `encoded.written()` is its written bytes.
//   - Node pointers (`*PageList.List.Node`) are `*PageListNode`.
//   - `std.ascii.indexOfIgnoreCase` is not in the Go standard library with
//     Zig's semantics (ASCII-only case folding, first index); the
//     implementation writes it, ASCII only, not `strings.EqualFold`
//     (Unicode).
//   - `highlight`'s local anonymous struct (`tl`, with `br` and `prune`) is
//     a local Go type in the body; it is not declared here.
//
// # Widths
//
// `data_offset`, the `start_offset` and `len` parameters of `highlight` and
// the result of `Append` are counts (`usize`) and so `uint`. `Meta.serial`
// is `u64`, `uint64`.
//
// # Build options
//
// `std.debug.runtime_safety` (the asserts in `highlight` and the whole of
// `assertIntegrity`) is on in upstream's tests; the Go implementation keeps
// the checks unconditionally. The `std.valgrind.runningOnValgrind()` check
// has no Go counterpart.
//
// # Rulings (project lead)
//
//   - Empty needle: upstream's `next` reaches `highlight(0, 0)`, whose
//     `start + len - 1` wraps and whose meta loops end in `unreachable`
//     (ReleaseSafe: integer overflow; ReleaseFast: SIGSEGV). The port treats
//     an empty needle as a search with no matches: `Next` returns null at
//     once and leaves the window as it is. `init` and `append` accept the
//     empty needle as upstream does (docs/upstream-crashes.md in term51;
//     search_rulings_test.go).
//   - The end of `next`, `self.data.len() - self.needle.len + 1`, underflows
//     before the `+ 1` when pruning leaves exactly `needle.len - 1` bytes
//     (sliding_window.zig:279): a safe build traps, the release build wraps
//     to the intended 0, which is what the corpus records. The port computes
//     `len + 1 - needle.len` (docs/search-corpus.md in term51).
//
// # Declarations that are not Zig declarations
//
//   - the `String` method of searchSlidingWindowDirection and its name
//     table searchSlidingWindowDirectionNames;
//   - searchSlidingWindowCellCount: `@intCast` of a cell map y (u32) to a
//     chunk row (u16), asserting that the value fits as Zig's cast does;
//   - searchSlidingWindowReverseChunkField: `std.mem.reverse` on one column
//     of the chunk MultiArrayList (`slice.items(.node)`, `.start`, `.end`).
//     `highlight` reverses the node, start and end columns of a reverse
//     match but not the serial column, so in a multi-chunk reverse match the
//     serials stay in data order while the nodes are in screen order
//     (upstream quirk, findings section 19 item 4); the Go code reverses
//     the three columns one by one to keep that.
//
// # Control flow
//
// Labeled blocks: `slices:`, `prefix:`, `suffix:` and `formatter:` are
// plain blocks that assign a variable; `overlap:` and `prune:` are
// closures whose early `return` is `break :overlap` / `break :prune`; `tl:`
// is a closure returning the local struct. `while (...) |meta| { ... }
// else { ... }` is a loop followed by a test of a `found` flag (`break`
// skips the else branch). Every `else unreachable` and the switch on an
// exhaustive enum end in `panic("unreachable")`. `.?` on an iterator result
// is `assert(meta != nil)`.
//
// # Integer traps made explicit
//
// `data_len - self.data_offset` in `next`, `start + len - 1` in
// `highlight`, the u16 `ends[0] - 1` and `starts[last] + 1` of the
// reverse fixup, `@intCast` of the cell map y values, and `rows - 1` in
// `append` are asserted, since Zig traps there.
//
// # Omitted
//
// Imports and aliases with nothing to declare: `std`, `assert`,
// `Allocator`, `CircBuf`, `terminal`, `point`, `size`, `PageList`, `Pin`,
// `Selection`, `Screen`, `Terminal`, `PageFormatter` (use
// `FormatterPageFormatter`), `FlattenedHighlight` (use
// `HighlightFlattened`). No function is omitted. No C ABI declaration exists
// in this file.
//
// # Zig defaults a Go literal does not give
//
// `data_offset` defaults to 0, the Go zero value; no other field has a Zig
// default. A SearchSlidingWindow is made only by its `init`, which sets
// fields that are NOT the Go zero value: `data` and `meta` are
// `CircBuf.init(alloc, 0)`, which is a FULL zero-capacity buffer (the zero
// DatastructCircBuf is not full; see datastruct_circ_buf.go), and
// `overlap_buf` is `needle.len * 2` bytes. For `.reverse` the stored needle
// is reversed. So a translated test makes a window with
// SearchSlidingWindowInit, never with a literal.

package terminal

// SearchSlidingWindow is sliding_window.SlidingWindow.
//
// Searches page nodes via a sliding window. The sliding window maintains
// the invariant that data isn't pruned until (1) we've searched it and
// (2) we've accounted for overlaps across pages to fit the needle.
//
// The sliding window is first initialized empty. Pages are then appended
// in the order to search them. The sliding window supports both a forward
// and reverse order specified via `init`. The pages should be appended
// in the correct order matching the search direction.
//
// All appends grow the window. The window is only pruned when a search
// is done (positive or negative match) via `next()`.
//
// To avoid unnecessary memory growth, the recommended usage is to
// call `next()` until it returns null and then `append` the next page
// and repeat the process. This will always maintain the minimum
// required memory to search for the needle.
//
// The caller is responsible for providing the pages and ensuring they're
// in the proper order. The SlidingWindow itself doesn't own the pages, but
// it will contain pointers to them in order to return selections. If any
// pages become invalid, the caller should clear the sliding window and
// start over.
//
// Zig's first field, `alloc: Allocator` ("The allocator to use for all the
// data within this window."), is dropped.
type SearchSlidingWindow struct {
	// The data buffer is a circular buffer of u8 that contains the
	// encoded page text that we can use to search for the needle.
	data searchSlidingWindowDataBuf

	// The meta buffer is a circular buffer that contains the metadata
	// about the pages we're searching. This usually isn't that large
	// so callers must iterate through it to find the offset to map
	// data to meta.
	meta searchSlidingWindowMetaBuf

	// Buffer that can fit any amount of chunks necessary for next
	// to never fail allocation.
	//
	// Zig: std.MultiArrayList(FlattenedHighlight.Chunk). See the file
	// header.
	chunk_buf []HighlightFlattenedChunk

	// Offset into data for our current state. This handles the
	// situation where our search moved through meta[0] but didn't
	// do enough to prune it.
	//
	// Zig default: 0.
	data_offset uint

	// The needle we're searching for. Does own the memory.
	//
	// Zig: []const u8.
	needle []uint8

	// The search direction. If the direction is forward then pages should
	// be appended in forward linked list order from the PageList. If the
	// direction is reverse then pages should be appended in reverse order.
	//
	// This is important because in most cases, a reverse search is going
	// to be more desirable to search from the end of the active area
	// backwards so more recent data is found first. Supporting both is
	// trivial though and will let us do more complex optimizations in the
	// future (e.g. starting from the viewport and doing a forward/reverse
	// concurrently from that point).
	direction searchSlidingWindowDirection

	// A buffer to store the overlap search data. This is used to search
	// overlaps between pages where the match starts on one page and
	// ends on another. The length is always `needle.len * 2`.
	overlap_buf []uint8
}

// searchSlidingWindowDirection is SlidingWindow.Direction.
//
// Zig: enum { forward, reverse }, values numbered from zero in the order
// listed.
type searchSlidingWindowDirection uint8

const (
	searchSlidingWindowDirectionForward searchSlidingWindowDirection = iota
	searchSlidingWindowDirectionReverse
)

// searchSlidingWindowDirectionNames are the tag names of
// searchSlidingWindowDirection, in order.
var searchSlidingWindowDirectionNames = [...]string{
	"forward",
	"reverse",
}

// String returns the Zig tag name (@tagName).
func (self searchSlidingWindowDirection) String() string {
	return searchSlidingWindowDirectionNames[self]
}

// searchSlidingWindowDataBuf is SlidingWindow.DataBuf: CircBuf(u8, 0). Made
// with DatastructCircBufInit[uint8](0, size).
type searchSlidingWindowDataBuf = DatastructCircBuf[uint8]

// searchSlidingWindowMetaBuf is SlidingWindow.MetaBuf: CircBuf(Meta,
// undefined). Made with DatastructCircBufInit(searchSlidingWindowMeta{},
// size); Zig's `undefined` default is the Go zero value.
type searchSlidingWindowMetaBuf = DatastructCircBuf[searchSlidingWindowMeta]

// searchSlidingWindowMeta is SlidingWindow.Meta.
type searchSlidingWindowMeta struct {
	node *PageListNode
	// Zig: u64.
	serial uint64
	// Zig: std.ArrayList(point.Coordinate).
	cell_map []PointCoordinate
}

// Deinit is SlidingWindow.Meta.deinit.
//
// Zig: pub fn deinit(self: *Meta, alloc: Allocator) void. The allocator
// parameter is dropped. It only frees `cell_map`; the stub is kept for the
// callers.
func (self *searchSlidingWindowMeta) Deinit() {
	// Zig: self.cell_map.deinit(alloc), which frees the list and leaves it
	// undefined. The garbage collector frees; the list is dropped (nil).
	self.cell_map = nil
}

// SearchSlidingWindowInit is SlidingWindow.init.
//
// Zig: pub fn init(alloc: Allocator, direction: Direction, needle_unowned:
// []const u8) Allocator.Error!SlidingWindow. The allocator parameter is
// dropped. The needle is copied (and reversed for `.reverse`). The error is
// ErrOutOfMemory (dcs.go).
func SearchSlidingWindowInit(
	direction searchSlidingWindowDirection,
	needle_unowned []uint8,
) (SearchSlidingWindow, error) {
	data, err := DatastructCircBufInit[uint8](0, 0)
	if err != nil {
		return SearchSlidingWindow{}, err
	}
	// Zig: errdefer data.deinit(alloc); Go memory.

	meta, err := DatastructCircBufInit(searchSlidingWindowMeta{}, 0)
	if err != nil {
		return SearchSlidingWindow{}, err
	}
	// Zig: errdefer meta.deinit(alloc); Go memory.

	// Zig: alloc.dupe(u8, needle_unowned).
	needle := make([]uint8, len(needle_unowned))
	copy(needle, needle_unowned)
	// Zig: errdefer alloc.free(needle); Go memory.
	switch direction {
	case searchSlidingWindowDirectionForward:
	case searchSlidingWindowDirectionReverse:
		stdMemReverse(needle)
	default:
		panic("unreachable")
	}

	overlap_buf := make([]uint8, len(needle)*2)
	// Zig: errdefer alloc.free(overlap_buf); Go memory.

	return SearchSlidingWindow{
		data:        data,
		meta:        meta,
		chunk_buf:   nil, // .empty
		needle:      needle,
		direction:   direction,
		overlap_buf: overlap_buf,
	}, nil
}

// Deinit is SlidingWindow.deinit.
//
// Kept as a stub: every test calls it. It deinits every Meta and the
// buffers.
func (self *SearchSlidingWindow) Deinit() {
	// Zig: self.alloc.free(self.overlap_buf); self.alloc.free(self.needle).
	// The garbage collector frees; the slices are dropped.
	self.overlap_buf = nil
	self.needle = nil
	self.chunk_buf = nil // Zig: self.chunk_buf.deinit(self.alloc)
	self.data.Deinit()

	meta_it := self.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	for meta := meta_it.Next(); meta != nil; meta = meta_it.Next() {
		meta.Deinit()
	}
	self.meta.Deinit()
}

// ClearAndRetainCapacity is SlidingWindow.clearAndRetainCapacity.
//
// Clear all data but retain allocated capacity.
func (self *SearchSlidingWindow) ClearAndRetainCapacity() {
	meta_it := self.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	for meta := meta_it.Next(); meta != nil; meta = meta_it.Next() {
		meta.Deinit()
	}
	self.meta.Clear()
	self.data.Clear()
	self.data_offset = 0
}

// Next is SlidingWindow.next.
//
// Search the window for the next occurrence of the needle. As
// the window moves, the window will prune itself while maintaining
// the invariant that the window is always big enough to contain
// the needle.
//
// This returns a flattened highlight on a match. The
// flattened highlight requires allocation and is therefore more expensive
// than a normal selection, but it is more efficient to render since it
// has all the information without having to dereference pointers into
// the terminal state.
//
// The flattened highlight chunks reference internal memory for this
// sliding window and are only valid until the next call to `next()`
// or `append()`. If the caller wants to retain the flattened highlight
// then they should clone it.
//
// Zig: ?FlattenedHighlight.
func (self *SearchSlidingWindow) Next() Optional[HighlightFlattened] {
	// Go-only (project lead's ruling, see the file header): an empty
	// needle finds nothing. Upstream continues below to highlight(0, 0)
	// and crashes.
	if len(self.needle) == 0 {
		return None[HighlightFlattened]()
	}

	var slices [2][]uint8
	{
		// slices:
		// If we have less data then the needle then we can't possibly match
		data_len := self.data.Len()
		if data_len < uint(len(self.needle)) {
			return None[HighlightFlattened]()
		}

		// Zig: usize subtraction traps on underflow.
		assert(self.data_offset <= data_len)
		slices = self.data.GetPtrSlice(
			self.data_offset,
			data_len-self.data_offset,
		)
	}

	// Search the first slice for the needle.
	if idx := stdASCIIIndexOfIgnoreCase(slices[0], self.needle); idx.Valid {
		return Some(self.highlight(
			idx.Value,
			uint(len(self.needle)),
		))
	}

	// Search the overlap buffer for the needle.
	if len(slices[0]) > 0 && len(slices[1]) > 0 {
		// overlap:
		if hl, ok := func() (HighlightFlattened, bool) {
			// Get up to needle.len - 1 bytes from each side (as much as
			// we can) and store it in the overlap buffer.
			var prefix []uint8
			{
				// prefix:
				len_ := min(uint(len(slices[0])), uint(len(self.needle))-1)
				idx := uint(len(slices[0])) - len_
				prefix = slices[0][idx:]
			}
			var suffix []uint8
			{
				// suffix:
				len_ := min(uint(len(slices[1])), uint(len(self.needle))-1)
				suffix = slices[1][0:len_]
			}
			overlap_len := uint(len(prefix) + len(suffix))
			assert(overlap_len <= uint(len(self.overlap_buf)))
			copy(self.overlap_buf[0:len(prefix)], prefix)
			copy(self.overlap_buf[len(prefix):overlap_len], suffix)

			// Search the overlap
			idx := stdASCIIIndexOfIgnoreCase(
				self.overlap_buf[0:overlap_len],
				self.needle,
			)
			if !idx.Valid {
				return HighlightFlattened{}, false // break :overlap
			}

			// We found a match in the overlap buffer. We need to map the
			// index back to the data buffer in order to get our selection.
			return self.highlight(
				uint(len(slices[0])-len(prefix))+idx.Value,
				uint(len(self.needle)),
			), true
		}(); ok {
			return Some(hl)
		}
	}

	// Search the last slice for the needle.
	if idx := stdASCIIIndexOfIgnoreCase(slices[1], self.needle); idx.Valid {
		return Some(self.highlight(
			uint(len(slices[0]))+idx.Value,
			uint(len(self.needle)),
		))
	}

	// Special case 1-lengthed needles to delete the entire buffer.
	if len(self.needle) == 1 {
		self.ClearAndRetainCapacity()
		self.assertIntegrity()
		return None[HighlightFlattened]()
	}

	// No match. We keep `needle.len - 1` bytes available to
	// handle the future overlap case.
	func() {
		// prune:
		meta_it := self.meta.Iterator(DatastructCircBufIteratorDirectionReverse)
		var saved uint = 0
		found := false
		for meta := meta_it.Next(); meta != nil; meta = meta_it.Next() {
			needed := uint(len(self.needle)) - 1 - saved
			if uint(len(meta.cell_map)) >= needed {
				// We save up to this meta. We set our data offset
				// to exactly where it needs to be to continue
				// searching.
				self.data_offset = uint(len(meta.cell_map)) - needed
				found = true
				break
			}

			saved += uint(len(meta.cell_map))
		}
		if !found {
			// If we exited the while loop naturally then we
			// never got the amount we needed and so there is
			// nothing to prune.
			assert(saved < uint(len(self.needle))-1)
			return // break :prune
		}

		prune_count := self.meta.Len() - meta_it.idx
		if prune_count == 0 {
			// This can happen if we need to save up to the first
			// meta value to retain our window.
			return // break :prune
		}

		// We can now delete all the metas up to but NOT including
		// the meta we found through meta_it.
		meta_it = self.meta.Iterator(DatastructCircBufIteratorDirectionForward)
		var prune_data_len uint = 0
		for range prune_count {
			meta := meta_it.Next()
			assert(meta != nil) // .?
			prune_data_len += uint(len(meta.cell_map))
			meta.Deinit()
		}
		self.meta.DeleteOldest(prune_count)
		self.data.DeleteOldest(prune_data_len)
	}()

	// Our data offset now moves to needle.len - 1 from the end so
	// that we can handle the overlap case.
	//
	// Zig: self.data.len() - self.needle.len + 1, which underflows before
	// the `+ 1` when exactly needle.len - 1 bytes are left; the port adds
	// first (project lead's ruling, see the file header). Any smaller
	// length would still trap in Zig.
	assert(self.data.Len()+1 >= uint(len(self.needle)))
	self.data_offset = self.data.Len() + 1 - uint(len(self.needle))

	self.assertIntegrity()
	return None[HighlightFlattened]()
}

// highlight is SlidingWindow.highlight.
//
// Return a flattened highlight for the given start and length.
//
// The flattened highlight can be used to render the highlight
// in the most efficient way because it doesn't require a terminal
// lock to access terminal data to compare whether some viewport
// matches the highlight (because it doesn't need to traverse
// the page nodes).
//
// The start index is assumed to be relative to the offset. i.e.
// index zero is actually at `self.data[self.data_offset]`. The
// selection will account for the offset.
func (self *SearchSlidingWindow) highlight(start_offset uint, len_ uint) HighlightFlattened {
	start := start_offset + self.data_offset
	// Zig: `start + len - 1` traps on overflow (len 0 at start 0; the
	// empty needle never gets here, see Next).
	assert(start+len_ >= 1)
	end := start + len_ - 1
	// Zig: if (comptime std.debug.runtime_safety); see the file header.
	assert(start < self.data.Len())
	assert(start+len_ <= self.data.Len())

	// Clear our previous chunk buffer to store this result
	self.chunk_buf = self.chunk_buf[:0]
	result := HighlightFlattenedEmpty

	// Go through the meta nodes to find our start.
	type tlBr struct {
		it       DatastructCircBufIterator[searchSlidingWindowMeta]
		consumed uint
	}
	type tlPrune struct {
		meta uint
		data uint
	}
	type tlT struct {
		// If non-null, we need to continue searching for the bottom-right.
		br Optional[tlBr]

		// Data to prune, both are lengths.
		prune tlPrune
	}
	tl := func() tlT {
		// tl:
		meta_it := self.meta.Iterator(DatastructCircBufIteratorDirectionForward)
		var meta_consumed uint = 0
		for meta := meta_it.Next(); meta != nil; meta = meta_it.Next() {
			// Always increment our consumed count so that our index
			// is right for the end search if we do it.
			prior_meta_consumed := meta_consumed
			meta_consumed += uint(len(meta.cell_map))

			// meta_i is the index we expect to find the match in the
			// cell map within this meta if it contains it.
			meta_i := start - prior_meta_consumed

			// This meta doesn't contain the match. This means we
			// can also prune this set of data because we only look
			// forward.
			if meta_i >= uint(len(meta.cell_map)) {
				continue
			}

			// Now we look for the end. In MOST cases it is the same as
			// our starting chunk because highlights are usually small and
			// not on a boundary, so let's optimize for that.
			end_i := end - prior_meta_consumed
			if end_i < uint(len(meta.cell_map)) {
				// The entire highlight is within this meta.
				start_map := meta.cell_map[meta_i]
				end_map := meta.cell_map[end_i]
				result.top_x = start_map.x
				result.bot_x = end_map.x
				self.chunk_buf = append(self.chunk_buf, HighlightFlattenedChunk{
					node:   meta.node,
					serial: meta.serial,
					start:  searchSlidingWindowCellCount(start_map.y),
					end:    searchSlidingWindowCellCount(end_map.y + 1),
				})

				return tlT{
					br: None[tlBr](),
					prune: tlPrune{
						meta: meta_it.idx - 1,
						data: prior_meta_consumed,
					},
				}
			} else {
				// We found the meta that contains the start of the match
				// only. Consume this entire node from our start offset.
				map_ := meta.cell_map[meta_i]
				result.top_x = map_.x
				self.chunk_buf = append(self.chunk_buf, HighlightFlattenedChunk{
					node:   meta.node,
					serial: meta.serial,
					start:  searchSlidingWindowCellCount(map_.y),
					end:    meta.node.data.size.rows,
				})

				return tlT{
					br: Some(tlBr{
						it:       meta_it,
						consumed: meta_consumed,
					}),
					prune: tlPrune{
						meta: meta_it.idx - 1,
						data: prior_meta_consumed,
					},
				}
			}
		}
		// Precondition that the start index is within the data buffer.
		panic("unreachable")
	}()

	// Search for our end.
	if tl.br.Valid {
		br := tl.br.Value
		meta_it := br.it
		meta_consumed := br.consumed
		found := false
		for meta := meta_it.Next(); meta != nil; meta = meta_it.Next() {
			// meta_i is the index we expect to find the match in the
			// cell map within this meta if it contains it.
			meta_i := end - meta_consumed
			if meta_i >= uint(len(meta.cell_map)) {
				// This meta doesn't contain the match. We still add it
				// to our results because we want the full flattened list.
				self.chunk_buf = append(self.chunk_buf, HighlightFlattenedChunk{
					node:   meta.node,
					serial: meta.serial,
					start:  0,
					end:    meta.node.data.size.rows,
				})

				meta_consumed += uint(len(meta.cell_map))
				continue
			}

			// We found it
			map_ := meta.cell_map[meta_i]
			result.bot_x = map_.x
			self.chunk_buf = append(self.chunk_buf, HighlightFlattenedChunk{
				node:   meta.node,
				serial: meta.serial,
				start:  0,
				end:    searchSlidingWindowCellCount(map_.y + 1),
			})
			found = true
			break
		}
		if !found {
			// Precondition that the end index is within the data buffer.
			panic("unreachable")
		}
	}

	// Our offset into the current meta block is the start index
	// minus the amount of data fully consumed. We then add one
	// to move one past the match so we don't repeat it.
	self.data_offset = start - tl.prune.data + 1

	// If we went beyond our initial meta node we can prune.
	if tl.prune.meta > 0 {
		// Deinit all our memory in the meta blocks prior to our
		// match.
		meta_it := self.meta.Iterator(DatastructCircBufIteratorDirectionForward)
		var meta_consumed uint = 0
		for range tl.prune.meta {
			meta := meta_it.Next()
			assert(meta != nil) // .?
			meta_consumed += uint(len(meta.cell_map))
			meta.Deinit()
		}
		// Zig: if (comptime std.debug.runtime_safety); see the file header.
		{
			assert(meta_it.idx == tl.prune.meta)
			next := meta_it.Next()
			assert(next != nil) // .?
			assert(next.node == self.chunk_buf[0].node)
		}
		self.meta.DeleteOldest(tl.prune.meta)

		// Delete all the data up to our current index.
		assert(tl.prune.data > 0)
		self.data.DeleteOldest(tl.prune.data)
	}

	switch self.direction {
	case searchSlidingWindowDirectionForward:
	case searchSlidingWindowDirectionReverse:
		// Zig: slice.items(.node), items(.start), items(.end): the fields
		// of the chunks; `serial` is not among them.
		chunks := self.chunk_buf

		if len(self.chunk_buf) > 1 {
			// Reverse all our chunks. This should be pretty obvious why.
			//
			// Zig: std.mem.reverse on the node, start and end columns of
			// the MultiArrayList, each on its own; the serial column is
			// not reversed.
			searchSlidingWindowReverseChunkField(chunks, func(c *HighlightFlattenedChunk) *PageListNode { return c.node },
				func(c *HighlightFlattenedChunk, v *PageListNode) { c.node = v })
			searchSlidingWindowReverseChunkField(chunks, func(c *HighlightFlattenedChunk) SizeCellCountInt { return c.start },
				func(c *HighlightFlattenedChunk, v SizeCellCountInt) { c.start = v })
			searchSlidingWindowReverseChunkField(chunks, func(c *HighlightFlattenedChunk) SizeCellCountInt { return c.end },
				func(c *HighlightFlattenedChunk, v SizeCellCountInt) { c.end = v })

			// Now normally with forward traversal with multiple pages,
			// the suffix of the first page and the prefix of the last
			// page are used.
			//
			// For a reverse traversal, this is inverted (since the
			// pages are in reverse order we get the suffix of the last
			// page and the prefix of the first page). So we need to
			// invert this.
			//
			// We DON'T need to do this for any middle pages because
			// they always use the full page.
			//
			// This is a fixup that makes our start/end match the
			// same logic as the loops above if they were in forward
			// order.
			n := len(chunks)
			assert(n >= 2)
			assert(chunks[0].end > 0) // Zig: u16 subtraction traps
			chunks[0].start = chunks[0].end - 1
			chunks[0].end = chunks[0].node.data.size.rows
			assert(chunks[n-1].start < 0xFFFF) // Zig: u16 addition traps
			chunks[n-1].end = chunks[n-1].start + 1
			chunks[n-1].start = 0
		} else {
			// For a single chunk, the y values are in reverse order
			// (start is the screen-end, end is the screen-start).
			// Swap them to get proper top-to-bottom order.
			start_y := chunks[0].start
			assert(chunks[0].end > 0) // Zig: u16 subtraction traps
			chunks[0].start = chunks[0].end - 1
			assert(start_y < 0xFFFF) // Zig: u16 addition traps
			chunks[0].end = start_y + 1
		}

		// X values also need to be reversed since the top/bottom
		// are swapped for the nodes.
		top_x := result.top_x
		result.top_x = result.bot_x
		result.bot_x = top_x
	default:
		panic("unreachable")
	}

	// Copy over our MultiArrayList so it points to the proper memory.
	result.chunks = self.chunk_buf
	return result
}

// searchSlidingWindowCellCount is `@intCast` of a cell map y (u32) to a
// chunk row (size.CellCountInt, u16), which traps when the value does not
// fit. Not a Zig declaration.
func searchSlidingWindowCellCount(y uint32) SizeCellCountInt {
	assert(y <= 0xFFFF) // Zig: @intCast traps
	return SizeCellCountInt(y)
}

// searchSlidingWindowReverseChunkField is std.mem.reverse on one column of
// the chunk MultiArrayList (`slice.items(.field)`), the other columns left
// in place. Not a Zig declaration.
func searchSlidingWindowReverseChunkField[F any](
	chunks []HighlightFlattenedChunk,
	get func(*HighlightFlattenedChunk) F,
	set func(*HighlightFlattenedChunk, F),
) {
	column := make([]F, len(chunks))
	for i := range chunks {
		column[i] = get(&chunks[i])
	}
	stdMemReverse(column)
	for i := range chunks {
		set(&chunks[i], column[i])
	}
}

// Append is SlidingWindow.append.
//
// Add a new node to the sliding window. This will always grow
// the sliding window; data isn't pruned until it is consumed
// via a search (via next()).
//
// Returns the number of bytes of content added to the sliding window.
// The total bytes will be larger since this omits metadata, but it is
// an accurate measure of the text content size added.
//
// Zig: Allocator.Error!usize. The error is ErrOutOfMemory (dcs.go); Zig
// also maps any formatter or writer error to it.
func (self *SearchSlidingWindow) Append(node *PageListNode) (uint, error) {
	// Initialize our metadata for the node.
	meta := searchSlidingWindowMeta{
		node:     node,
		serial:   node.serial,
		cell_map: nil, // .empty
	}
	// Zig: errdefer meta.deinit(self.alloc); Go memory.

	// This is suboptimal but we need to encode the page once to
	// temporary memory, and then copy it into our circular buffer.
	// In the future, we should benchmark and see if we can encode
	// directly into the circular buffer.
	encoded := oscWriterAllocatingInit()
	defer encoded.deinit()

	// Encode the page into the buffer.
	var formatter FormatterPageFormatter
	{
		// formatter:
		opts := FormatterOptionsPlain // .emit = .plain
		opts.unwrap = true
		formatter = FormatterPageFormatterInit(&meta.node.data, opts)
		formatter.point_map = Some(FormatterPageFormatterPointMap{
			map_: &meta.cell_map,
		})
	}
	if err := formatter.format(&encoded.writer); err != nil {
		// writer uses anyerror but the only realistic error on
		// an ArrayList is out of memory.
		return 0, ErrOutOfMemory
	}
	assert(len(meta.cell_map) == len(encoded.written()))

	// If the node we're adding isn't soft-wrapped, we add the
	// trailing newline.
	assert(node.data.size.rows > 0) // Zig: u16 subtraction traps
	row := node.data.GetRow(uint(node.data.size.rows - 1))
	if !row.wrap {
		if err := encoded.writer.writeByte('\n'); err != nil {
			return 0, ErrOutOfMemory
		}
		// Zig: meta.cell_map.getLastOrNull() orelse .{ .x = 0, .y = 0 }
		last := PointCoordinate{x: 0, y: 0}
		if len(meta.cell_map) > 0 {
			last = meta.cell_map[len(meta.cell_map)-1]
		}
		meta.cell_map = append(meta.cell_map, last)
	}

	// If our written data is empty, then there is nothing to
	// add to our data set.
	written := encoded.written()
	if len(written) == 0 {
		self.assertIntegrity()
		return 0, nil
	}

	// Get our written data. If we're doing a reverse search then we
	// need to reverse all our encodings.
	switch self.direction {
	case searchSlidingWindowDirectionForward:
	case searchSlidingWindowDirectionReverse:
		stdMemReverse(written)
		stdMemReverse(meta.cell_map)
	default:
		panic("unreachable")
	}

	// Ensure our buffers are big enough to store what we need.
	if err := self.data.EnsureUnusedCapacity(uint(len(written))); err != nil {
		return 0, err
	}
	if err := self.meta.EnsureUnusedCapacity(1); err != nil {
		return 0, err
	}
	// Zig: try self.chunk_buf.ensureTotalCapacity(self.alloc,
	// self.meta.capacity()).
	if uint(cap(self.chunk_buf)) < self.meta.Capacity() {
		chunk_buf := make([]HighlightFlattenedChunk, len(self.chunk_buf), self.meta.Capacity())
		copy(chunk_buf, self.chunk_buf)
		self.chunk_buf = chunk_buf
	}

	// Append our new node to the circular buffer.
	self.data.AppendSliceAssumeCapacity(written)
	self.meta.AppendAssumeCapacity(meta)

	self.assertIntegrity()
	return uint(len(written)), nil
}

// testChangeNeedle is SlidingWindow.testChangeNeedle.
//
// Only for tests!
//
// Zig: new is []const u8; asserts it has the needle's length. It does NOT
// reverse the new needle for a reverse window.
func (self *SearchSlidingWindow) testChangeNeedle(new []uint8) {
	assert(len(new) == len(self.needle))
	// Zig: self.alloc.free(self.needle); self.alloc.dupe(u8, new) catch
	// unreachable.
	needle := make([]uint8, len(new))
	copy(needle, new)
	self.needle = needle
}

// assertIntegrity is SlidingWindow.assertIntegrity.
//
// Integrity check: the data length equals the sum of the metas' cell map
// lengths, and the data offset is within bounds. See "Build options" in
// the file header.
func (self *SearchSlidingWindow) assertIntegrity() {
	// Zig: if (comptime !std.debug.runtime_safety) return; on in the port,
	// see the file header.

	// We don't run integrity checks on Valgrind because its soooooo slow,
	// Valgrind is our integrity checker, and we run these during unit
	// tests (non-Valgrind) anyways so we're verifying anyways.
	//
	// Zig: if (std.valgrind.runningOnValgrind() > 0) return; no Go
	// counterpart.

	// Integrity check: verify our data matches our metadata exactly.
	meta_it := self.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	var data_len uint = 0
	for m := meta_it.Next(); m != nil; m = meta_it.Next() {
		data_len += uint(len(m.cell_map))
	}
	assert(data_len == self.data.Len())

	// Integrity check: verify our data offset is within bounds.
	assert(self.data.Len() == 0 || self.data_offset < self.data.Len())
}
