# devlaunch

`devlaunch` is a local Windows project launcher built in Go.

It provides:

- a Cobra CLI
- a local SSR web UI served by the same Go binary
- a runtime that reads `.devlaunch/manifest.json` and `.devlaunch/state.local.json`
- embedded PowerShell scripts for Windows-specific execution
- a repo-hosted skill under `skills/devlaunch`

## Scope

V1 targets:

- Windows
- PowerShell
- Windows Terminal
- localhost-only UI

The runtime model only uses:

- `apps`
- `services`

`docker compose up -d` must always be modeled as a `service`.

## Architecture

```text
devlaunch
├── cmd/devlaunch          Cobra entrypoint
├── internal/cli           CLI commands
├── internal/runtime       manifest/state/registry orchestration
├── internal/web           local SSR web UI + HTTP handlers
├── internal/windows       Go wrappers over PowerShell scripts
├── scripts/ps1            native Windows execution scripts
├── skill                  embedded skill source
└── devlaunch-specs        product and implementation specs
```

Execution model:

```text
Cobra CLI / SSR UI
        |
        v
   runtime in Go
        |
        +--> manifest/state/registry logic in Go
        |
        +--> Windows actions via embedded scripts/ps1
```

## CLI

Top-level commands:

```text
devlaunch init
devlaunch project list
devlaunch project start [project-id]
devlaunch project stop [project-id]
devlaunch project status [project-id]
devlaunch project open [project-id]
devlaunch ui start [-d]
devlaunch ui stop
devlaunch skill install
```

Resolution rule:

- if `[project-id]` is provided, `devlaunch` targets a project from the global registry
- if `[project-id]` is omitted, `devlaunch` targets the current working directory

## Local Files

Per project:

```text
<repo>\.devlaunch\manifest.json
<repo>\.devlaunch\state.local.json
```

Global registry:

```text
%USERPROFILE%\.devlaunch\registry\projects.json
```

## UI

The UI is only a convenience layer.

It is served by the same Go binary and exposes actions that also exist in the CLI:

- start
- stop
- open folder
- status/list through the runtime API

Start it in foreground:

```powershell
go run .\cmd\devlaunch ui start
```

Start it detached:

```powershell
go run .\cmd\devlaunch ui start -d
```

Stop it:

```powershell
go run .\cmd\devlaunch ui stop
```

## Build

Format and build:

```powershell
gofmt -w .\cmd .\internal
$env:GOMODCACHE='C:\PROJETS\devlaunch\.gomodcache'
$env:GOCACHE='C:\PROJETS\devlaunch\.gocache'
go mod tidy
go build ./...
```

Run the CLI:

```powershell
go run .\cmd\devlaunch --help
```

Build a standalone binary:

```powershell
go build -o .\bin\devlaunch.exe .\cmd\devlaunch
```

The resulting binary embeds:

- `scripts/ps1`

At runtime, `devlaunch` extracts them under:

```text
%USERPROFILE%\.devlaunch\runtime\
```

## Skill Install

The skill is published from this repository under:

```text
skills/devlaunch
```

`devlaunch skill install` proxies:

```text
npx skills add mahmoud-nn/devlaunch -g --skill devlaunch
```

## Notes

- Go is used for orchestration, CLI, UI, manifest/state/registry handling, and simple reliable operations.
- PowerShell scripts are used for Windows-native actions when that is more reliable than encoding the behavior directly in Go.
- Specs live in [devlaunch-specs/README.md](/C:/PROJETS/devlaunch/devlaunch-specs/README.md).
