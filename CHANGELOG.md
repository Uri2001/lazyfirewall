# Changelog

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
