# MarkView 配置文件支持一期 Implementation Plan

相关文档：

- [TODO 需求](../../TODO.md#新增支持全局和项目级别的配置文件-)
- [设计文档](../specs/2026-05-28-markview-config-files-design.md)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现全局/项目级 JSON 配置文件的一期能力：服务端配置合并、项目 `.env` 定向加载、页面配置注入、`preview_exts` 生效，以及 `layout` 的基础状态链路。

**Architecture:** `internal/config` 负责配置文件模型、查找、解析、归一化和合并，`internal/bootstrap` 只负责确定目标项目目录并把 CLI/registry 来源传入合并流程。服务端通过模板注入 `app-config-data`，前端通过 `web/src/app-config.ts` 读取运行时配置，并让链接预览和 layout dataset 使用同一份配置结果。

**Tech Stack:** Go 1.25、`encoding/json`、`os.UserConfigDir`、`github.com/gookit/goutil/envutil`、`github.com/gookit/goutil/testutil/assert`、TypeScript、Bun、JSDOM。

---

## 复审结论

设计文档覆盖了一期和二期范围，可以进入一期实施计划。需要在一期中特别处理：

- 新增端口来源 `config`，否则 `server.port` 来自配置文件时会被现有 registry 自动选端口逻辑覆盖。
- `.env` 必须在最终 `targetDir` 确定后加载，并通过 `BaseDir: targetDir` 定向加载。
- 一期只做 `layout` 的类型、默认值、本地偏好和页面 dataset 基础链路，不实现三栏视觉。
- 固定配置端口被占用时严格报错，只有 `unset`/registry 场景继续自动寻找可用端口。

## 文件结构

- Create: `internal/config/file_config.go`
  - 定义 `FileConfig`、`ServerFileConfig`、`UIFileConfig`、默认值、配置文件查找和 JSON 加载。
- Create: `internal/config/file_config_test.go`
  - 覆盖全局路径、项目配置查找优先级、JSON 解析错误、无效 layout、列表归一化。
- Create: `internal/config/merge.go`
  - 定义配置来源、合并输入、合并输出，把 defaults/global/registry/project/env/cli 合并到 `Config`。
- Create: `internal/config/merge_test.go`
  - 覆盖优先级、port source、watch skip、preview ext 的 append/override。
- Modify: `internal/config/config.go`
  - 增加 `PortSourceConfig`、`PreviewExts`、`UILayout`、`AppConfig()`，让 `Init` 更少直接读 env。
- Modify: `internal/config/consts.go`
  - 增加默认 preview ext、layout、配置文件名常量。
- Modify: `internal/bootstrap/bootstrap.go`
  - 调整 `prepare()` 顺序：先解析 target，再加载项目 `.env`，再合并配置。
- Modify: `internal/bootstrap/bootstrap_test.go`
  - 覆盖项目 `.env` 定向加载、配置端口不走 registry、CLI 覆盖配置。
- Modify: `internal/handlers/page_handler.go`
  - `PageData` 增加 `AppConfigJSON`，完整页面渲染时注入前端运行时配置。
- Modify: `web/template.html`
  - 新增 `app-config-data` script。
- Create: `web/src/app-config.ts`
  - 读取和归一化服务端注入配置。
- Create: `web/src/app-config.test.ts`
  - 覆盖默认值、扩展名归一化、layout fallback。
- Modify: `web/src/link-preview.ts`
  - 从 app config 读取 previewable extensions。
- Modify: `web/src/link-preview.test.ts`
  - 覆盖新增扩展名可预览。
- Modify: `web/src/preferences.ts`
  - 增加 `LAYOUT_MODE_STORAGE_KEY`、`LayoutMode`、layout 归一化/读写。
- Modify: `web/src/preferences.test.ts`
  - 覆盖 layout mode 偏好读取和回退。
- Modify: `web/src/app.ts`
  - 页面初始化时解析 app config，设置 layout dataset；局部导航不重置 layout。

## Task 1: 配置文件模型与读取

**Files:**
- Create: `internal/config/file_config.go`
- Create: `internal/config/file_config_test.go`
- Modify: `internal/config/consts.go`

- [x] **Step 1: 写失败测试**

在 `internal/config/file_config_test.go` 新增测试：

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestFindProjectConfigUsesFirstExistingFile(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.json"), []byte(`{"server":{"port":6101}}`), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, ".markview.json"), []byte(`{"server":{"port":6102}}`), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.local.json"), []byte(`{"server":{"port":6103}}`), 0644))

	path, ok, err := FindProjectConfig(dir)

	assert.NoErr(t, err)
	assert.True(t, ok)
	assert.Eq(t, filepath.Join(dir, "markview.local.json"), path)
}

func TestLoadFileConfigParsesPointerValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markview.json")
	assert.NoErr(t, os.WriteFile(path, []byte(`{
		"server": {"port": 0, "private": false, "watch": false},
		"ui": {"preview_exts": "append:.ini", "layout": "toc-right"}
	}`), 0644))

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
	assert.NoErr(t, os.WriteFile(path, []byte(`{"server":`), 0644))

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
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/config
```

Expected: FAIL，提示 `FindProjectConfig` 返回值/实现未匹配，或相关方法未定义。

- [x] **Step 3: 实现配置文件模型**

在 `internal/config/consts.go` 增加：

```go
const (
	GlobalConfigFile = "markview.json"
)

var ProjectConfigFiles = []string{
	"markview.local.json",
	".markview.json",
	"markview.json",
}

var DefaultPreviewExts = []string{".md", ".json", ".jsonl", ".yaml", ".yml", ".toml"}

const (
	UILayoutCompact   = "compact"
	UILayoutTOCMiddle = "toc-middle"
	UILayoutTOCRight  = "toc-right"
)
```

创建 `internal/config/file_config.go`：

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileConfig struct {
	Server ServerFileConfig `json:"server"`
	UI     UIFileConfig     `json:"ui"`
}

type ServerFileConfig struct {
	Port         *int    `json:"port"`
	Private      *bool   `json:"private"`
	Watch        *bool   `json:"watch"`
	WatchDir     *string `json:"watch_dir"`
	WatchSkipDir *string `json:"watch_skip_dir"`
}

type UIFileConfig struct {
	PreviewExts *string `json:"preview_exts"`
	Layout      *string `json:"layout"`
}

func GlobalConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "markview", GlobalConfigFile), nil
}

func FindProjectConfig(targetDir string) (string, bool, error) {
	for _, name := range ProjectConfigFiles {
		path := filepath.Join(targetDir, name)
		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() {
				continue
			}
			return path, true, nil
		}
		if os.IsNotExist(err) {
			continue
		}
		return "", false, err
	}
	return "", false, nil
}

func LoadFileConfig(path string) (FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileConfig{}, err
	}
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return cfg, nil
}

func NormalizeUILayout(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", UILayoutCompact:
		return UILayoutCompact, nil
	case UILayoutTOCMiddle:
		return UILayoutTOCMiddle, nil
	case UILayoutTOCRight:
		return UILayoutTOCRight, nil
	default:
		return "", fmt.Errorf("unsupported ui.layout %q, supported: %s, %s, %s", value, UILayoutCompact, UILayoutTOCMiddle, UILayoutTOCRight)
	}
}

func NormalizeExtListSetting(defaults []string, setting string) ([]string, error) {
	mode := "append"
	value := strings.TrimSpace(setting)
	if strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		mode = strings.TrimSpace(parts[0])
		value = strings.TrimSpace(parts[1])
	}
	if mode != "append" && mode != "override" {
		return nil, fmt.Errorf("unsupported list mode %q, supported: append, override", mode)
	}

	result := append([]string(nil), defaults...)
	if mode == "override" {
		result = nil
	}
	for _, item := range strings.Split(value, ",") {
		ext := strings.TrimSpace(item)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		result = appendUniqueString(result, strings.ToLower(ext))
	}
	return result, nil
}

func appendUniqueString(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
```

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./internal/config
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add internal/config/consts.go internal/config/file_config.go internal/config/file_config_test.go
git commit -m "feat(config): load markview config files"
```

## Task 2: 配置合并规则

**Files:**
- Create: `internal/config/merge.go`
- Create: `internal/config/merge_test.go`
- Modify: `internal/config/config.go`

- [x] **Step 1: 写失败测试**

在 `internal/config/merge_test.go` 新增：

```go
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
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/config
```

Expected: FAIL，提示 `MergeRuntimeConfig`、`MergeInput`、`PortSourceConfig`、`PreviewExts`、`UILayout` 未定义。

- [x] **Step 3: 扩展运行时 Config**

在 `internal/config/config.go` 中：

```go
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

const (
	PortSourceUnset    PortSource = "unset"
	PortSourceCLI      PortSource = "cli"
	PortSourceEnv      PortSource = "env"
	PortSourceConfig   PortSource = "config"
	PortSourceRegistry PortSource = "registry"
)
```

并更新默认值：

```go
var Cfg = Config{
	PortSource:    PortSourceUnset,
	EnableWatch:   true,
	WatchSkipDirs: DefaultSkipDirs,
	PreviewExts:   DefaultPreviewExts,
	UILayout:      UILayoutCompact,
}
```

- [x] **Step 4: 实现合并逻辑**

创建 `internal/config/merge.go`：

```go
package config

import (
	"fmt"
	"strconv"
	"strings"
)

type MergeInput struct {
	Global       FileConfig
	RegistryPort *int
	Project      FileConfig
	Env          EnvConfig
	CLI          CLIConfig
}

type EnvConfig struct {
	Port         *int
	Entry        *string
	Watch        *bool
	WatchDir     *string
	WatchSkipDir *string
}

type CLIConfig struct {
	Port    *int
	Private *bool
}

func MergeRuntimeConfig(input MergeInput) (Config, error) {
	cfg := Config{
		PortSource:    PortSourceUnset,
		EnableWatch:   true,
		WatchSkipDirs: append([]string(nil), DefaultSkipDirs...),
		PreviewExts:   append([]string(nil), DefaultPreviewExts...),
		UILayout:      UILayoutCompact,
	}

	if err := applyFileConfig(&cfg, input.Global, PortSourceConfig); err != nil {
		return Config{}, err
	}
	if input.RegistryPort != nil {
		cfg.SetPort(*input.RegistryPort)
		cfg.PortSource = PortSourceRegistry
	}
	if err := applyFileConfig(&cfg, input.Project, PortSourceConfig); err != nil {
		return Config{}, err
	}
	if err := applyEnvConfig(&cfg, input.Env); err != nil {
		return Config{}, err
	}
	applyCLIConfig(&cfg, input.CLI)
	return cfg, nil
}

func applyFileConfig(cfg *Config, fileCfg FileConfig, portSource PortSource) error {
	if fileCfg.Server.Port != nil {
		cfg.SetPort(*fileCfg.Server.Port)
		cfg.PortSource = portSource
	}
	if fileCfg.Server.Private != nil {
		cfg.Private = *fileCfg.Server.Private
	}
	if fileCfg.Server.Watch != nil {
		cfg.EnableWatch = *fileCfg.Server.Watch
	}
	if fileCfg.Server.WatchDir != nil {
		cfg.WatchDirs = splitCommaList(*fileCfg.Server.WatchDir)
	}
	if fileCfg.Server.WatchSkipDir != nil {
		dirs, err := normalizeDirListSetting(DefaultSkipDirs, *fileCfg.Server.WatchSkipDir)
		if err != nil {
			return err
		}
		cfg.WatchSkipDirs = ensureNodeModules(dirs)
	}
	if fileCfg.UI.PreviewExts != nil {
		exts, err := NormalizeExtListSetting(cfg.PreviewExts, *fileCfg.UI.PreviewExts)
		if err != nil {
			return err
		}
		cfg.PreviewExts = exts
	}
	if fileCfg.UI.Layout != nil {
		layout, err := NormalizeUILayout(*fileCfg.UI.Layout)
		if err != nil {
			return err
		}
		cfg.UILayout = layout
	}
	return nil
}

func applyEnvConfig(cfg *Config, env EnvConfig) error {
	if env.Port != nil {
		cfg.SetPort(*env.Port)
		cfg.PortSource = PortSourceEnv
	}
	if env.Watch != nil {
		cfg.EnableWatch = *env.Watch
	}
	if env.WatchDir != nil {
		cfg.WatchDirs = splitCommaList(*env.WatchDir)
	}
	if env.WatchSkipDir != nil {
		dirs, err := normalizeDirListSetting(DefaultSkipDirs, *env.WatchSkipDir)
		if err != nil {
			return err
		}
		cfg.WatchSkipDirs = ensureNodeModules(dirs)
	}
	return nil
}

func applyCLIConfig(cfg *Config, cli CLIConfig) {
	if cli.Port != nil {
		cfg.SetPort(*cli.Port)
		cfg.PortSource = PortSourceCLI
	}
	if cli.Private != nil {
		cfg.Private = *cli.Private
	}
}

func ParseOptionalEnvInt(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("ENV %s %q is not a valid integer", EnvPort, raw)
	}
	return &value, nil
}

func splitCommaList(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func normalizeDirListSetting(defaults []string, setting string) ([]string, error) {
	mode := "append"
	value := strings.TrimSpace(setting)
	if strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		mode = strings.TrimSpace(parts[0])
		value = strings.TrimSpace(parts[1])
	}
	if mode != "append" && mode != "override" {
		return nil, fmt.Errorf("unsupported list mode %q, supported: append, override", mode)
	}

	result := append([]string(nil), defaults...)
	if mode == "override" {
		result = nil
	}
	for _, item := range splitCommaList(value) {
		result = appendUniqueString(result, item)
	}
	return result, nil
}

func ensureNodeModules(dirs []string) []string {
	for _, dir := range dirs {
		if dir == "node_modules" {
			return dirs
		}
	}
	return append(dirs, "node_modules")
}
```

- [x] **Step 5: 运行测试确认通过**

Run:

```bash
go test ./internal/config
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add internal/config/config.go internal/config/merge.go internal/config/merge_test.go
git commit -m "feat(config): merge runtime config sources"
```

## Task 3: 启动流程接入

**Files:**
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `internal/bootstrap/bootstrap_test.go`
- Modify: `internal/config/config.go`

- [x] **Step 1: 写失败测试**

在 `internal/bootstrap/bootstrap_test.go` 新增：

```go
func TestPrepareLoadsDotenvFromTargetDir(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".env"), []byte("MKVIEW_PORT=6222\nMKVIEW_WATCH=false\n"), 0644))
	t.Setenv(config.EnvPort, "")
	t.Setenv(config.EnvWatch, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{}

	err := prepare([]string{targetDir}, testContentFS())

	assert.NoErr(t, err)
	assert.Eq(t, 6222, config.Cfg.PortInt)
	assert.Eq(t, config.PortSourceEnv, config.Cfg.PortSource)
	assert.False(t, config.Cfg.EnableWatch)
}

func TestPrepareProjectConfigPortSkipsProjectRegistryMode(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".markview.json"), []byte(`{"server":{"port":6223}}`), 0644))
	t.Setenv(config.EnvPort, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{}

	err := prepare([]string{targetDir}, testContentFS())

	assert.NoErr(t, err)
	assert.Eq(t, 6223, config.Cfg.PortInt)
	assert.Eq(t, config.PortSourceConfig, config.Cfg.PortSource)
	assert.False(t, shouldUseProjectPortRegistry())
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/bootstrap
```

Expected: FAIL，`.env` 仍从当前工作目录加载，配置端口仍未进入 `prepare()`。

- [x] **Step 3: 调整 `prepare()` 顺序**

在 `internal/bootstrap/bootstrap.go` 中：

- 保留目标目录/入口文件解析逻辑。
- 把 `envutil.DotenvLoad` 移到 targetDir 确定之后。
- 使用 `cfg.BaseDir = targetDir` 定向加载项目 `.env`。
- 加载全局配置、项目配置、registry port。
- 调用 `config.MergeRuntimeConfig` 生成新 `config.Cfg`。
- 最后调用 `config.Cfg.Init(targetDir, entryFile)` 做目录和入口文件校验。

目标结构：

```go
func prepare(args []string, content fs.FS) error {
	targetDir, entryFile := resolvePrepareTarget(args)
	if err := loadProjectDotenv(targetDir); err != nil {
		clog.Warnf("Failed to load dotenv: %v", err)
	}

	merged, err := buildRuntimeConfig(targetDir)
	if err != nil {
		return err
	}
	config.Cfg = merged

	utils.EnableDebug = envutil.GetBool(config.EnvDebug, false)
	config.EnableDebug = utils.EnableDebug
	if err := config.Cfg.Init(targetDir, entryFile); err != nil {
		return err
	}
	handlers.IfsReader = func(path string) ([]byte, error) {
		return fs.ReadFile(content, path)
	}
	return nil
}
```

`loadProjectDotenv` 使用：

```go
func loadProjectDotenv(targetDir string) error {
	return envutil.DotenvLoad(func(cfg *envutil.Dotenv) {
		cfg.BaseDir = targetDir
		cfg.Files = []string{".env"}
		cfg.IgnoreNotExist = true
	})
}
```

`shouldUseProjectPortRegistry()` 更新为：

```go
func shouldUseProjectPortRegistry() bool {
	return config.Cfg.PortSource == config.PortSourceUnset ||
		config.Cfg.PortSource == config.PortSourceRegistry ||
		(config.Cfg.PortSource == config.PortSourceCLI && config.Cfg.PortInt < 0)
}
```

- [x] **Step 4: 收敛 `Config.Init()` 职责**

修改 `internal/config/config.go`：

- 保留 `TargetDir`、`EntryFile` 设置。
- 保留目录和入口文件校验。
- 如果 `EntryFile` 为空，仍从 `EnvEntry` 或默认 `README.md` 读取。
- 不再覆盖 `EnableWatch`、`WatchDirs`、`WatchSkipDirs`、`PortSource` 等已经合并好的值。
- 如果 `PortInt == 0 && PortSource == PortSourceUnset`，设置默认端口 `6100`。
- 如果 `PortInt < 0`，`PortStr()` 继续返回 `"0"`。

- [x] **Step 5: 运行相关测试**

Run:

```bash
go test ./internal/config ./internal/bootstrap
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add internal/bootstrap/bootstrap.go internal/bootstrap/bootstrap_test.go internal/config/config.go
git commit -m "feat(config): apply config files during startup"
```

## Task 4: 服务端页面配置注入

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/handlers/page_handler.go`
- Modify: `internal/handlers/handlers_test.go`
- Modify: `web/template.html`

- [x] **Step 1: 写失败测试**

在 `internal/handlers/handlers_test.go` 增加完整页面渲染断言，或在已有 handler 测试中补充：

```go
func TestRenderFullPageInjectsAppConfigJSON(t *testing.T) {
	origCfg := config.Cfg
	origReader := IfsReader
	t.Cleanup(func() {
		config.Cfg = origCfg
		IfsReader = origReader
	})

	config.Cfg = config.Config{
		TargetDir:   t.TempDir(),
		PreviewExts: []string{".md", ".json", ".ini"},
		UILayout:    config.UILayoutTOCRight,
	}
	assert.NoErr(t, os.WriteFile(filepath.Join(config.Cfg.TargetDir, "README.md"), []byte("# Test"), 0644))
	IfsReader = func(path string) ([]byte, error) {
		switch path {
		case "web/template-main.html":
			return []byte(`<article id="content">{{ .Content }}</article><script id="current-file-path-data" type="application/json">{{ .CurrentFilePathJSON }}</script>`), nil
		case "web/template.html":
			return []byte(`<html><body>{{ .MainContent }}<script id="app-config-data" type="application/json">{{ .AppConfigJSON }}</script></body></html>`), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	pageData, err := buildPageData(filepath.Join(config.Cfg.TargetDir, "README.md"))
	assert.NoErr(t, err)
	rec := httptest.NewRecorder()

	renderFullPage(rec, pageData)

	body := rec.Body.String()
	assert.StrContains(t, body, `"previewExts":[".md",".json",".ini"]`)
	assert.StrContains(t, body, `"layout":"toc-right"`)
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
go test ./internal/handlers
```

Expected: FAIL，`AppConfigJSON` 或 `Config.AppConfig` 未定义。

- [x] **Step 3: 增加 AppConfig 输出**

在 `internal/config/config.go` 增加：

```go
type AppConfig struct {
	PreviewExts []string `json:"previewExts"`
	Layout      string   `json:"layout"`
}

func (c *Config) AppConfig() AppConfig {
	exts := c.PreviewExts
	if len(exts) == 0 {
		exts = DefaultPreviewExts
	}
	layout := c.UILayout
	if layout == "" {
		layout = UILayoutCompact
	}
	return AppConfig{
		PreviewExts: exts,
		Layout:      layout,
	}
}
```

在 `internal/handlers/page_handler.go`：

- `PageData` 增加 `AppConfigJSON template.JS`。
- `renderFullPage` 传入 `utils.MustMarshalJSON(config.Cfg.AppConfig())`。

在 `web/template.html` 的主脚本前增加：

```html
<script id="app-config-data" type="application/json">{{ .AppConfigJSON }}</script>
```

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./internal/handlers
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add internal/config/config.go internal/handlers/page_handler.go internal/handlers/handlers_test.go web/template.html
git commit -m "feat(config): inject app config into page"
```

## Task 5: 前端读取配置并驱动 preview_exts

**Files:**
- Create: `web/src/app-config.ts`
- Create: `web/src/app-config.test.ts`
- Modify: `web/src/link-preview.ts`
- Modify: `web/src/link-preview.test.ts`

- [x] **Step 1: 写失败测试**

创建 `web/src/app-config.test.ts`：

```ts
import { describe, expect, test } from 'bun:test';
import { JSDOM } from 'jsdom';
import {
    DEFAULT_APP_CONFIG,
    normalizeAppConfig,
    readAppConfig,
} from './app-config';

describe('app config', () => {
    test('uses defaults when script is missing', () => {
        const dom = new JSDOM('<!doctype html><html><body></body></html>');
        expect(readAppConfig(dom.window.document)).toEqual(DEFAULT_APP_CONFIG);
    });

    test('normalizes injected preview extensions and layout', () => {
        const dom = new JSDOM(`
            <script id="app-config-data" type="application/json">
                {"previewExts":["ini",".conf"],"layout":"toc-middle"}
            </script>
        `);
        expect(readAppConfig(dom.window.document)).toEqual({
            previewExts: ['.ini', '.conf'],
            layout: 'toc-middle',
        });
    });

    test('falls back for invalid layout', () => {
        expect(normalizeAppConfig({ previewExts: ['txt'], layout: 'wide' })).toEqual({
            previewExts: ['.txt'],
            layout: 'compact',
        });
    });
});
```

在 `web/src/link-preview.test.ts` 增加：

```ts
test('uses configured preview extensions', () => {
    expect(isPreviewableContentPath('/config/app.ini', ['.ini'])).toBe(true);
    expect(isPreviewableContentPath('/config/app.ini', ['.json'])).toBe(false);
});
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test
```

Expected: FAIL，`app-config` 模块不存在，`isPreviewableContentPath` 还不接受配置扩展名。

- [x] **Step 3: 实现 `app-config.ts`**

创建 `web/src/app-config.ts`：

```ts
export const APP_CONFIG_DATA_ID = 'app-config-data';
export const LAYOUT_MODES = ['compact', 'toc-middle', 'toc-right'] as const;
export type LayoutMode = (typeof LAYOUT_MODES)[number];

export interface AppConfig {
    previewExts: string[];
    layout: LayoutMode;
}

export const DEFAULT_APP_CONFIG: AppConfig = {
    previewExts: ['.md', '.json', '.jsonl', '.yaml', '.yml', '.toml'],
    layout: 'compact',
};

export function normalizeAppConfig(value: Partial<AppConfig> | null | undefined): AppConfig {
    const previewExts = Array.isArray(value?.previewExts) && value.previewExts.length > 0
        ? value.previewExts.map(normalizeExt).filter(Boolean)
        : DEFAULT_APP_CONFIG.previewExts;
    const layout = LAYOUT_MODES.includes(value?.layout as LayoutMode)
        ? value?.layout as LayoutMode
        : DEFAULT_APP_CONFIG.layout;
    return { previewExts, layout };
}

export function readAppConfig(documentRef: Document = document): AppConfig {
    const element = documentRef.getElementById(APP_CONFIG_DATA_ID);
    if (!element?.textContent) {
        return DEFAULT_APP_CONFIG;
    }
    try {
        return normalizeAppConfig(JSON.parse(element.textContent) as Partial<AppConfig>);
    } catch {
        return DEFAULT_APP_CONFIG;
    }
}

function normalizeExt(value: string): string {
    const trimmed = String(value).trim().toLowerCase();
    if (!trimmed) return '';
    return trimmed.startsWith('.') ? trimmed : `.${trimmed}`;
}
```

- [x] **Step 4: 修改 link preview 使用配置**

在 `web/src/link-preview.ts`：

```ts
import { DEFAULT_APP_CONFIG } from './app-config';

export function isPreviewableContentPath(
	pathname: string,
	previewExts = DEFAULT_APP_CONFIG.previewExts,
): boolean {
	const lowerPath = pathname.split(/[?#]/, 1)[0].toLowerCase();
	return previewExts
		.filter(ext => ext !== '.md')
		.some(ext => lowerPath.endsWith(ext));
}
```

注意：`ui.preview_exts` 可以包含 `.md`，但 `.md` 不能进入 raw content preview 判断。Markdown 链接仍按现有逻辑作为可渲染页面处理：普通点击走 inline navigation，预览按钮走 `q=main` 渲染后的预览面板。

在 `shouldShowPreviewButton`、`loadInternalContent`、`resolveNavigationTarget` 相关调用处，后续从 `app.ts` 传入读取到的 `appConfig.previewExts`。如果直接跨模块传参改动太大，可以在 `link-preview.ts` 增加：

```ts
let configuredPreviewExts = DEFAULT_APP_CONFIG.previewExts;

export function configureLinkPreview(options: { previewExts: string[] }) {
    configuredPreviewExts = options.previewExts;
}
```

并让内部默认使用 `configuredPreviewExts`。

- [x] **Step 5: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add web/src/app-config.ts web/src/app-config.test.ts web/src/link-preview.ts web/src/link-preview.test.ts
git commit -m "feat(web): read app preview config"
```

## Task 6: Layout 基础链路

**Files:**
- Modify: `web/src/preferences.ts`
- Modify: `web/src/preferences.test.ts`
- Modify: `web/src/app.ts`
- Modify: `web/src/app-config.ts`
- Modify: `web/src/app-config.test.ts`

- [x] **Step 1: 写失败测试**

在 `web/src/preferences.test.ts` 增加：

```ts
import {
    DEFAULT_LAYOUT_MODE,
    LAYOUT_MODE_STORAGE_KEY,
    normalizeLayoutMode,
    persistLayoutMode,
    readStoredPreferences,
} from './preferences';

test('normalizeLayoutMode accepts supported modes and falls back otherwise', () => {
    expect(normalizeLayoutMode('toc-middle')).toBe('toc-middle');
    expect(normalizeLayoutMode('toc-right')).toBe('toc-right');
    expect(normalizeLayoutMode('wide')).toBe(DEFAULT_LAYOUT_MODE);
});

test('readStoredPreferences includes layout mode', () => {
    const storage = new Map<string, string>([
        [LAYOUT_MODE_STORAGE_KEY, 'toc-right'],
    ]);

    const preferences = readStoredPreferences({
        getItem: key => storage.get(key) ?? null,
    });

    expect(preferences.layoutMode).toBe('toc-right');
});

test('persistLayoutMode stores supported value', () => {
    const storage = new Map<string, string>();

    persistLayoutMode('toc-middle', {
        setItem: (key, value) => storage.set(key, value),
    });

    expect(storage.get(LAYOUT_MODE_STORAGE_KEY)).toBe('toc-middle');
});
```

在 `web/src/app-config.test.ts` 增加显式断言：

```ts
test('uses compact layout when injected layout is missing', () => {
    expect(normalizeAppConfig({ previewExts: ['json'] })).toEqual({
        previewExts: ['.json'],
        layout: 'compact',
    });
});
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
cd web && bun test web/src/preferences.test.ts web/src/app-config.test.ts
```

Expected: FAIL，layout mode 偏好函数未定义。

- [x] **Step 3: 实现 layout preference**

在 `web/src/preferences.ts`：

```ts
import type { LayoutMode } from './app-config';

export const LAYOUT_MODE_STORAGE_KEY = 'markview:layout-mode';
export const DEFAULT_LAYOUT_MODE: LayoutMode = 'compact';

export function normalizeLayoutMode(value: string | null | undefined): LayoutMode {
    if (value === 'compact' || value === 'toc-middle' || value === 'toc-right') {
        return value;
    }
    return DEFAULT_LAYOUT_MODE;
}
```

更新 `readStoredPreferences` 返回：

```ts
layoutMode: normalizeLayoutMode(storage.getItem(LAYOUT_MODE_STORAGE_KEY)),
```

新增：

```ts
export function persistLayoutMode(value: LayoutMode, storage: StorageWriter = window.localStorage) {
    try {
        storage.setItem(LAYOUT_MODE_STORAGE_KEY, normalizeLayoutMode(value));
    } catch {}
}
```

- [x] **Step 4: 在 app 初始化中应用 dataset**

在 `web/src/app.ts`：

- `setupOnce()` 或 DOMContentLoaded 中读取 `const appConfig = readAppConfig();`
- 调用 `configureLinkPreview({ previewExts: appConfig.previewExts })`
- 从 `readStoredPreferences()` 得到 `layoutMode`
- 如果本地没有 layout mode，使用 `appConfig.layout`
- 设置：

```ts
function applyLayoutMode(mode: LayoutMode) {
    document.documentElement.dataset.layout = mode;
}
```

一期不改 CSS 三栏布局，只保证 dataset 存在，后续二期使用。

- [x] **Step 5: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add web/src/preferences.ts web/src/preferences.test.ts web/src/app.ts web/src/app-config.ts web/src/app-config.test.ts
git commit -m "feat(web): add layout config foundation"
```

## Task 7: 集成验证与文档收尾

**Files:**
- Modify: `docs/TODO.md`

- [x] **Step 1: 运行后端完整测试**

Run:

```bash
go test ./...
```

Expected: PASS。

- [x] **Step 2: 运行前端测试**

Run:

```bash
cd web && bun test
```

Expected: PASS。

- [x] **Step 3: 构建前端资源**

Run:

```bash
cd web && bun run build
```

Expected: PASS，并更新 `web/dist` 中的构建产物。

- [x] **Step 4: 构建 Go 程序**

Run:

```bash
go build ./...
```

Expected: PASS。

- [x] **Step 5: 手动 smoke 验证**

创建临时项目配置：

```json
{
  "server": {
    "port": 6223,
    "private": true,
    "watch": false
  },
  "ui": {
    "preview_exts": "append:.ini",
    "layout": "toc-right"
  }
}
```

验证：

- 启动后监听 `127.0.0.1:6223`。
- 页面 HTML 包含 `app-config-data`。
- `app-config-data` 中包含 `.ini` 和 `toc-right`。
- 不存在配置文件时，默认行为仍为端口 `6100` 和 `compact`。

- [x] **Step 6: 更新 TODO 说明**

由于完整三栏布局属于二期，本期不要把总 TODO 标记为完成。可以在 `docs/TODO.md` 对该项追加阶段说明：

```md
- [ ] 新增支持全局和项目级别的配置文件 `markview.json`（详细说明见下面对应章节）
  - [x] 一期：配置文件读取/合并、页面配置注入、preview_exts 生效、layout 基础链路
  - [ ] 二期：设置面板 layout 控件和完整三栏布局
```

- [x] **Step 7: 提交最终收尾**

```bash
git add docs/TODO.md web/dist
git commit -m "chore: verify config files phase one"
```

如果 `web/dist` 没有变化，只提交 `docs/TODO.md`；如果 `docs/TODO.md` 的阶段说明已经由用户提前改过，先保留用户原有内容，只补充必要 checkbox。

## 最终质量门

完成所有任务后运行：

```bash
go test ./...
cd web && bun test
go build ./...
```

预期全部 PASS。然后执行：

```bash
git status --short --branch
git pull --rebase --autostash
git push
git status --short --branch
```

预期最终状态显示 `main...origin/main` 且没有未提交的本任务变更。若仍有用户原有 `docs/TODO.md` 改动，确认没有被误提交或覆盖。

## Plan Self-Review

- Spec coverage: 一期覆盖全局/项目配置文件、配置优先级、项目 `.env`、registry port、页面注入、`preview_exts` 和 layout 基础链路。完整三栏布局明确留到二期计划。
- Placeholder scan: 本计划不包含未定项或临时代码；每个任务都有明确测试、实现方向、命令和提交点。
- Type consistency: Go 使用 `FileConfig`、`MergeInput`、`EnvConfig`、`CLIConfig`；前端使用 `AppConfig`、`LayoutMode`、`DEFAULT_APP_CONFIG`；命名在任务间保持一致。
