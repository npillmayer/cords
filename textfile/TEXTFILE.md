# `textfile` Package Findings

## Scope Reviewed

- `/Users/npi/prg/go/cords/textfile/ReadMe.md`
- `/Users/npi/prg/go/cords/textfile/doc.go`
- `/Users/npi/prg/go/cords/textfile/load.go`
- `/Users/npi/prg/go/cords/textfile/load_test.go`

## High-Level Assessment

The `textfile` package is still built around the pre-btree/pre-chunk rope model and is currently not compatible with the active `cords` API. Conceptually, it implements an interesting lazy/asynchronous file-fragment loading mechanism, but the implementation is stale and contains multiple correctness and lifecycle issues.

## Current Build Status

`go test ./textfile` fails to compile:

- `textfile/load.go:74`: references removed `cords.Leaf`
- `textfile/load.go:78`: references removed `cords.StringLeaf`
- `textfile/load.go:211`: uses removed `Builder.Prepend`

This confirms `textfile` is not integrated with the current chunk-based `Cord` representation.

## Documented Intent vs Reality

Documentation claims:

- `ReadMe.md`: "work in progress, do not use"
- `doc.go`: efficient handling of large text via cords and concurrent fragment operations

Implementation reality:

- package does not compile against current APIs
- important runtime edge cases are not handled safely
- lifecycle and cleanup behavior is incomplete

## Architecture Summary (From `load.go`)

The package models a file as a chain/cycle of `fileLeaf` nodes:

- each leaf stores length and an `atomic.Value` extension
- extension is either:
  - loading metadata (`loadingInfo` with file position, next leaf, mutex), or
  - loaded content string
- a loader goroutine reads file fragments and broadcasts "loaded" messages
- one subscriber goroutine per leaf waits for its fragment message
- `fileLeaf.String()` blocks on a per-leaf mutex until the fragment is loaded

This is a lazy-loading design, but it is tightly coupled to removed legacy `cords` leaf internals.

## Major Findings

### 1. API incompatibility with current `cords`

- Legacy leaf APIs are used (`cords.Leaf`, `cords.StringLeaf`)
- Legacy builder API is used (`b.Prepend`)
- Current `cords` uses chunk-based tree items and builder methods like `AppendString`, `PrependString`, `AppendChunk`, `PrependChunk`

Impact: package is currently unusable.

### 2. EOF start-position bug (`initialPos == file size`)

- `Load` normalizes invalid/negative `initialPos` to `tf.info.Size()` (`textfile/load.go:128-130`)
- `createLeafsAndStartLoading` picks `start` with condition `k <= initialPos && initialPos < k+leaf.length` (`textfile/load.go:215-217`)
- For `initialPos == size`, no leaf satisfies `< k+leaf.length`, so `start` remains nil
- `loadAllFragmentsAsync` panics on nil start leaf (`textfile/load.go:237-239`)

Impact: documented "open at end" behavior (comment mentions `-1`) is inconsistent and can panic.

### 3. Empty-file divide-by-zero

- Fragment size may become `0` for empty files (`fragSize = tf.info.Size()` when size < 64; size can be 0) (`textfile/load.go:132-134`)
- Later: `rightmost := size / fragSize * fragSize` (`textfile/load.go:180`)

Impact: empty file input can panic due to division by zero.

### 4. Resource lifecycle leaks

- Opened file handle is never closed (`os.Open` in `openFile`, no matching `Close`)
- Publisher goroutine ranges over `fragChan` forever because channel is never closed (`textfile/load.go:284`; no `close(ch)` in loader)
- `startFileLoader` receives `wg` but does not use it for completion/lifecycle

Impact: leaked goroutines and file descriptors in long-lived processes.

### 5. Error propagation is incomplete

- Async read errors are stored in `tf.lastError` (`textfile/load.go:303`, `textfile/load.go:307`)
- This error is never surfaced to caller from `Load` or from blocking leaf reads

Impact: callers may receive partially broken data without a reliable error channel.

### 6. Text validity assumptions do not match current cord invariants

- Loader treats fragments as raw bytes and converts directly to string (`textfile/load.go:259`)
- No UTF-8 boundary or validity handling is done at fragment boundaries
- Current chunk-based ingestion in `cords` enforces UTF-8 validity and boundary safety

Impact: even after API migration, direct fragment slicing strategy would need redesign for UTF-8-safe chunk ingestion.

### 7. Concurrency model scales poorly

- One subscriber goroutine per fragment/leaf
- Broadcast delivery to all subscribers for each fragment message

Impact: O(number_of_fragments²) message fanout behavior and high goroutine count on large files.

### 8. Test coverage is minimal

- Only one test (`TestLoad`) with happy-path small file
- No tests for:
  - empty files
  - `initialPos = -1` / EOF start
  - UTF-8 edge fragments
  - error propagation
  - lifecycle cleanup

## What Still Makes Sense

- The product goal: loading large text into cords with predictable performance.
- The desire for progressive/lazy availability of data.
- Fragment-size heuristics as a starting point for tuning.

## What Is Superseded

- Custom `fileLeaf` integration into cord internals.
- Legacy leaf-based builder integration.
- Legacy metric/leaf-oriented assumptions about core rope representation.

## Recommended Direction (Migration Outline)

1. Rebase `textfile` on current public `cords` APIs only.
2. Replace legacy leaf integration with chunk-oriented ingestion (`chunk.NewBytes` rules respected).
3. Start with a synchronous correct loader first.
4. Reintroduce async/prefetch only with explicit lifecycle:
   - close file handles,
   - close channels,
   - bounded goroutines.
5. Define explicit error propagation contract for background loading.
6. Add coverage for EOF, empty file, UTF-8 boundaries, and cancellation/cleanup.

## Bottom Line

`textfile` is currently a legacy subsystem: useful as a design sketch, but not production-usable in the present codebase without a substantial rewrite to the chunk/sum-tree model.
