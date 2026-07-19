package projects

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
)

func TestRegistryPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path, err := RegistryPath()

	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(home, ".config", "markview", "markview-projects.json"), path)
}

func TestStableIDUsesNormalizedProjectKey(t *testing.T) {
	dir := t.TempDir()

	id1, err := StableID(dir)
	assert.NoErr(t, err)
	id2, err := StableID(filepath.Join(dir, "."))
	assert.NoErr(t, err)

	assert.Eq(t, 12, len(id1))
	_, err = hex.DecodeString(id1)
	assert.NoErr(t, err)
	assert.Eq(t, id1, id2)
}

func TestBuildIndexRejectsStableIDCollision(t *testing.T) {
	original := stableID
	t.Cleanup(func() { stableID = original })
	stableID = func(string) (string, error) { return "aaaaaaaaaaaa", nil }

	_, err := BuildIndex(Registry{
		filepath.Join(t.TempDir(), "a"): {Name: "A"},
		filepath.Join(t.TempDir(), "b"): {Name: "B"},
	})

	assert.Err(t, err)
}

func TestBuildIndexKeepsMissingProject(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing")
	assert.NoErr(t, os.Mkdir(existing, 0755))
	missing := filepath.Join(dir, "missing")

	index, err := BuildIndex(Registry{
		existing: {Name: "Existing"},
		missing:  {Name: "Missing"},
	})

	assert.NoErr(t, err)
	existingID, err := StableID(existing)
	assert.NoErr(t, err)
	missingID, err := StableID(missing)
	assert.NoErr(t, err)
	assert.True(t, index[existingID].Exists)
	assert.False(t, index[missingID].Exists)
	assert.Eq(t, missing, index[missingID].Path)
}

func TestSaveRenameFailurePreservesPreviousRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markview-projects.json")
	oldData := []byte("{\n  \"old\": {\"port\": 6100}\n}\n")
	assert.NoErr(t, os.WriteFile(path, oldData, 0644))
	original := renameFile
	t.Cleanup(func() { renameFile = original })
	renameFile = func(string, string) error { return errors.New("rename failed") }

	err := Save(path, Registry{"new": {Port: 6200}})

	assert.Err(t, err)
	current, readErr := os.ReadFile(path)
	assert.NoErr(t, readErr)
	assert.Eq(t, string(oldData), string(current))
}

func TestLoad(t *testing.T) {
	t.Run("missing file returns empty registry", func(t *testing.T) {
		registry, err := Load(filepath.Join(t.TempDir(), "missing.json"))

		assert.NoErr(t, err)
		assert.Eq(t, 0, len(registry))
	})

	t.Run("valid JSON loads records", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "markview-projects.json")
		err := os.WriteFile(path, []byte(`{"/repo":{"port":6101,"name":"repo","added":"2026-05-14T15:00:00+08:00"}}`), 0644)
		assert.NoErr(t, err)

		registry, err := Load(path)

		assert.NoErr(t, err)
		assert.Eq(t, 6101, registry["/repo"].Port)
		assert.Eq(t, "repo", registry["/repo"].Name)
		assert.Eq(t, "2026-05-14T15:00:00+08:00", registry["/repo"].Added)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "markview-projects.json")
		err := os.WriteFile(path, []byte(`{bad json`), 0644)
		assert.NoErr(t, err)

		_, err = Load(path)

		assert.Err(t, err)
	})
}

func TestUpsert(t *testing.T) {
	t.Run("creates record with default project name and added time", func(t *testing.T) {
		registry := Registry{}
		targetDir := filepath.Join(t.TempDir(), "docs")
		err := os.MkdirAll(targetDir, 0755)
		assert.NoErr(t, err)
		now := time.Date(2026, 5, 14, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))

		err = Upsert(registry, targetDir, 6102, now)

		assert.NoErr(t, err)
		key, err := ProjectKey(targetDir)
		assert.NoErr(t, err)
		record := registry[key]
		assert.Eq(t, 6102, record.Port)
		assert.Eq(t, "docs", record.Name)
		assert.Eq(t, "2026-05-14T15:00:00+08:00", record.Added)
	})

	t.Run("updates port without changing name or added time", func(t *testing.T) {
		registry := Registry{}
		targetDir := t.TempDir()
		key, err := ProjectKey(targetDir)
		assert.NoErr(t, err)
		registry[key] = ProjectRecord{
			Port:  6100,
			Name:  "custom",
			Added: "2026-05-01T10:00:00+08:00",
		}

		err = Upsert(registry, targetDir, 6103, time.Now())

		assert.NoErr(t, err)
		assert.Eq(t, 6103, registry[key].Port)
		assert.Eq(t, "custom", registry[key].Name)
		assert.Eq(t, "2026-05-01T10:00:00+08:00", registry[key].Added)
	})
}

func TestLookupPort(t *testing.T) {
	targetDir := t.TempDir()
	key, err := ProjectKey(targetDir)
	assert.NoErr(t, err)
	registry := Registry{
		key: {Port: 6104, Name: "repo", Added: "2026-05-14T15:00:00+08:00"},
	}

	port, ok := LookupPort(registry, targetDir)

	assert.True(t, ok)
	assert.Eq(t, 6104, port)
}

func TestSaveCreatesParentDirectoryAndWritesRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".config", "markview", "markview-projects.json")
	registry := Registry{
		"/repo": {Port: 6105, Name: "repo", Added: "2026-05-14T15:00:00+08:00"},
	}

	err := Save(path, registry)
	assert.NoErr(t, err)

	loaded, err := Load(path)
	assert.NoErr(t, err)
	assert.Eq(t, 6105, loaded["/repo"].Port)
}

func TestList(t *testing.T) {
	registry := Registry{
		"/projects/zeta":    {Port: 6102, Name: "zeta", Added: "2026-05-14T15:00:00+08:00"},
		"/projects/alpha-b": {Port: 6103, Name: "alpha", Added: "2026-05-14T15:00:00+08:00"},
		"/projects/alpha-a": {Port: 6101, Name: "alpha", Added: "2026-05-14T15:00:00+08:00"},
	}

	entries := List(registry)

	assert.Eq(t, 3, len(entries))
	assert.Eq(t, "/projects/alpha-a", entries[0].Path)
	assert.Eq(t, "/projects/alpha-b", entries[1].Path)
	assert.Eq(t, "/projects/zeta", entries[2].Path)
}

func TestResolve(t *testing.T) {
	t.Run("finds project by name", func(t *testing.T) {
		registry := Registry{
			"/projects/markview": {Port: 6100, Name: "markview", Added: "2026-05-14T15:00:00+08:00"},
		}

		entry, err := Resolve(registry, "markview")

		assert.NoErr(t, err)
		assert.Eq(t, "/projects/markview", entry.Path)
	})

	t.Run("finds project by full path", func(t *testing.T) {
		targetDir := t.TempDir()
		key, err := ProjectKey(targetDir)
		assert.NoErr(t, err)
		registry := Registry{
			key: {Port: 6100, Name: "docs", Added: "2026-05-14T15:00:00+08:00"},
		}

		entry, err := Resolve(registry, targetDir)

		assert.NoErr(t, err)
		assert.Eq(t, key, entry.Path)
	})

	t.Run("finds project by path base name", func(t *testing.T) {
		registry := Registry{
			"/projects/docs": {Port: 6100, Name: "custom-name", Added: "2026-05-14T15:00:00+08:00"},
		}

		entry, err := Resolve(registry, "docs")

		assert.NoErr(t, err)
		assert.Eq(t, "/projects/docs", entry.Path)
	})

	t.Run("returns not found for unknown selector", func(t *testing.T) {
		_, err := Resolve(Registry{}, "missing")

		assert.Err(t, err)
		assert.Eq(t, ErrProjectNotFound, err)
	})

	t.Run("returns ambiguous when selector matches multiple projects", func(t *testing.T) {
		registry := Registry{
			"/projects/a/docs": {Port: 6100, Name: "docs", Added: "2026-05-14T15:00:00+08:00"},
			"/projects/b/docs": {Port: 6101, Name: "docs", Added: "2026-05-14T15:00:00+08:00"},
		}

		_, err := Resolve(registry, "docs")

		assert.Err(t, err)
		assert.Eq(t, ErrProjectAmbiguous, err)
	})
}

func TestRemove(t *testing.T) {
	t.Run("deletes only matched project", func(t *testing.T) {
		registry := Registry{
			"/projects/docs":  {Port: 6100, Name: "docs", Added: "2026-05-14T15:00:00+08:00"},
			"/projects/notes": {Port: 6101, Name: "notes", Added: "2026-05-14T15:00:00+08:00"},
		}

		removed, err := Remove(registry, "docs")

		assert.NoErr(t, err)
		assert.Eq(t, "/projects/docs", removed.Path)
		_, exists := registry["/projects/docs"]
		assert.False(t, exists)
		_, exists = registry["/projects/notes"]
		assert.True(t, exists)
	})

	t.Run("returns not found and ambiguous errors", func(t *testing.T) {
		_, err := Remove(Registry{}, "missing")
		assert.Eq(t, ErrProjectNotFound, err)

		registry := Registry{
			"/projects/a/docs": {Port: 6100, Name: "docs", Added: "2026-05-14T15:00:00+08:00"},
			"/projects/b/docs": {Port: 6101, Name: "docs", Added: "2026-05-14T15:00:00+08:00"},
		}

		_, err = Remove(registry, "docs")
		assert.Eq(t, ErrProjectAmbiguous, err)
		assert.Eq(t, 2, len(registry))
	})
}

func TestPruneMissing(t *testing.T) {
	existingDir := t.TempDir()
	filePath := filepath.Join(t.TempDir(), "not-dir")
	err := os.WriteFile(filePath, []byte("x"), 0644)
	assert.NoErr(t, err)
	missingDir := filepath.Join(t.TempDir(), "missing")

	registry := Registry{
		existingDir: {Port: 6100, Name: "existing", Added: "2026-05-14T15:00:00+08:00"},
		filePath:    {Port: 6101, Name: "file", Added: "2026-05-14T15:00:00+08:00"},
		missingDir:  {Port: 6102, Name: "missing", Added: "2026-05-14T15:00:00+08:00"},
	}

	removed := PruneMissing(registry)

	assert.Eq(t, 2, len(removed))
	_, exists := registry[existingDir]
	assert.True(t, exists)
	_, exists = registry[filePath]
	assert.False(t, exists)
	_, exists = registry[missingDir]
	assert.False(t, exists)
}
