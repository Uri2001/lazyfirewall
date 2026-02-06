# Progress Tracking

## Week 1: Critical Fixes
- [x] Task 1.1: Path traversal fix
- [x] Task 1.2: XXE hardening
- [x] Task 2.1: updateRichRule atomicity
- [x] Task 2.2: Import/Restore transactionality
- [x] Task 3.1: log lines race mitigation
- [x] Task 3.2: cancel idempotency
- [x] Task 4.1: version detection fail-fast
- [x] Task 4.2: D-Bus timeout wiring

## Week 2: Testing & Stability
- [x] Task 5: comprehensive coverage to target thresholds
- [x] Task 6: split `internal/ui/update.go`
- [x] Task 7: config parser quoted comment handling
- [x] Task 8: backup description optional
- [x] Task 9: graceful shutdown

## Week 3+: Quality
- [x] Task 10: package-level GoDoc (`doc.go`) for core internal packages
- [x] Task 11: IPSet entry caching
- [x] Task 12: D-Bus rate limiting
- [ ] Task 13: profiling and optimization

## Notes
- Added baseline unit tests for critical paths touched in Week 1.
- Expanded parser/helper tests (`config`, `firewalld`, `backup`, `ui`) as part of Task 5 progression.
- Added extra UI navigation/mutation/model helper tests and backup XML roundtrip tests.
- Extracted UI mode-handling from `Update(...)` into dedicated helpers and added focused tests for those handlers.
- Added `internal/firewalld/zone_settings_test.go` for variant/port parsing and invalid-zone detection paths.
- Added `internal/firewalld/zones_test.go` for active-zones normalization and string dedupe helpers.
- Expanded `internal/ui/commands_test.go` with guard-rail tests for invalid zone names and import/restore prechecks.
- Refactored `updateRichRuleCmd` to a narrow `richRuleUpdater` interface and added explicit permanent/runtime branch tests.
- Added `internal/ui/model_test.go` to validate `NewModel` defaults and log store initialization behavior.
- Added `doc.go` package documentation for `internal/backup`, `internal/firewalld`, `internal/ui`, `internal/config`, `internal/validation`, and `internal/logger`.
- Added TTL-based cache for `GetIPSetEntries` with invalidation on IPSet entry mutations, remove/create IPSet, reload, and runtime commit.
- Added `internal/firewalld/ipset_cache_test.go` for cache copy/expiry/invalidation behavior.
- Added global D-Bus call throttling (`dbusMinCallGap`) in `call`, `callObject`, and signal match/unmatch flow.
- Added deterministic unit tests for D-Bus rate-limit delay calculations in `internal/firewalld/client_test.go`.
- Added benchmark baselines for config parsing/comment stripping and zone validation in `internal/config/config_bench_test.go` and `internal/validation/zone_bench_test.go`.
- Baseline benchmark snapshot: `BenchmarkStripComment` ~54 ns/op, 0 allocs/op.
- Baseline benchmark snapshot: `BenchmarkParse` ~918 ns/op, 1 alloc/op.
- Baseline benchmark snapshot: `BenchmarkIsValidZoneNameValid` ~55 ns/op, 0 allocs/op.
- Baseline benchmark snapshot: `BenchmarkIsValidZoneNameInvalid` ~119 ns/op, 2 allocs/op.
- Linux-tagged test packages are compile-verified in this environment (`go test -c`), while runtime execution requires a Linux host/CI runner.
