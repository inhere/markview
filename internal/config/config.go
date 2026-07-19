package config

import (
	"fmt"
	"strconv"

	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/x/clog"
)

var (
	EnableDebug bool
)

var DefaultSkipDirs = []string{
	"node_modules",
	"dist",
	"tmp",
	"temp",
}

type Config struct {
	Version string `json:"-"` // app version
	portStr string

	// public fields

	TargetDir     string
	EntryFile     string
	PortInt       int
	PortSource    PortSource
	EnableWatch   bool
	WatchDirs     []string
	WatchSkipDirs []string
	IncludeDirs   []string
	Private       bool
	NoBrowser     bool
	PreviewExts   []string
	IframeHosts   []string
	UILayout      string
	BasePath      string
}

type AppConfig struct {
	PreviewExts []string `json:"previewExts"`
	IframeHosts []string `json:"iframeHosts"`
	Layout      string   `json:"layout"`
	BasePath    string   `json:"basePath"`
}

type PortSource string

const (
	PortSourceUnset    PortSource = "unset"
	PortSourceCLI      PortSource = "cli"
	PortSourceEnv      PortSource = "env"
	PortSourceConfig   PortSource = "config"
	PortSourceRegistry PortSource = "registry"
)

// Cfg is the configuration struct instance.
var Cfg = Config{
	PortSource:    PortSourceUnset,
	EnableWatch:   true,
	WatchSkipDirs: DefaultSkipDirs,
	PreviewExts:   DefaultPreviewExts,
	UILayout:      UILayoutCompact,
}

// PortStr returns the port string.
func (c *Config) PortStr() string {
	if c.portStr == "" {
		c.portStr = fmt.Sprintf("%d", c.PortInt)
	}
	return c.portStr
}

// SetPort sets the port integer.
func (c *Config) SetPort(port int) {
	c.PortInt = port
	c.portStr = fmt.Sprintf("%d", port)
}

// ListenAddr returns the address to listen on.
// If Private is true, only listens on localhost (127.0.0.1).
// Otherwise, listens on all interfaces (0.0.0.0).
func (c *Config) ListenAddr() string {
	if c.Private {
		return "127.0.0.1:" + c.PortStr()
	}
	return ":" + c.PortStr()
}

func (c *Config) AppConfig() AppConfig {
	previewExts := c.PreviewExts
	if len(previewExts) == 0 {
		previewExts = DefaultPreviewExts
	}
	layout := c.UILayout
	if layout == "" {
		layout = UILayoutCompact
	}
	return AppConfig{
		PreviewExts: append([]string(nil), previewExts...),
		IframeHosts: append([]string(nil), c.IframeHosts...),
		Layout:      layout,
		BasePath:    c.BasePath,
	}
}

// Init initializes the configuration.
func (c *Config) Init(targetDir, entryFile string) (err error) {
	c.TargetDir = targetDir
	if entryFile != "" {
		c.EntryFile = entryFile
	}
	if c.EntryFile == "" {
		c.EntryFile = envutil.Getenv(EnvEntry, DefaultEntry)
	}

	if !fsutil.IsDir(c.TargetDir) {
		return fmt.Errorf("target %q is not a directory", c.TargetDir)
	}

	clog.Debugf("(%s) Config: Debug=%v, Watch=%v", c.Version, EnableDebug, c.EnableWatch)

	if c.PortInt > 0 {
		c.portStr = fmt.Sprintf("%d", c.PortInt)
		return nil
	}

	if c.PortInt < 0 {
		return fmt.Errorf("port %d must be greater than 0", c.PortInt)
	}

	if c.PortSource == "" || c.PortSource == PortSourceUnset {
		c.PortSource = PortSourceUnset
		c.portStr = DefaultPort
		c.PortInt, err = strconv.Atoi(c.portStr)
		if err != nil {
			return fmt.Errorf("default port %q is not a valid integer", c.portStr)
		}
		return nil
	}

	return fmt.Errorf("port %d must be greater than 0", c.PortInt)
}
