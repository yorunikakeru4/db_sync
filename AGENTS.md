# Backend Development

Rules here apply to the Go backend. `vue/`, `mcp/`, and `mcp/openapi/` have their own `AGENTS.md`.

## Core Rules

- Go 1.26+.
- Use `encoding/json` for JSON marshaling unless a package already requires something else.
- Use `github.com/goccy/go-yaml`, not `yaml.v2/v3/v4`.
- Minimize dependencies on `github.com/uptrace/*`. `bun` and `bunrouter` are allowed; prefer non-`uptrace` packages for everything else.
- Use `errors.AsType`, not `errors.As`.
- Enum string values and OpenTelemetry span names use `snake_case`.
- When renaming a YAML option, keep the old field as a deprecated pointer field, migrate it in init with `slog.Warn`, and only apply it when the new field is unset.
- Document all functions, structs and their fields, and package-level vars/consts.
- Prefer early returns. Keep code ordered top-down: high-level entry points before helpers/callees.
- Avoid unnecessary copying/allocations; document mutation when a function mutates inputs.

## Repo Conventions

- Most `pkg/*` directories have a `README.md`; read it before changing an unfamiliar package.
- Dependency injection uses `go.uber.org/fx`; modules usually expose `module.go`.
- `cmd/hosted` is the self-hosted monolith and uses in-memory processors.
- `cmd/cloud` is split into `api` and `worker` and uses Kafka.
- CLI command groups under `cmd/` stay in one package with sibling files; the parent file defines `Command()`.

## Testing And Telemetry

- Prefer local `datadriven/`; use table-driven tests only when datadriven is a poor fit.
- Standalone tests need a comment describing the scenario they cover.
- Mock interfaces with `testify/mock`.
- Rewrite datadriven expectations with `go test -rewrite ./...`.
- Update snapshots with `UPDATE_SNAPS=true go test ./...`.
- If an error is already logged with `e.Logger.ErrorContext`, do not also call `span.RecordError(err)` on that path; still set span status.

## Commands

- Build hosted: `go build cmd/hosted/*.go`
- Build cloud: `go build cmd/cloud/*.go`
- Main backend verification: `task test`
- If dashboard templates change: `go run cmd/cloud/*.go dashboard validate_templates`

## Changelog

- User-visible features and notable bug fixes should add a concise entry to `vue/src/CHANGELOG.md` under today's date.
