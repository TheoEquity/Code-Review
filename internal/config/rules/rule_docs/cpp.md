#### C++ High-Confidence Checks
- Memory safety: leaks, double-free, use-after-free, ownership confusion, and circular `shared_ptr` references.
- Bounds safety: unsafe array/vector indexing, invalid iterator use, and buffer writes without length checks.
- Exception safety: resources leaked or state left inconsistent when constructors, allocations, or calls throw.
- Concurrency: data races, unsynchronized shared mutation, and lifetime hazards across threads.
