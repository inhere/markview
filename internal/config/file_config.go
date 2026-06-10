package config

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	IncludeDir   *string `json:"include_dir"`
}

type UIFileConfig struct {
	PreviewExts *string `json:"preview_exts"`
	IframeHosts *string `json:"iframe_hosts"`
	Layout      *string `json:"layout"`
}

func GlobalConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "markview", GlobalConfigFile), nil
}

func LoadGlobalFileConfig() (FileConfig, bool, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return FileConfig{}, false, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return FileConfig{}, false, nil
		}
		return FileConfig{}, false, err
	}
	cfg, err := LoadFileConfig(path)
	if err != nil {
		return FileConfig{}, false, err
	}
	return cfg, true, nil
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

func LoadProjectFileConfig(targetDir string) (FileConfig, bool, error) {
	path, ok, err := FindProjectConfig(targetDir)
	if err != nil {
		return FileConfig{}, false, err
	}
	if !ok {
		return FileConfig{}, false, nil
	}
	cfg, err := LoadFileConfig(path)
	if err != nil {
		return FileConfig{}, false, err
	}
	return cfg, true, nil
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
		return "", fmt.Errorf(
			"unsupported ui.layout %q, supported: %s, %s, %s",
			value, UILayoutCompact, UILayoutTOCMiddle, UILayoutTOCRight,
		)
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

func NormalizeHostListSetting(setting string) []string {
	result := make([]string, 0)
	for _, item := range strings.Split(setting, ",") {
		host := normalizeIframeHost(item)
		if host == "" {
			continue
		}
		result = appendUniqueString(result, host)
	}
	return result
}

func normalizeIframeHost(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil {
			return parsed.Host
		}
		return ""
	}
	value = strings.TrimPrefix(value, "//")
	if slash := strings.Index(value, "/"); slash >= 0 {
		value = value[:slash]
	}
	return strings.TrimSpace(value)
}
