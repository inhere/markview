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
		Global:       FileConfig{Server: ServerFileConfig{Port: &globalPort}},
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

func TestMergeRuntimeConfigDoesNotRewriteNegativeCLIToRandomPort(t *testing.T) {
	cliPort := -1

	result, err := MergeRuntimeConfig(MergeInput{
		CLI: CLIConfig{Port: &cliPort},
	})

	assert.NoErr(t, err)
	assert.Eq(t, -1, result.PortInt)
	assert.Eq(t, PortSourceCLI, result.PortSource)
	assert.Eq(t, "-1", result.PortStr())
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

func TestMergeRuntimeConfigUsesEnvEntryFile(t *testing.T) {
	entry := "README.md"

	result, err := MergeRuntimeConfig(MergeInput{
		Env: EnvConfig{Entry: &entry},
	})

	assert.NoErr(t, err)
	assert.Eq(t, entry, result.EntryFile)
}

func TestMergeRuntimeConfigWatchSkipOverrideKeepsNodeModules(t *testing.T) {
	skip := "override:.cache"

	result, err := MergeRuntimeConfig(MergeInput{
		Project: FileConfig{Server: ServerFileConfig{WatchSkipDir: &skip}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, []string{".cache", "node_modules"}, result.WatchSkipDirs)
}

func TestMergeRuntimeConfigIncludeDirPriority(t *testing.T) {
	globalInclude := "append:.global"
	projectInclude := "append:.project"
	envInclude := "append:.env"

	result, err := MergeRuntimeConfig(MergeInput{
		Global:  FileConfig{Server: ServerFileConfig{IncludeDir: &globalInclude}},
		Project: FileConfig{Server: ServerFileConfig{IncludeDir: &projectInclude}},
		Env:     EnvConfig{IncludeDir: &envInclude},
	})

	assert.NoErr(t, err)
	assert.Eq(t, []string{".env"}, result.IncludeDirs)
}

func TestMergeRuntimeConfigRejectsUnsupportedIncludeDirMode(t *testing.T) {
	include := "replace:.docs"

	_, err := MergeRuntimeConfig(MergeInput{
		Project: FileConfig{Server: ServerFileConfig{IncludeDir: &include}},
	})

	assert.Err(t, err)
}

func TestMergeRuntimeConfigProjectPreviewExtsUseDefaultBase(t *testing.T) {
	t.Run("project append ignores global append additions", func(t *testing.T) {
		globalPreview := "append:.ini"
		projectPreview := "append:.conf"

		result, err := MergeRuntimeConfig(MergeInput{
			Global:  FileConfig{UI: UIFileConfig{PreviewExts: &globalPreview}},
			Project: FileConfig{UI: UIFileConfig{PreviewExts: &projectPreview}},
		})

		assert.NoErr(t, err)
		assert.Eq(t, []string{".md", ".json", ".jsonl", ".yaml", ".yml", ".toml", ".conf"}, result.PreviewExts)
	})

	t.Run("project override replaces the default list", func(t *testing.T) {
		globalPreview := "append:.ini"
		projectPreview := "override:.conf"

		result, err := MergeRuntimeConfig(MergeInput{
			Global:  FileConfig{UI: UIFileConfig{PreviewExts: &globalPreview}},
			Project: FileConfig{UI: UIFileConfig{PreviewExts: &projectPreview}},
		})

		assert.NoErr(t, err)
		assert.Eq(t, []string{".conf"}, result.PreviewExts)
	})
}

func TestMergeRuntimeConfigProjectWatchSkipUsesDefaultBase(t *testing.T) {
	globalSkip := "append:.cache"
	projectSkip := "append:coverage"

	result, err := MergeRuntimeConfig(MergeInput{
		Global:  FileConfig{Server: ServerFileConfig{WatchSkipDir: &globalSkip}},
		Project: FileConfig{Server: ServerFileConfig{WatchSkipDir: &projectSkip}},
	})

	assert.NoErr(t, err)
	assert.Eq(t, []string{"node_modules", "dist", "tmp", "temp", "coverage"}, result.WatchSkipDirs)
}

func TestMergeRuntimeConfigRejectsUnsupportedListModes(t *testing.T) {
	t.Run("preview exts", func(t *testing.T) {
		preview := "replace:.ini"

		_, err := MergeRuntimeConfig(MergeInput{
			Project: FileConfig{UI: UIFileConfig{PreviewExts: &preview}},
		})

		assert.Err(t, err)
	})

	t.Run("watch skip dir", func(t *testing.T) {
		skip := "replace:.cache"

		_, err := MergeRuntimeConfig(MergeInput{
			Project: FileConfig{Server: ServerFileConfig{WatchSkipDir: &skip}},
		})

		assert.Err(t, err)
	})
}

func intPtr(value int) *int {
	return &value
}
