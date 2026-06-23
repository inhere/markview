package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestFindProjectConfigUsesFirstExistingFile(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.json"), []byte(`{"server":{"port":6101}}`), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, ".markview.json"), []byte(`{"server":{"port":6102}}`), 0o644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.local.json"), []byte(`{"server":{"port":6103}}`), 0o644))

	path, ok, err := FindProjectConfig(dir)

	assert.NoErr(t, err)
	assert.True(t, ok)
	assert.Eq(t, filepath.Join(dir, "markview.local.json"), path)
}

func TestGlobalConfigPathUsesUserConfigDir(t *testing.T) {
	baseDir := t.TempDir()
	setUserConfigEnv(t, baseDir)
	expectedBase, err := os.UserConfigDir()
	assert.NoErr(t, err)

	path, err := GlobalConfigPath()

	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(expectedBase, "markview", GlobalConfigFile), path)
}

func TestLoadProjectFileConfigLoadsSelectedProjectFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".markview.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{"server":{"port":6102}}`), 0o644))

	cfg, ok, err := LoadProjectFileConfig(dir)

	assert.NoErr(t, err)
	assert.True(t, ok)
	assert.Eq(t, 6102, *cfg.Server.Port)
}

func TestFindProjectConfigReturnsStatError(t *testing.T) {
	invalidTargetDir := string([]byte{'b', 'a', 'd', 0})

	path, ok, err := FindProjectConfig(invalidTargetDir)

	assert.Err(t, err)
	assert.False(t, ok)
	assert.Eq(t, "", path)
	assert.False(t, os.IsNotExist(err))
}

func TestLoadProjectFileConfigIgnoresMissingConfig(t *testing.T) {
	cfg, ok, err := LoadProjectFileConfig(t.TempDir())

	assert.NoErr(t, err)
	assert.False(t, ok)
	assert.Eq(t, FileConfig{}, cfg)
}

func TestLoadGlobalFileConfigLoadsGlobalConfig(t *testing.T) {
	baseDir := t.TempDir()
	setUserConfigEnv(t, baseDir)

	configBase, err := os.UserConfigDir()
	assert.NoErr(t, err)
	configDir := filepath.Join(configBase, "markview")
	assert.NoErr(t, os.MkdirAll(configDir, 0o755))
	assert.NoErr(t, os.WriteFile(filepath.Join(configDir, GlobalConfigFile), []byte(`{"ui":{"layout":"toc-right"}}`), 0o644))

	cfg, ok, err := LoadGlobalFileConfig()

	assert.NoErr(t, err)
	assert.True(t, ok)
	assert.Eq(t, "toc-right", *cfg.UI.Layout)
}

func TestLoadFileConfigParsesPointerValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markview.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{
		"server": {"port": 0, "private": false, "watch": false, "include_dir": ".docs,.wiki"},
		"ui": {"preview_exts": "append:.ini", "layout": "toc-right", "iframe_hosts": "192.168.1.20:8080,intranet.local"}
	}`), 0o644))

	cfg, err := LoadFileConfig(path)

	assert.NoErr(t, err)
	assert.Eq(t, 0, *cfg.Server.Port)
	assert.Eq(t, false, *cfg.Server.Private)
	assert.Eq(t, false, *cfg.Server.Watch)
	assert.Eq(t, ".docs,.wiki", *cfg.Server.IncludeDir)
	assert.Eq(t, "append:.ini", *cfg.UI.PreviewExts)
	assert.Eq(t, "toc-right", *cfg.UI.Layout)
	assert.Eq(t, "192.168.1.20:8080,intranet.local", *cfg.UI.IframeHosts)
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
	assert.Eq(t, []string{".md", ".json", ".jsonl", ".yaml", ".yml", ".toml", ".html", ".ini", ".conf"}, exts)

	exts, err = NormalizeExtListSetting(DefaultPreviewExts, "override:ini")
	assert.NoErr(t, err)
	assert.Eq(t, []string{".ini"}, exts)
}

func TestNormalizeHostListSettingPreservesSuffixRules(t *testing.T) {
	hosts := NormalizeHostListSetting("*.hyy.preview.test, .corp.local, http://Api.HYY.PREVIEW.TEST/app, intranet.local")

	assert.Eq(t, []string{"*.hyy.preview.test", ".corp.local", "api.hyy.preview.test", "intranet.local"}, hosts)
}

func setUserConfigEnv(t *testing.T, baseDir string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", baseDir)
		return
	}

	t.Setenv("XDG_CONFIG_HOME", baseDir)
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
}
