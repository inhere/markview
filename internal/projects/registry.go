package projects

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	configDirName = ".config"
	appDirName    = "markview"
	registryFile  = "markview-projects.json"
)

type ProjectRecord struct {
	Port  int    `json:"port"`
	Name  string `json:"name"`
	Added string `json:"added"`
}

type Registry map[string]ProjectRecord

func RegistryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configDirName, appDirName, registryFile), nil
}

func ProjectKey(targetDir string) (string, error) {
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func Load(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Registry{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return Registry{}, nil
	}

	registry := Registry{}
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func Save(path string, registry Registry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func LookupPort(registry Registry, targetDir string) (int, bool) {
	key, err := ProjectKey(targetDir)
	if err != nil {
		return 0, false
	}
	record, ok := registry[key]
	if !ok || record.Port <= 0 {
		return 0, false
	}
	return record.Port, true
}

func Upsert(registry Registry, targetDir string, port int, now time.Time) error {
	key, err := ProjectKey(targetDir)
	if err != nil {
		return err
	}

	if record, ok := registry[key]; ok {
		record.Port = port
		registry[key] = record
		return nil
	}

	registry[key] = ProjectRecord{
		Port:  port,
		Name:  filepath.Base(key),
		Added: now.Format(time.RFC3339),
	}
	return nil
}
