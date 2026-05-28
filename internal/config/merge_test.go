package config

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestMergeRuntimeConfigPriority(t *testing.T) {
	envPort := 6204
	cliPort := 6205
	private := true
	watch := false
	watchDir := "docs,example"
	projectPort := 6203
	globalPort := 6201
	projectPreview := "append:.ini"
	projectLayout := "toc-middle"

	result, err := MergeRuntimeConfig(MergeInput{
		Global: FileConfig{Server: ServerFileConfig{Port: &globalPort}},
		RegistryPort: intPtr(6202),
		Project: FileConfig{
			Server: ServerFileConfig{Port: &projectPort, Private: &private, Watch: &watch, WatchDir: &watchDir},
			UI:     UIFileConfig{PreviewExts: &projectPreview, Layout: &projectLayout},
		},
		Env: EnvConfig{Port: &envPort},
		CLI: CLIConfig{Port: &cliPort},
	})

	assert.NoErr(t, err)
	assert.Eq(t, cliPort, result.PortInt)
	assert.Eq(t, PortSourceCLI, result.PortSource)
	assert.True(t, result.Private)
	assert.False(t, result.EnableWatch)
	assert.Eq(t, []string{"docs", "example"}, result.WatchDirs)
	assert.Eq(t, []string{".md", ".json", ".jsonl", ".yaml", ".yml", ".toml", ".ini"}, result.PreviewExts)
	assert.Eq(t, UILayoutTOCMiddle, result.UILayout)
}

func TestMergeRuntimeConfigUsesConfigPortSource(t *testing.T) {
	projectPort := 6203

	result, err := MergeRuntimeConfig(MergeInput{
		Project: FileConfig{Server: ServerFileConfig{Port: &projectPort}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, projectPort, result.PortInt)
	assert.Eq(t, PortSourceConfig, result.PortSource)
}

func TestMergeRuntimeConfigUsesRegistryBeforeGlobalPort(t *testing.T) {
	globalPort := 6201

	result, err := MergeRuntimeConfig(MergeInput{
		Global:       FileConfig{Server: ServerFileConfig{Port: &globalPort}},
		RegistryPort: intPtr(6202),
	})

	assert.NoErr(t, err)
	assert.Eq(t, 6202, result.PortInt)
	assert.Eq(t, PortSourceRegistry, result.PortSource)
}

func TestMergeRuntimeConfigWatchSkipOverrideKeepsNodeModules(t *testing.T) {
	skip := "override:.cache"

	result, err := MergeRuntimeConfig(MergeInput{
		Project: FileConfig{Server: ServerFileConfig{WatchSkipDir: &skip}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, []string{".cache", "node_modules"}, result.WatchSkipDirs)
}

func intPtr(value int) *int {
	return &value
}
