# Project Port Registry Design

## Goal

Persist per-project preview ports in `~/.config/markview/markview-projects.json` so repeated launches keep stable local URLs when the user wants automatic port selection.

## Scope

This design only covers project port memory. It does not implement `--projects list/show/remove/prune`, `-P|--project`, or single-page sharing.

## Activation Rules

MarkView reads and writes the project registry only in these cases:

- CLI port is explicitly random: `-p -1` or `--port -1`
- Port is not set by CLI and not set by `MKVIEW_PORT`

MarkView does not read or write the registry when:

- CLI uses a fixed port, such as `-p 8080`
- `MKVIEW_PORT` is set, including `MKVIEW_PORT=-1`

This keeps explicit configuration authoritative and prevents manual port choices from overwriting remembered project ports.

## Registry Location

All platforms use:

```text
~/.config/markview/markview-projects.json
```

`~` resolves through `os.UserHomeDir()`. On Windows this is typically `C:\Users\<user>`.

## Registry Format

The file is a JSON object keyed by cleaned absolute project path:

```json
{
  "/abs/project/path": {
    "port": 6100,
    "name": "project-name",
    "added": "2026-05-14T15:00:00+08:00"
  }
}
```

Record rules:

- `name` defaults to the project directory base name.
- `added` is written only when the project is first added.
- Existing records update only `port`.
- Paths use `filepath.Abs` and `filepath.Clean` before lookup.

## Port Selection

### No CLI/ENV Port

When the user starts `markview` without `--port/-p` and without `MKVIEW_PORT`:

1. Load the registry.
2. If the project has a saved port, try it first.
3. If the saved port is unavailable, try the next ports in ascending order.
4. If the project has no saved port, try the default port `6100`.
5. If `6100` is unavailable, try the next ports in ascending order.
6. Save the actual bound port after the listener is created.

### CLI `-p -1`

When the user starts `markview -p -1`:

1. Load the registry.
2. If the project has a saved port, try it first.
3. If the saved port is unavailable, try the next ports in ascending order.
4. If the project has no saved port, bind `:0` and let the OS choose.
5. Save the actual bound port after the listener is created.

### Fixed Port

When the user provides a fixed CLI port or `MKVIEW_PORT`, the current behavior remains: MarkView uses that port and does not touch the registry.

## Error Handling

The registry is a convenience feature and must not block preview startup.

- Missing registry file means an empty registry.
- Missing parent directory is created on save.
- Invalid JSON logs a warning and behaves as an empty registry.
- Save errors log a warning and do not stop the server.
- If 100 ascending ports are unavailable, fallback to `:0` and save the OS-selected port.

## Code Boundaries

Add a focused package:

```text
internal/projects
```

Responsibilities:

- Resolve registry path.
- Load and save registry JSON.
- Normalize project keys.
- Look up and upsert project records.

Keep network binding in `main.go` or a small main-level helper. The registry package should not know about listeners or sockets.

## CLI Source Tracking

`config.Cfg.PortInt` alone cannot distinguish "unset" from explicit values. The main command should detect whether `port` was visited by `cflag` after parsing. Since `cflag.CFlags` embeds `flag.FlagSet`, this can use `Visit`.

Configuration should track whether the port was explicitly provided by CLI or environment so `run()` can decide whether registry mode is active.

## Tests

Registry tests should cover:

- `RegistryPath` uses `~/.config/markview/markview-projects.json`.
- Missing file loads an empty registry.
- Valid JSON loads records.
- Invalid JSON returns an error to the caller.
- `Upsert` creates a record with default name and added timestamp.
- `Upsert` updates an existing port while preserving name and added.

Main-level tests should cover:

- CLI `-p -1` activates registry mode.
- No CLI/ENV port activates registry mode.
- Fixed CLI port does not activate registry mode.
- `MKVIEW_PORT` does not activate registry mode.
- Saved port is preferred when available.
- Occupied saved/default port chooses the next available port.
