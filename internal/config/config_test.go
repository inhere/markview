package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestConfigInitPreservesMergedEnvPortSource(t *testing.T) {
	targetDir := t.TempDir()
	err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644)
	assert.NoErr(t, err)

	cfg := Config{PortInt: 6123, PortSource: PortSourceEnv}
	err = cfg.Init(targetDir, "")

	assert.NoErr(t, err)
	assert.Eq(t, PortSourceEnv, cfg.PortSource)
	assert.Eq(t, 6123, cfg.PortInt)
	assert.Eq(t, "6123", cfg.PortStr())
}

func TestConfigInitUsesUnsetPortSourceWithoutCliOrEnvPort(t *testing.T) {
	targetDir := t.TempDir()
	err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644)
	assert.NoErr(t, err)
	t.Setenv(EnvPort, "")

	cfg := Config{}
	err = cfg.Init(targetDir, "")

	assert.NoErr(t, err)
	assert.Eq(t, PortSourceUnset, cfg.PortSource)
	assert.Eq(t, 6100, cfg.PortInt)
	assert.Eq(t, "6100", cfg.PortStr())
}

func TestConfigInitAllowsMissingEntryFile(t *testing.T) {
	targetDir := t.TempDir()

	cfg := Config{}
	err := cfg.Init(targetDir, "")

	assert.NoErr(t, err)
	assert.Eq(t, DefaultEntry, cfg.EntryFile)
	assert.Eq(t, 6100, cfg.PortInt)
}

func TestConfigInitRejectsNegativePort(t *testing.T) {
	targetDir := t.TempDir()
	err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644)
	assert.NoErr(t, err)

	cfg := Config{PortInt: -1, PortSource: PortSourceCLI}
	err = cfg.Init(targetDir, "")

	assert.Err(t, err)
	assert.StrContains(t, err.Error(), "must be greater than 0")
}

func TestConfigAppConfigCopiesPreviewExtsFromConfig(t *testing.T) {
	cfg := Config{
		PreviewExts: []string{".md", ".txt"},
		IframeHosts: []string{"intranet.local"},
	}

	appConfig := cfg.AppConfig()
	appConfig.PreviewExts[0] = ".changed"
	appConfig.IframeHosts[0] = "changed.local"

	assert.Eq(t, []string{".md", ".txt"}, cfg.PreviewExts)
	assert.Eq(t, []string{"intranet.local"}, cfg.IframeHosts)
}

func TestConfigAppConfigCopiesDefaultPreviewExts(t *testing.T) {
	originalDefaults := append([]string(nil), DefaultPreviewExts...)
	t.Cleanup(func() {
		DefaultPreviewExts = originalDefaults
	})

	cfg := Config{}
	appConfig := cfg.AppConfig()
	appConfig.PreviewExts[0] = ".changed"

	assert.Eq(t, originalDefaults, DefaultPreviewExts)
}

func TestDefaultPreviewExtsIncludesHTML(t *testing.T) {
	assert.Contains(t, DefaultPreviewExts, ".html")
}
