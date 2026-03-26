# MarkView

MarkView is a high-performance, zero-config Markdown preview server with Live Reload, powered by Go and Bun.

![MarkView](https://img.shields.io/badge/MarkView-v1.0.0-blue)

## Features

- **🚀 Zero Config**: Just run the executable in any directory.
- **⚡ Fast**: Powered by Go backend and Bun-bundled frontend.
- **🔄 Live Reload**: Instant updates via SSE when files change.
- **🎨 Rich Rendering**:
  - GFM (GitHub Flavored Markdown) support via `goldmark`.
  - Syntax highlighting via `highlight.js`.
  - Mermaid diagrams via `mermaid.js` with **fullscreen modal support**.
  - Auto-generated Table of Contents (TOC) with scroll spy.
- **📱 Responsive**: Mobile-friendly "Swiss Document" layout.
- **📦 Single Binary**: ~13MB standalone executable with no external dependencies.

## Usage

### Running the Executable

Download `markview.exe` and run it:

```powershell
# Serve current directory
.\markview.exe

# Serve specific directory
.\markview.exe "path/to/docs"

# Serve specific directory and set default entry file
.\markview.exe "path/to/docs" "intro.md"
```

The server will start at `http://localhost:3000` (default).

### Configuration

You can configure the port via environment variable:

```powershell
$env:SERVER_PORT = "8080"; .\markview.exe
```

## Development

### Prerequisites

- **Go** 1.22+
- **Bun** 1.0+ (for frontend bundling)

### Project Structure

```
markview/
├── frontend/           # TypeScript frontend
│   ├── app.ts          # Main client logic (Mermaid, TOC, SSE)
│   ├── template.html   # Go HTML template
│   └── package.json    # Frontend dependencies
├── main.go             # Go backend server
├── go.mod              # Go dependencies
└── README.md           # Documentation
```

### Build from Source

1. **Install Frontend Dependencies & Bundle**:

```bash
cd frontend
bun install
bun run build
# Generates app.js
```

2. **Build Backend**:

```bash
cd ..
go build --ldflags "-w -s" -o markview.exe
```

> The `//go:embed` directive will automatically include `frontend/app.js` and `frontend/template.html` into the binary.

## License

MIT
