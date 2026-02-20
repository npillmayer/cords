# `textfile` Progressive Loading Proposal

## Goal

Enable clients to ingest very large UTF-8 text files as `cords.Cord` values and
start processing early, before the full file has finished loading.

The intended UX is:

1. Start load.
2. Receive a usable cord snapshot quickly (front of file available).
3. Continue receiving updated snapshots while background loading appends more data.
4. Finish when loading completes or fails.

## Constraints

- Current `cords.Cord` is immutable and chunk-backed.
- The old legacy "blocking leaf placeholder" design no longer fits current internals.
- Therefore progressive behavior should be expressed as *snapshot updates*, not
  mutating internal leaf payloads after publication.

## Proposed API

```go
type Loader struct {
    // Monotonic growing snapshots. Each value is a valid immutable cord.
    Updates <-chan cords.Cord

    // Closed when background loading has terminated (success or error).
    Done <-chan struct{}

    // Returns terminal error (nil on success).
    Err func() error

    // Blocks until done and returns terminal error.
    Wait func() error

    // Cancels loading and releases resources.
    Close func() error
}

func LoadAsync(name string, fragSize int64) (*Loader, error)
```

`Load(name, initialPos, fragSize, wg)` remains as synchronous convenience API
for callers who want the full cord before processing.

## Loading Model

1. Open file and start one reader goroutine.
2. Read byte buffers.
3. Maintain UTF-8 boundary safety across read boundaries.
4. Append validated bytes/chunks to a builder.
5. Periodically publish a new cord snapshot on `Updates`.
6. On EOF, publish final snapshot (if needed), close `Done`.
7. On error, set terminal error, close `Done`.
8. On cancellation, stop read loop, close file, close channels.

## Snapshot Semantics

- Snapshots are immutable and safe to retain.
- New snapshots may share structure with earlier ones.
- Order on `Updates` is monotonic by loaded prefix length.
- Clients can process incrementally by tracking last processed byte offset.

## Error and Lifecycle Semantics

- `Err()` is only meaningful after `Done` is closed.
- `Wait()` returns the same terminal result as `Err()`.
- `Close()` is idempotent and may be called by clients at any time.
- File handles and goroutines must always terminate on success, error, or cancel.

## Suggested Client Pattern

```go
ldr, err := textfile.LoadAsync("huge.txt", 0)
if err != nil { /* handle */ }
defer ldr.Close()

var last uint64
for c := range ldr.Updates {
    // Process newly available range [last, c.Len()).
    // ...
    last = c.Len()
}
if err := ldr.Wait(); err != nil {
    // handle load failure/cancel
}
```

## Non-Goals (for first iteration)

- Random-access on-demand page-in from disk.
- Full mmap-based implementation.
- Concurrent multi-reader shard loading.

The first target is a robust streaming loader with progressive immutable
snapshots and clear lifecycle semantics.
