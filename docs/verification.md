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

## Final gate: the whole library (project lead rulings, 2026-10-06)

Code: Go b40fcb9 (library code identical to fdb1e46; b40fcb9 only updates
replayer self-tests for the new corpus suite) with term51 2326139.

### The rule run

| Run | Platform | Rule |
|---|---|---|
| Every test except the split runs (all unit tests, every recorded area, search, input in all three platform modes, tables, self-tests) | native | `TERM51_FULL=1` |
| Split | native | started as: full rule for csi, esc, modes, resize, text, parser, fuzz; reduced rule (`TERM51_SPLIT_LONG=64,64,64`) for dcs, osc, scroll. Stopped part-way and finished with the sampled default split (see "Why the full-state splits stopped") |
| Every test except the split runs | 32-bit ARM (qemu) | all unit tests; the recorded areas sampled (default run) |
| Split | 32-bit ARM (qemu) | the sampled default run |
| Everything, every recorded case in full, plus the sampled split | the BlackBerry phone (real 32-bit ARM, QNX) | run by the project lead |

### Why dcs moved to the reduced rule, and emulated ARM to sampling

The final gate replays the golden kind (the whole terminal state) for every
split variant, not only the actions as for the stream layer. Under the full
rule dcs alone took about 4 hours with actions only; its 300 KB feeds now
also dump a whole terminal per variant. dcs's long feeds are payload runs,
`NextSlice` is per-byte (see "Why not the full rule everywhere" above), and
csi's full split with full state was clean, so dcs joins osc and scroll on
the reduced rule.

The phone is the full 32-bit gate (project lead, 2026-10-06): it runs every
recorded case in full on the real chip. The emulated ARM run therefore keeps
all unit tests and samples the recorded cases and the split.

### Why the full-state splits stopped (project lead ruling, 2026-10-06)

Unlike the stream layer's splits, every variant here also replays and
compares the whole terminal state. That proved far heavier than estimated:
after about three hours csi's last suite and `text/style_split` (46,708-column
screens, feeds of about 1 MB, with the page self-checks on) were still
running, and dcs, esc, modes, resize, parser and fuzz had not started. The
full-state splits were stopped at that point, all clean, and the rest ran
with the sampled default split, as on ARM and the phone. The reason this is
enough: the stream layer's splits already proved under the full and reduced
rules, in every area, that every cut gives the same action sequence
(11,114,684 variants, 0 mismatches), and the terminal state is a function of
that sequence.

Full-state split variants completed before the stop (0 mismatches each):

| Area | Rule | Suites | Variants (actions and golden) |
|---|---|---|---|
| csi | full | 44 of 45 | 9,998,362 |
| text | full | 13 of 14 | 702,658 |
| osc, scroll | reduced | 11 of 13 | 1,660,329 |
| total | | | 12,361,349 |

### Results

Native, every test except the split runs (`TERM51_FULL=1`): 204,963 passed on
ababf3f, 7 skipped (exactly the tests upstream itself skips); the only
failures were the two `text/style_split` crash cases and three self-tests
that count them, all fixed by the merge that followed. Rerun on b40fcb9 of
everything that merge can reach (the Screen, Terminal, page and PageList
tests and the self-tests): 9,455 passed, 0 failed. The text area in full on b40fcb9: 6,806 of 6,806
cases.

Native sampled split on b40fcb9 (csi, text, scroll, dcs, esc, modes, resize,
parser, fuzz): 2,426 cases, 228,813 variants (actions and golden), 0
mismatches.

32-bit ARM (qemu), every test except the split runs, recorded cases
sampled: 10,567 passed, 7 skipped (upstream's own), 0 failed. Sampled split:
113 of 129 suites, 2,264 cases, 216,695 variants, 0 mismatches (csi, dcs,
esc, fuzz, modes, osc, parser, resize, and 4 of scroll's 6 suites). It was
stopped there (project lead ruling, 2026-10-06): scroll's last suite pushes
12,000 lines through a tiny scrollback on every cut and ran for hours under
emulation; that suite and the text area are covered by the native sampled
split above and by the phone's sampled split on the real 32-bit chip.

Phone (project lead's runs): every recorded case in full and every unit test
pass on the real device (the only failures were self-tests missing source
files in the copy); the rerun on the final commit with the sampled split:
PHONE_FINAL (the project lead fills this in when the run ends).

