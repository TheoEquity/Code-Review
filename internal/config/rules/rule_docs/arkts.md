#### ArkTS High-Confidence Checks
- Correctness: `@State` object or array mutations that fail to refresh UI, unsafe null access, and invalid parent-child state synchronization.
- Lifecycle: timers, listeners, and subscriptions created in lifecycle hooks without matching cleanup.
- Render safety: network requests, timers, logging side effects, or expensive synchronous work in `build`.
- Security: user input used in SQL/command/string execution, sensitive data logged or uploaded, and insecure network requests.
