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
	IncludeDir   *string
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
	if fileCfg.Server.IncludeDir != nil {
		dirs, err := normalizeDirListSetting(nil, *fileCfg.Server.IncludeDir)
		if err != nil {
			return err
		}
		cfg.IncludeDirs = dirs
	}
	if fileCfg.UI.PreviewExts != nil {
		exts, err := NormalizeExtListSetting(DefaultPreviewExts, *fileCfg.UI.PreviewExts)
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
	if env.Entry != nil && strings.TrimSpace(*env.Entry) != "" {
		cfg.EntryFile = *env.Entry
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
	if env.IncludeDir != nil {
		dirs, err := normalizeDirListSetting(nil, *env.IncludeDir)
		if err != nil {
			return err
		}
		cfg.IncludeDirs = dirs
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
