#### JavaScript/TypeScript High-Confidence Checks
- Correctness: unsafe null/undefined access, incorrect async error handling, stale closure bugs, and state updates that can lose user data.
- React: effects with missing cleanup, render-time side effects, invalid hook usage, and unstable list keys that break state association.
- Security: unsafe HTML injection, use of `eval` or `Function`, user-controlled script execution, and exposed secrets.
- Performance: repeated network calls during render/effects, expensive work on every render path, and unbounded async fan-out.
