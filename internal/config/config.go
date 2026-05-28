package config

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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
	TargetDir     string
	EntryFile     string
	PortInt       int
	PortSource    PortSource
	portStr       string
	EnableWatch   bool
	WatchDirs     []string
	WatchSkipDirs []string
	Private       bool
	NoBrowser     bool
	PreviewExts   []string
	UILayout      string
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
	if port < 0 {
		c.portStr = "0"
		return
	}
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

// Init initializes the configuration.
func (c *Config) Init(targetDir, entryFile string) (err error) {
	c.TargetDir = targetDir
	c.EntryFile = entryFile
	if entryFile == "" {
		c.EntryFile = envutil.Getenv(EnvEntry, DefaultEntry)
	}

	if !fsutil.IsDir(c.TargetDir) {
		return fmt.Errorf("target %q is not a directory", c.TargetDir)
	}
	entryPath := filepath.Join(c.TargetDir, c.EntryFile)
	if !fsutil.IsFile(entryPath) {
		return fmt.Errorf("entry file %q is not exist", entryPath)
	}

	// Environment variables
	// EnableDebug = envutil.GetBool(EnvDebug, false)
	c.EnableWatch = envutil.GetBool(EnvWatch, true)
	clog.Debugf("Config: Debug=%v, Watch=%v", EnableDebug, c.EnableWatch)

	// port value
	if c.PortInt > 0 {
		c.portStr = fmt.Sprintf("%d", c.PortInt)
	} else if c.PortInt < 0 {
		c.portStr = "0" // 0 表示随机端口, 后续会根据随机端口更新
	} else {
		if c.PortSource == "" {
			c.PortSource = PortSourceUnset
		}
		c.portStr = envutil.Getenv(EnvPort, DefaultPort)
		if envutil.Getenv(EnvPort, "") != "" {
			c.PortSource = PortSourceEnv
		}
		c.PortInt, err = strconv.Atoi(c.portStr)
		if err != nil {
			return fmt.Errorf("ENV MKVIEW_PORT %q is not a valid integer", c.portStr)
		}
		if c.PortInt < 0 {
			c.portStr = "0"
		}
	}

	// Watch directory. multi use comma split
	if dirstr := envutil.Getenv(EnvWatchDir, ""); dirstr != "" {
		clog.Debugf("Config: Watch directory=%s", dirstr)
		c.WatchDirs = strings.Split(dirstr, ",")
	}

	// Watch skip directory. multi use comma split
	if skipstr := envutil.Getenv(EnvWatchSkipDir, ""); skipstr != "" {
		if strings.HasPrefix(skipstr, "override") {
			c.WatchSkipDirs = strings.Split(skipstr[10:], ",")
			// Always skip node_modules dir
			if !strings.Contains(skipstr, "node_modules") {
				c.WatchSkipDirs = append(c.WatchSkipDirs, "node_modules")
			}
		} else {
			c.WatchSkipDirs = append(DefaultSkipDirs, strings.Split(skipstr, ",")...)
		}
	}

	clog.Debugf("Config: Watch skip directory=%s", strings.Join(c.WatchSkipDirs, ","))
	return nil
}
