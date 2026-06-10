#### Mapper/DAO XML High-Confidence Checks
- SQL correctness: wrong logical operators, missing join predicates, invalid dynamic SQL conditions, and obvious syntax defects.
- SQL injection: `${}` or string-built `LIKE` clauses using user-controlled values.
- Performance: missing filters on large queries, unbounded result sets, and repeated expensive subqueries.
- Mapper consistency: XML `id` and interface method mismatches that break runtime binding.
