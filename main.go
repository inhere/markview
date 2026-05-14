package main

import (
	"embed"

	"github.com/inhere/markview/internal/bootstrap"
)

//go:embed web/template.html web/template-main.html web/dist
var content embed.FS

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	GitHash   = "unknown"
	BuildTime = "unknown"
)

func main() {
	bootstrap.Run(content, Version, GitHash, BuildTime)
}
