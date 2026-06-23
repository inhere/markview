package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/color"
	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/markview/internal/projects"
)

func TestRunProjectsActionListEmpty(t *testing.T) {
	withTempProjectRegistry(t, projects.Registry{}, func(path string) {
		var out bytes.Buffer

		err := runProjectsAction("list", nil, &out)

		assert.NoErr(t, err)
		assert.StrContains(t, out.String(), "No saved projects.")
		assert.Eq(t, path, mustProjectRegistryPath(t))
	})
}

func TestRunProjectsActionListUsesCliUITableFields(t *testing.T) {
	projectDir := t.TempDir()
	withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(_ string) {
		var out bytes.Buffer

		err := runProjectsAction("list", nil, &out)

		assert.NoErr(t, err)
		output := out.String()
		assert.StrContains(t, output, "Saved projects")
		assert.StrContains(t, output, "NAME")
		assert.StrContains(t, output, "PORT")
		assert.StrContains(t, output, "ADDED")
		assert.StrContains(t, output, "PATH")
		assert.StrContains(t, output, "markview")
		assert.StrContains(t, output, "2026-05-14 15:00:00")
		assert.False(t, strings.Contains(output, "2026-05-14T15:00:00+08:00"))
		assert.StrContains(t, output, projectDir)
	})
}

func disableColor() {
	color.Disable()
	ccolor.Disable()
}

func resetColor() {
	// color.Reset()
	ccolor.RevertColorSupport()
}

func TestRunProjectsActionShow(t *testing.T) {
	projectDir := t.TempDir()
	withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(_ string) {
		var out bytes.Buffer
		disableColor()
		defer resetColor()

		err := runProjectsAction("show", []string{"markview"}, &out)

		assert.NoErr(t, err)
		output := out.String()
		assert.StrContains(t, output, "name")
		assert.StrContains(t, output, "markview")
		assert.StrContains(t, output, "Path")
		assert.StrContains(t, output, projectDir)
		assert.StrContains(t, output, "port")
		assert.StrContains(t, output, "6100")
		assert.StrContains(t, output, "2026-05-14 15:00:00")
		assert.False(t, strings.Contains(output, "2026-05-14T15:00:00+08:00"))
		assert.StrContains(t, output, "Exists")
		assert.StrContains(t, output, "true")
	})
}

func TestRunProjectsActionRemove(t *testing.T) {
	projectDir := t.TempDir()
	withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(path string) {
		var out bytes.Buffer
		disableColor()
		defer resetColor()

		err := runProjectsAction("remove", []string{"markview"}, &out)

		assert.NoErr(t, err)
		assert.StrContains(t, out.String(), "Removed project")
		loaded, err := projects.Load(path)
		assert.NoErr(t, err)
		assert.Eq(t, 0, len(loaded))
	})
}

func TestRunProjectsActionPrune(t *testing.T) {
	existingDir := t.TempDir()
	missingDir := filepath.Join(t.TempDir(), "missing")
	registry := registryForTest(t, existingDir, "existing", 6100)
	registry[missingDir] = projects.ProjectRecord{Name: "missing", Port: 6101, Added: "2026-05-14T15:00:00+08:00"}

	withTempProjectRegistry(t, registry, func(path string) {
		var out bytes.Buffer
		disableColor()
		defer resetColor()

		err := runProjectsAction("prune", nil, &out)

		assert.NoErr(t, err)
		assert.StrContains(t, out.String(), "Removed 1 missing project records.")
		loaded, err := projects.Load(path)
		assert.NoErr(t, err)
		assert.Eq(t, 1, len(loaded))
		_, exists := loaded[existingDir]
		assert.True(t, exists)
	})
}

func TestRunProjectsActionErrors(t *testing.T) {
	withTempProjectRegistry(t, projects.Registry{}, func(_ string) {
		err := runProjectsAction("show", nil, &bytes.Buffer{})
		assert.Err(t, err)
		assert.True(t, strings.Contains(err.Error(), "project selector is required"))

		err = runProjectsAction("unknown", nil, &bytes.Buffer{})
		assert.Err(t, err)
		assert.True(t, strings.Contains(err.Error(), "unknown projects action"))
	})
}

func TestRunProjectsActionPruneNoMissingRecords(t *testing.T) {
	existingDir := t.TempDir()
	withTempProjectRegistry(t, registryForTest(t, existingDir, "existing", 6100), func(_ string) {
		var out bytes.Buffer

		err := runProjectsAction("prune", nil, &out)

		assert.NoErr(t, err)
		assert.StrContains(t, out.String(), "No missing project records.")
	})
}

func TestRunDispatchesProjectsActionBeforeServerStartup(t *testing.T) {
	origProjectsAction := projectsAction
	t.Cleanup(func() {
		projectsAction = origProjectsAction
	})
	projectsAction = "list"

	withTempProjectRegistry(t, projects.Registry{}, func(_ string) {
		cmd := newCommand(testOptions())

		err := cmd.Parse([]string{"--projects", "list"})

		assert.NoErr(t, err)
	})
}

func withTempProjectRegistry(t *testing.T, registry projects.Registry, run func(path string)) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "markview-projects.json")
	assert.NoErr(t, projects.Save(path, registry))

	origRegistryPath := projectRegistryPath
	t.Cleanup(func() {
		projectRegistryPath = origRegistryPath
	})
	projectRegistryPath = func() (string, error) {
		return path, nil
	}

	run(path)
}

func registryForTest(t *testing.T, projectDir string, name string, port int) projects.Registry {
	t.Helper()

	key, err := projects.ProjectKey(projectDir)
	assert.NoErr(t, err)
	return projects.Registry{
		key: {Name: name, Port: port, Added: "2026-05-14T15:00:00+08:00"},
	}
}

func mustProjectRegistryPath(t *testing.T) string {
	t.Helper()

	path, err := projectRegistryPath()
	assert.NoErr(t, err)
	return path
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
