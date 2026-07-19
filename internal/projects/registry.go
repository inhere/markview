package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

type IndexedProject struct {
	ID     string
	Path   string
	Record ProjectRecord
	Exists bool
}

type ProjectIndex map[string]IndexedProject

var (
	ErrProjectNotFound  = errors.New("project not found")
	ErrProjectAmbiguous = errors.New("project selector is ambiguous")
	stableID            = StableID
	renameFile          = os.Rename
)

type ProjectEntry struct {
	Path   string
	Record ProjectRecord
}

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

func StableID(targetDir string) (string, error) {
	key, err := ProjectKey(targetDir)
	if err != nil {
		return "", err
	}
	key = filepath.ToSlash(key)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12], nil
}

func BuildIndex(registry Registry) (ProjectIndex, error) {
	index := make(ProjectIndex, len(registry))
	for targetDir, record := range registry {
		path, err := ProjectKey(targetDir)
		if err != nil {
			return nil, err
		}
		id, err := stableID(path)
		if err != nil {
			return nil, err
		}
		if existing, ok := index[id]; ok && existing.Path != path {
			return nil, fmt.Errorf("project ID collision %s: %q and %q", id, existing.Path, path)
		}
		info, statErr := os.Stat(path)
		index[id] = IndexedProject{
			ID:     id,
			Path:   path,
			Record: record,
			Exists: statErr == nil && info.IsDir(),
		}
	}
	return index, nil
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

	tmp, err := os.CreateTemp(filepath.Dir(path), ".markview-projects-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return renameFile(tmpPath, path)
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

func List(registry Registry) []ProjectEntry {
	entries := make([]ProjectEntry, 0, len(registry))
	for path, record := range registry {
		entries = append(entries, ProjectEntry{Path: path, Record: record})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Record.Name == entries[j].Record.Name {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Record.Name < entries[j].Record.Name
	})
	return entries
}

func Resolve(registry Registry, selector string) (ProjectEntry, error) {
	matches := make([]ProjectEntry, 0, 1)
	cleanSelector := filepath.Clean(selector)
	if absSelector, err := filepath.Abs(selector); err == nil {
		cleanSelector = filepath.Clean(absSelector)
	}

	for _, entry := range List(registry) {
		if entry.Path == cleanSelector || entry.Record.Name == selector || filepath.Base(entry.Path) == selector {
			matches = append(matches, entry)
		}
	}

	if len(matches) == 0 {
		return ProjectEntry{}, ErrProjectNotFound
	}
	if len(matches) > 1 {
		return ProjectEntry{}, ErrProjectAmbiguous
	}
	return matches[0], nil
}

func Remove(registry Registry, selector string) (ProjectEntry, error) {
	entry, err := Resolve(registry, selector)
	if err != nil {
		return ProjectEntry{}, err
	}
	delete(registry, entry.Path)
	return entry, nil
}

func PruneMissing(registry Registry) []ProjectEntry {
	removed := make([]ProjectEntry, 0)
	for _, entry := range List(registry) {
		info, err := os.Stat(entry.Path)
		if err == nil && info.IsDir() {
			continue
		}
		delete(registry, entry.Path)
		removed = append(removed, entry)
	}
	return removed
}
