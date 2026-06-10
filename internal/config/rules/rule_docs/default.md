#### P4 System Fallback
Use this only as a lightweight fallback when no higher-priority rule gives more specific guidance.

Report only high-confidence issues in changed code:
- Correctness defects: broken control flow, missing boundary handling, unsafe nil/null access, and incorrect error handling.
- Security defects: injection risk, unsafe user-controlled output, sensitive data exposure, and missing authorization checks.
- Performance defects: obvious N+1 queries, unbounded large-data processing, and repeated expensive work in hot paths.
- Reliability defects: resource leaks, unsafe concurrent mutation, and incomplete cleanup on error paths.
- Test gaps: missing coverage for newly added critical logic or edge cases.

Avoid style-only, preference-only, spelling-only, and broad best-practice findings unless they directly create one of the risks above.
