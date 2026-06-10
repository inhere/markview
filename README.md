# MarkView

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/inhere/markview?style=flat-square)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/inhere/markview)](https://github.com/inhere/markview)
[![Unit-Tests](https://github.com/inhere/markview/actions/workflows/go.yml/badge.svg)](https://github.com/inhere/markview)

---

[English](./README.md) | [简体中文](./README.zh-CN.md)

MarkView is a zero-config Markdown preview server powered by Go and Bun.

It focuses on local documentation reading: fast startup, live updates, clear sidebar navigation, and solid support for Mermaid diagrams and code highlighting.

![MarkView](./example/images/screenshot_example.png)

## Features

- **🚀 Zero Config**: run it in any directory, open `README.md` by default, or show a directory listing when it is missing
- **⚡ Single-binary delivery**: the Go binary embeds `web/dist` and the HTML template, so no separate static deployment is required
- **🔄 Live Reload**: watches Markdown changes and updates the page through SSE
- **🔍 Full-text search**:
  - `Files` tree file name search
  - Support for searching document content, including headers, code blocks, etc.
- **🧭 Dual sidebar navigation**:
  - `Files` tree with expandable directories and current-file highlighting
  - `On This Page` table of contents with scroll spy
- **🔁 Inline navigation**:
  - file tree clicks do not trigger full page reloads
  - in-document internal Markdown links are handled inline
  - browser back/forward navigation is supported
- **📖 Split preview**: hover over internal Markdown links to reveal a preview button, click to open the target document in a side-by-side panel
- **🎨 Rich rendering**:
  - GFM (GitHub Flavored Markdown)
  - Image fullscreen viewer
  - `highlight.js` syntax highlighting
  - `mermaid.js` rendering with source toggle and fullscreen viewer
- **⚙️ Reading preferences**:
  - page width presets
  - font size increase, decrease, and reset
  - settings persisted in `localStorage`
- **📱 Responsive layout**: sidebar-first reading on desktop, single-column layout on mobile

## Install

```bash
go install github.com/inhere/markview@latest
```

Install by [Eget](https://github.com/inherelab/eget):

```bash
eget install inhere/markview
```

## Usage

### Run the executable

Download and run `markview`:

```bash
# Preview the current directory (optional port; unset uses automatic project port)
markview [-p PORT]

# Preview a specific directory (default is current directory)
markview "path/to/docs"

# Preview a specific directory and set the default entry file (default entry is `README.md`)
markview "path/to/docs" "intro.md"
```

When no port is specified, MarkView automatically chooses and remembers a project port, preferring `6100` when available.

> Example documents are available in [example/](example/).

### Configuration

You can adjust the port and default entry via `.env` / environment variables / options:

Using environment variables:

```bash
MKVIEW_PORT=8080 markview
MKVIEW_ENTRY=guide.md markview
MKVIEW_INCLUDE_DIR=.docs markview
```

```powershell
$env:MKVIEW_PORT = "8080"; markview
$env:MKVIEW_ENTRY = "guide.md"; markview
$env:MKVIEW_INCLUDE_DIR = ".docs"; markview
```

Using project config, create `markview.local.json`, `.markview.json`, or `markview.json` in the project root:

```json
{
  "server": {
    "include_dir": ".docs,.wiki"
  }
}
```

See [docs/markview-json-example.md](docs/markview-json-example.md) for a complete `markview.json` example.

`server.include_dir` and `MKVIEW_INCLUDE_DIR` allow selected skipped directories, including dot-prefixed documentation directories, to appear in the file tree. `.git` and `node_modules` are always skipped.

Using CLI options:

```bash
markview -p 6543
markview . "guide.md"
```

## Development

### Prerequisites

- **Go** 1.25+
- **Bun** 1.0+

### Project structure

```text
markview/
├── web/           # Frontend source, template, and build output
│   ├── src/
│   │   ├── app.ts              # Page lifecycle, navigation, orchestration
│   │   ├── sidebar.ts          # File tree and TOC logic
│   │   ├── mermaid.ts          # Mermaid enhancement and fullscreen behavior
│   │   ├── link-preview.ts     # Split preview for internal Markdown links
│   │   ├── preferences.ts      # Persisted reading preferences
│   │   └── live-status.ts      # SSE connection status handling
│   ├── template.html           # Main page template and CSS
│   ├── dist/                   # Bun build output embedded by Go
│   └── package.json
├── main.go                     # Go server entrypoint
├── handlers.go                 # Cache header and handler helpers
├── example/                    # Example Markdown documents
└── README.md
```

### Build from source

1. Install web dependencies and build:

```bash
cd web
bun install
bun run build
```

This generates `web/dist/` and also copies:
- `highlight.css`
- `logo.svg`
- `favicon.svg`

2. Build the backend:

```bash
cd ..
go build --ldflags "-w -s" -o markview.exe

# Or install to GOPATH/bin
go install -ldflags "-s -w" .
```

You can also use the provided `Makefile`:

```bash
make web
make build
make run
```

### Verification

Useful verification commands:

```bash
go test ./...
cd web && bun test ./src/*.test.ts
cd web && bun run build
```

> `go:embed` packages both `web/template.html` and `web/dist/` into the final binary.

## License

MIT
