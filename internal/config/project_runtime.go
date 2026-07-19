package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gookit/goutil/envutil"
)

type ProjectLoadOptions struct {
	GlobalMode   bool
	RegistryPort *int
	CLI          CLIConfig
	NoBrowser    bool
}

func LoadProjectRuntimeConfig(targetDir string, options ProjectLoadOptions) (Config, error) {
	dotenv, err := LoadProjectDotenv(targetDir)
	if err != nil {
		return Config{}, err
	}
	globalCfg, _, err := LoadGlobalFileConfig()
	if err != nil {
		return Config{}, err
	}
	projectCfg, _, err := LoadProjectFileConfig(targetDir)
	if err != nil {
		return Config{}, err
	}
	envCfg, err := RuntimeEnvConfig(dotenv)
	if err != nil {
		return Config{}, err
	}
	if options.GlobalMode {
		globalCfg.Server.Port = nil
		globalCfg.Server.Private = nil
		projectCfg.Server.Port = nil
		projectCfg.Server.Private = nil
		envCfg.Port = nil
		options.RegistryPort = nil
		options.CLI = CLIConfig{}
	}
	merged, err := MergeRuntimeConfig(MergeInput{
		Global:       globalCfg,
		RegistryPort: options.RegistryPort,
		Project:      projectCfg,
		Env:          envCfg,
		CLI:          options.CLI,
	})
	if err != nil {
		return Config{}, err
	}
	merged.NoBrowser = options.NoBrowser
	return merged, nil
}

func LoadProjectDotenv(targetDir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(targetDir, ".env"))
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return envutil.SplitText2map(string(data)), nil
}

func RuntimeEnvConfig(dotenv map[string]string) (EnvConfig, error) {
	port, err := ParseOptionalEnvInt(projectEnvValue(EnvPort, dotenv))
	if err != nil {
		return EnvConfig{}, err
	}
	watch, err := parseProjectEnvBool(EnvWatch, projectEnvValue(EnvWatch, dotenv))
	if err != nil {
		return EnvConfig{}, err
	}
	envCfg := EnvConfig{Port: port, Watch: watch}
	if value := projectEnvValue(EnvEntry, dotenv); value != "" {
		envCfg.Entry = &value
	}
	if value := projectEnvValue(EnvWatchDir, dotenv); value != "" {
		envCfg.WatchDir = &value
	}
	if value := projectEnvValue(EnvWatchSkipDir, dotenv); value != "" {
		envCfg.WatchSkipDir = &value
	}
	if value := projectEnvValue(EnvIncludeDir, dotenv); value != "" {
		envCfg.IncludeDir = &value
	}
	if value := projectEnvValue(EnvPreviewExts, dotenv); value != "" {
		envCfg.PreviewExts = &value
	}
	if value := projectEnvValue(EnvIframeHosts, dotenv); value != "" {
		envCfg.IframeHosts = &value
	}
	return envCfg, nil
}

func projectEnvValue(name string, dotenv map[string]string) string {
	if value, ok := dotenv[name]; ok {
		return value
	}
	return envutil.Getenv(name, "")
}

func parseProjectEnvBool(name, raw string) (*bool, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("ENV %s %q is not a valid boolean", name, raw)
	}
	return &value, nil
}
