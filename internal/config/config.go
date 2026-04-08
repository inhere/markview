package config

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/envutil"
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
	portStr          string
	EnableWatch   bool
	WatchDirs     []string
	WatchSkipDirs []string
}

// Cfg is the configuration struct instance.
var Cfg = Config{
	EnableWatch: true,
	WatchSkipDirs: DefaultSkipDirs,
}

func (c *Config) PortStr() string {
	return c.portStr
}

func (c *Config) Init(targetDir, entryFile string) error {
	c.TargetDir = targetDir
	c.EntryFile = entryFile
	if entryFile == "" {
		c.EntryFile = envutil.Getenv(EnvEntry, DefaultEntry)
	}

	// Environment variables
	// EnableDebug = envutil.GetBool(EnvDebug, false)
	c.EnableWatch = envutil.GetBool(EnvWatch, true)
	clog.Debugf("Config: Debug=%v, Watch=%v", EnableDebug, c.EnableWatch)

	// port value
	if c.PortInt > 0 {
		c.portStr = fmt.Sprintf("%d", c.PortInt)
	} else {
		c.portStr = envutil.Getenv(EnvPort, DefaultPort)
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
