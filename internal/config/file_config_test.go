package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestFindProjectConfigUsesFirstExistingFile(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.json"), []byte(`{"server":{"port":6101}}`), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, ".markview.json"), []byte(`{"server":{"port":6102}}`), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.local.json"), []byte(`{"server":{"port":6103}}`), 0o644))

	path, ok := FindProjectConfig(dir)

	assert.True(t, ok)
	assert.Eq(t, filepath.Join(dir, "markview.local.json"), path)
}

func TestLoadFileConfigParsesPointerValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markview.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{
		"server": {"port": 0, "private": false, "watch": false},
		"ui": {"preview_exts": "append:.ini", "layout": "toc-right"}
	}`), 0o644))

	cfg, err := LoadFileConfig(path)

	assert.NoErr(t, err)
	assert.Eq(t, 0, *cfg.Server.Port)
	assert.Eq(t, false, *cfg.Server.Private)
	assert.Eq(t, false, *cfg.Server.Watch)
	assert.Eq(t, "append:.ini", *cfg.UI.PreviewExts)
	assert.Eq(t, "toc-right", *cfg.UI.Layout)
}

func TestLoadFileConfigReturnsPathInJSONError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markview.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{"server":`), 0o644))

	_, err := LoadFileConfig(path)

	assert.Err(t, err)
	assert.StrContains(t, err.Error(), path)
}

func TestNormalizeLayoutRejectsUnsupportedValue(t *testing.T) {
	_, err := NormalizeUILayout("wide")

	assert.Err(t, err)
	assert.StrContains(t, err.Error(), "compact")
}

func TestNormalizeExtListSetting(t *testing.T) {
	exts, err := NormalizeExtListSetting(DefaultPreviewExts, "append:ini,.conf")
	assert.NoErr(t, err)
	assert.Eq(t, []string{".md", ".json", ".jsonl", ".yaml", ".yml", ".toml", ".ini", ".conf"}, exts)

	exts, err = NormalizeExtListSetting(DefaultPreviewExts, "override:ini")
	assert.NoErr(t, err)
	assert.Eq(t, []string{".ini"}, exts)
}
