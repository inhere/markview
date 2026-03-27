# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**重要** 优先使用中文回复和响应

## Build and run

- `make help` — show available make targets.
- `make build` — build the frontend bundle first, then compile `markview.exe`.
- `make frontend` — install frontend deps if needed and bundle `frontend/app.ts` into `frontend/dist` with Bun.
- `make backend` — compile the Go binary only.
- `make run` — build everything and run the local server.
- `make install` — build frontend assets and install the Go binary to `GOPATH/bin`.
- `make clean` — remove `markview.exe`, `frontend/dist`, and `dist/`.
- `make build-all` — cross-compile release binaries into `dist/`.

## Test and verification

- There are currently no `*_test.go` files in the repository.
- `go test ./...` — use this as the default Go verification command.
- `go test ./... -run TestName` — run a single Go test by name once tests exist.
- `cd frontend && bun run build` — quickest way to verify frontend bundling succeeds.

## Runtime behavior

- Running the binary with no args serves the current directory and opens `README.md` by default.
- `./markview.exe [directory] [default-entry]`
- `SERVER_PORT=8080 ./markview.exe` — override the default port (`3000`).

## Architecture overview

### Single-binary Go server

The app is a single `main.go` program. It:

- parses CLI args for the served directory and default entry file,
- reads `SERVER_PORT`,
- starts a recursive filesystem watcher for the served directory,
- serves HTTP routes for the markdown page, static files from the served directory, and the SSE reload endpoint.

Important entry points:

- `main.go:48` — bootstraps config, watcher, embedded asset serving, and HTTP routes.
- `main.go:104` — resolves request paths relative to the served directory and rejects path traversal.
- `main.go:150` — renders Markdown files into the HTML template.
- `main.go:195` — handles SSE clients for live reload.
- `main.go:235` — watches the served directory recursively and broadcasts reload events when `.md` files change.

### Embedded frontend assets

The Go binary embeds `frontend/template.html` and the built `frontend/dist` directory via `go:embed` in `main.go:24`.

That means frontend changes are not picked up by the Go binary until the Bun build is rerun. In practice, `make build` or `make frontend` must happen before rebuilding or releasing the Go binary.

### Markdown rendering pipeline

Markdown rendering is server-side:

- files are read directly from the served directory,
- `goldmark` is configured with GFM support and auto heading IDs,
- raw HTML is allowed via `html.WithUnsafe()`,
- rendered HTML is injected into the Go HTML template as `template.HTML`.

This is why both the renderer settings in `main.go` and the DOM logic in the frontend matter when changing document behavior.

### Frontend responsibilities

The frontend is split between:

- `frontend/template.html` — page shell, layout, sidebar, toolbar, modal markup, and most CSS.
- `frontend/app.ts` — runtime behavior.

`frontend/app.ts` is responsible for:

- dynamic loading of Mermaid and Highlight.js,
- registering syntax highlight languages,
- transforming Mermaid code fences into interactive diagram containers,
- generating the table of contents from rendered headings,
- scroll-spy behavior for the TOC,
- toolbar controls for width and font size,
- fullscreen Mermaid modal behavior and zoom controls,
- listening to `/sse` and reloading the page when the backend broadcasts `reload`.

### Live reload model

Live reload is markdown-focused:

- the Go watcher walks the served directory recursively,
- `.git` and `node_modules` directories are skipped,
- only `.md` write events trigger `broadcast("reload")`,
- new directories are added to the watcher when created,
- the browser keeps an `EventSource` connection to `/sse` and refreshes on reload messages.

If a change affects template or frontend bundle behavior, rebuilding the app is required; editing markdown alone should live-reload without rebuilding.

## Repo structure

- `main.go` — backend server, markdown rendering, SSE, and file watching.
- `frontend/app.ts` — client-side behavior.
- `frontend/template.html` — embedded HTML/CSS template.
- `frontend/dist/` — Bun build output embedded into the Go binary.
- `assets/` — static project assets.
- `dist/` — cross-compiled release binaries from `make build-all`.

## Notes for future changes

- Prefer updating `Makefile` targets when changing the build flow; it is the canonical developer entrypoint.
- Be careful when editing request-path handling in `main.go`; the `filepath.Rel` check is the main path traversal guard.
- Frontend and backend changes are coupled through the embedded asset pipeline, so verify both Bun build output and Go rebuilds when touching UI behavior.
