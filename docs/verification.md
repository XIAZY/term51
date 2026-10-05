# Verification runs

What was actually run to sign off each layer, per area and platform, and
why. The replayer and its environment variables are described in
`terminal/corpus_replay_test.go` and `terminal/corpus_replay_split_test.go`.

## Stream layer (project lead ruling, 2026-10-05)

### What is compared

The `actions`, `actions_noalloc` and `parser` kinds of every recorded case.
The `golden` kind needs `Terminal` and is not part of this sign-off; it is
counted but expected to fail until the terminal layer is ported.

### The rule run

| Run | Areas | Platform | Rule |
|---|---|---|---|
| Area tests (`TestCorpusReplay<Area>`) | all ten | native and 32-bit ARM | `TERM51_FULL=1`: every case of every suite |
| Split (`TestCorpusReplaySplit<Area>`) | csi, dcs, esc, modes, resize, text, parser, fuzz | native | `TERM51_FULL=1`, the full rule: every offset of a feed of at most 1,000 bytes; of a longer feed the first 256 offsets, the last 256 and 1,024 seeded ones |
| Split | osc, scroll | native | the reduced rule: `TERM51_FULL=1 TERM51_SPLIT_LONG=64,64,64`: every offset of a feed of at most 1,000 bytes; of a longer feed the first 64, the last 64 and 64 seeded |
| Split | all ten | 32-bit ARM | the default run (no `TERM51_FULL`): 20 sampled cases per suite plus the first case of each marker kind; offsets 1..64 of every feed and 3 seeded ones above 64 |

### Why not the full rule everywhere

A split variant replays its whole case, so a run costs variants times case
length. With the full rule that is about 45 billion byte-feeds for the
action kinds: about 45 hours natively at the measured rate of about one
billion byte-feeds an hour (scroll 28.8 billion, osc 8.4 billion, dcs 4.3
billion), and about ten times that on ARM under qemu.

The risk the split runs guard against is narrow for this port. In the
reference build (`build_options.simd = false`) `Stream.nextSlice` is a loop
over `next` (mapping rule 8; `stream.go`), so any split of a feed hands the
same byte sequence to the same per-byte function, and split invariance
follows from the code's structure. `nextSliceCapped`, the one chunked path,
is held to per-byte `next` by
`TestStream_go_only_nextSliceCapped_equals_per_byte_next` (30,000 seeded
inputs). The full rule on csi (5,107,039 action variants, 0 mismatches) and
dcs is further evidence.

### When the full rule comes back

If a change gives `NextSlice` (or any feed path) a route that is not
per-byte `next`, for example a SIMD or chunked decode path, the full split
rule (`TERM51_FULL=1` without `TERM51_SPLIT_LONG`) is run again, natively, for
every area whose cases reach that path, before the change is accepted.

Prefix-sharing (replaying a case once up to the cut and cloning the
terminal) is deliberately not built into the replayer: it would make the
oracle depend on `Terminal` cloning for little gain.

### Results

Filled in when the runs finish.
