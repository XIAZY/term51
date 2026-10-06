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
| Split | all ten | 32-bit ARM | the default run (no `TERM51_FULL`): 20 sampled cases per suite plus the first case of each marker kind; offsets 1..64 of every feed and 3 seeded ones above 64 (the default offsets over every case would take about 24 hours under qemu) |

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

Native (2026-10-05, main at 191ba5f..849b780; the stream layer did not
change in between): no mismatch in the `actions`, `actions_noalloc` or
`parser` kinds. The `golden` kind fails everywhere, as expected before
`Terminal` is ported.

Area tests, every case: 190,104 cases in 128 suites; `actions` 190,104
cases, `actions_noalloc` 4,840, `parser` 9,010 traces; 0 mismatches.

| Area | Split rule | Cases | Action variants | Mismatches |
|---|---|---:|---:|---:|
| csi | full | 52,898 | 5,107,039 | 0 |
| dcs | full | 648 | 233,778 | 0 |
| esc | full | 2,032 | 85,929 | 0 |
| fuzz | full | 30,900 | 2,335,667 | 0 |
| modes | full | 4,773 | 599,719 | 0 |
| parser | full | 78,397 | 1,030,798 | 0 |
| resize | full | 4,116 | 143,400 | 0 |
| text | full | 6,803 | 388,407 | 0 |
| osc | reduced | 4,192 | 393,394 | 0 |
| scroll | reduced | 5,345 | 796,553 | 0 |
| total | | | 11,114,684 | 0 |

`osc/rgbi_float` (the recorded cases of the Zig std parseFloat defect):
57 cases, 0 action mismatches; split 13,296 action variants, 0
mismatches.

32-bit ARM: area tests in full and the default split (the sampled default
run: 20 cases per suite plus the first case of each marker kind; offsets
1..64 and 3 seeded per feed), 2026-10-05 21:28Z to 2026-10-06: no mismatch
in the `actions`, `actions_noalloc` or `parser` kinds. Area tests: 190,104 of
190,104 cases. Sampled split: 2,565 cases, 125,804 action variants.
The sampled default split on ARM is the project lead's ruling (2026-10-05):
the split risk is architecture-independent by construction (`NextSlice` is a
per-byte loop on every platform), and the full ARM area run covers the
32-bit-specific behaviour.

Sign-off: the stream layer was signed off by the project lead on these
native results (2026-10-05); the ARM area run confirms it.
