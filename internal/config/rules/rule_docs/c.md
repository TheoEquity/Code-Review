#### C High-Confidence Checks
- Memory safety: leaks, double-free, use-after-free, missing cleanup on error paths, and ownership confusion.
- Buffer safety: unchecked array access, unsafe string copy/format operations, and incorrect loop bounds.
- Error handling: ignored allocation or system-call failures that can corrupt state or crash later.
- Concurrency: shared mutable data accessed without synchronization.
