#### Kotlin High-Confidence Checks
- Correctness: unsafe `!!`, nullable API results used without guards, non-exhaustive state handling, and boundary mistakes in collection access.
- Coroutines: work launched outside the owning scope, missing cancellation, and exceptions lost in `async` or dispatcher switches.
- Performance: repeated expensive collection transformations on large data, object creation in hot loops, and blocking IO on main/UI threads.
- Reliability: files, streams, cursors, or network resources not closed through all paths.
