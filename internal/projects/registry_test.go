package projects

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
)

func TestRegistryPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path, err := RegistryPath()

	assert.NoErr(t, err)
	assert.Eq(t, filepath.Join(home, ".config", "markview", "markview-projects.json"), path)
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
