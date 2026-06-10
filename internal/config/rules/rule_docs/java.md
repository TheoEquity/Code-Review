#### Java High-Confidence Checks
- Correctness: wrong branch conditions, missing boundary checks, unreachable logic that changes behavior, unsafe null access, and unintended switch fall-through.
- Performance: database or remote calls inside loops, N+1 query patterns, and unbounded processing of large result sets.
- Concurrency: check-then-act races, non-atomic compound updates, unsafe lazy initialization, and concurrent writes to non-thread-safe collections.
- Reliability: resources not closed on success or failure paths, swallowed exceptions that hide failed operations, and cleanup skipped after partial failure.
