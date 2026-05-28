package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
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

func TestConfigInitPreservesCliPortSource(t *testing.T) {
	targetDir := t.TempDir()
	err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644)
	assert.NoErr(t, err)
	t.Setenv(EnvPort, "6123")

	cfg := Config{PortInt: -1, PortSource: PortSourceCLI}
	err = cfg.Init(targetDir, "")

	assert.NoErr(t, err)
	assert.Eq(t, PortSourceCLI, cfg.PortSource)
	assert.Eq(t, -1, cfg.PortInt)
	assert.Eq(t, "0", cfg.PortStr())
}

func TestConfigInitKeepsMergedEnvRandomPortListeningOnZero(t *testing.T) {
	targetDir := t.TempDir()
	err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644)
	assert.NoErr(t, err)

	cfg := Config{PortInt: -1, PortSource: PortSourceEnv}
	err = cfg.Init(targetDir, "")

	assert.NoErr(t, err)
	assert.Eq(t, PortSourceEnv, cfg.PortSource)
	assert.Eq(t, -1, cfg.PortInt)
	assert.Eq(t, "0", cfg.PortStr())
}
