# Changelog

## 2026-02-10

- ui: replaced text-color and `>` cursor selection with background-color highlighting across all lists (zones, services, ports, rich rules, network, IPSets, templates, backups).
- ui: unified tab styles (Services/Ports/Rich Rules/Network/IPSets/Info) with cyan/gray background palette matching selection colors; removed gaps between tabs for a seamless panel.
- ui: styled main panel header with cyan background and added "Zone:" prefix.
- ui: refactored statusbar view and changed background to cyan.
- ui: replaced bullet/icon markers with straight text marks.
- ui: added vertical separator between Runtime/Permanent columns in split view.
- fix: corrected split view column width calculation to account for mainStyle border and padding, preventing text overflow on long content.
- fix: fixed split view separator rendering by styling each line independently to prevent ANSI code interference with `JoinHorizontal`.
- fix: added `tea.ClearScreen` on help mode exit to prevent rendering artifacts from taller help content.

## 2026-02-06

- security: added `internal/validation.IsValidZoneName` and applied zone-name validation in backup/UI command paths.
- security: hardened XML parsing in `internal/backup/zone_xml.go` with strict decoder and size limits.
- fix: made rich-rule update transactional with rollback (`updateRichRuleTransaction`).
- fix: added transactional restore/import flows with rollback attempts on reload failure.
- fix: made log line storage thread-safe via shared locked store and snapshot reads.
- fix: made signal unsubscribe cancel idempotent.
- fix: changed firewalld version detection to fail-fast on invalid/unreadable version values.
- feat: added D-Bus call timeouts to client and signal match operations.
- fix: made backup description input truly optional.
- feat: graceful shutdown path with signal-aware UI context.
- fix: improved config comment stripping to respect quoted strings.
- refactor: split UI update logic into `update_loop.go`, `update_input.go`, `update_navigation.go`, `update_mutations.go`, and `update_helpers.go` with minimal `update.go` entrypoint.
- test: added unit tests for validation, backup XML handling, backup restore flow, rich-rule transaction helper, signal cancel idempotency, version parsing, config parsing, and UI input/log helpers.
- test: expanded coverage with additional config parser tests, firewalld permission helper tests, backup list/cleanup/XML writer tests, and UI helper tests.
- test: added UI navigation/mutation/model helper tests and expanded view/backup XML roundtrip assertions.
- refactor: removed duplicated local zone-name validation in UI IPSet input path and reused `validation.IsValidZoneName`.
- refactor: extracted Update mode handling into dedicated helpers (`handleHelpMode`, `handleTemplateMode`, `handleBackupMode`, `handleInputMode`, `handleDetailsMode`) in `internal/ui/update_modes.go`.
- test: added mode-handler tests in `internal/ui/update_modes_test.go`.
- test: added `internal/firewalld/zone_settings_test.go` for zone settings parsing and variant conversion/error paths.
- test: added `internal/firewalld/zones_test.go` for active-zones normalization and dedupe helper coverage.
- test: expanded `internal/ui/commands_test.go` with zone validation and import/restore guard-rail coverage.
- refactor: narrowed `updateRichRuleCmd` dependency to `richRuleUpdater` interface for easier command-level testing.
- test: added `updateRichRuleCmd` permanent/runtime branch tests with a mock updater.
- test: added `internal/ui/model_test.go` for `NewModel` defaults and lazy log store initialization checks.
- docs: added package-level GoDoc files (`doc.go`) for backup, firewalld, ui, config, validation, and logger packages.
- perf: added TTL cache for IPSet entries in `firewalld.Client` with mutation/reload invalidation hooks.
- test: added `internal/firewalld/ipset_cache_test.go` for cache isolation, expiry, invalidation, and cached-read behavior.
- perf: added lightweight D-Bus call rate limiting (`dbusMinCallGap`) in generic call paths and signal subscription match handling.
- test: added D-Bus rate-limit delay tests in `internal/firewalld/client_test.go`.
- perf: added baseline benchmarks for config parsing/comment stripping and zone-name validation.
- perf: optimized config parser line scanning and quoted-string fast-path; `BenchmarkParse` improved to ~693 ns/op with 0 allocs/op.
- perf: optimized `validation.IsValidZoneName` invalid-path by using sentinel errors; invalid benchmark now ~7 ns/op with 0 allocs/op.
- build: added ldflags-based version metadata wiring in `Makefile` and release build script.
- ci: added GitHub Actions CI workflow with tests, linters, security checks, and linux multi-arch build matrix.
- release: added automated GitHub release workflow for tags `v*` with linux artifacts (`amd64`, `arm64`, `arm`, `386`) and SHA256/SHA512 checksums.
