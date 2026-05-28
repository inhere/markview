package bootstrap

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/projects"
	"github.com/inhere/markview/internal/utils"
)

func TestStaticHandlerSetsRevalidateCacheHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	rec := httptest.NewRecorder()

	newStaticHandler(testContentFS()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
		t.Fatalf("expected revalidate cache header, got %q", got)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/css") {
		t.Fatalf("expected css content-type, got %q", contentType)
	}
}

func TestBeforeServerRunOpensBrowserByDefault(t *testing.T) {
	origNoBrowser := config.Cfg.NoBrowser
	origOpenBrowser := openBrowser
	t.Cleanup(func() {
		config.Cfg.NoBrowser = origNoBrowser
		openBrowser = origOpenBrowser
	})

	var openedURL string
	openBrowser = func(url string) error {
		openedURL = url
		return nil
	}
	config.Cfg.NoBrowser = false

	beforeServerRun(6123, false)

	assert.Eq(t, "http://127.0.0.1:6123", openedURL)
}

func TestBeforeServerRunSkipsBrowserWhenDisabled(t *testing.T) {
	origNoBrowser := config.Cfg.NoBrowser
	origOpenBrowser := openBrowser
	t.Cleanup(func() {
		config.Cfg.NoBrowser = origNoBrowser
		openBrowser = origOpenBrowser
	})

	var opened bool
	openBrowser = func(string) error {
		opened = true
		return nil
	}
	config.Cfg.NoBrowser = true

	beforeServerRun(6123, false)

	assert.False(t, opened)
}

func TestMarkPortFlagVisited(t *testing.T) {
	origPortSource := config.Cfg.PortSource
	t.Cleanup(func() {
		config.Cfg.PortSource = origPortSource
	})

	cmd := newCommand(testOptions())
	cmd.Func = nil
	err := cmd.Parse([]string{"-p", "-1"})
	assert.NoErr(t, err)

	markPortFlagVisited(cmd)

	assert.Eq(t, config.PortSourceCLI, config.Cfg.PortSource)
}

func TestProjectFlagParse(t *testing.T) {
	origSelectedProject := selectedProject
	t.Cleanup(func() {
		selectedProject = origSelectedProject
	})

	tests := []struct {
		name string
		args []string
	}{
		{name: "short project flag", args: []string{"-P", "markview"}},
		{name: "long project flag", args: []string{"--project", "markview"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedProject = ""
			cmd := newCommand(testOptions())
			cmd.Func = nil

			err := cmd.Parse(tt.args)

			assert.NoErr(t, err)
			assert.Eq(t, "markview", selectedProject)
		})
	}
}

func TestResolveSelectedProjectTarget(t *testing.T) {
	t.Run("resolves project by name", func(t *testing.T) {
		projectDir := t.TempDir()
		withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(_ string) {
			targetDir, err := resolveSelectedProjectTarget("markview")

			assert.NoErr(t, err)
			assert.Eq(t, projectDir, targetDir)
		})
	})

	t.Run("resolves project by full path", func(t *testing.T) {
		projectDir := t.TempDir()
		withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(_ string) {
			targetDir, err := resolveSelectedProjectTarget(projectDir)

			assert.NoErr(t, err)
			assert.Eq(t, projectDir, targetDir)
		})
	})

	t.Run("returns error for unknown project", func(t *testing.T) {
		withTempProjectRegistry(t, projects.Registry{}, func(_ string) {
			_, err := resolveSelectedProjectTarget("missing")

			assert.Err(t, err)
		})
	})

	t.Run("returns error for ambiguous project", func(t *testing.T) {
		registry := projects.Registry{
			"/projects/a/docs": {Name: "docs", Port: 6100, Added: "2026-05-14T15:00:00+08:00"},
			"/projects/b/docs": {Name: "docs", Port: 6101, Added: "2026-05-14T15:00:00+08:00"},
		}
		withTempProjectRegistry(t, registry, func(_ string) {
			_, err := resolveSelectedProjectTarget("docs")

			assert.Err(t, err)
		})
	})
}

func TestBuildPrepareArgsForSelectedProject(t *testing.T) {
	t.Run("uses selected project path with default entry", func(t *testing.T) {
		projectDir := t.TempDir()
		args, err := buildPrepareArgsForSelectedProject(projectDir, nil)

		assert.NoErr(t, err)
		assert.Eq(t, []string{projectDir}, args)
	})

	t.Run("uses one positional argument as entry file", func(t *testing.T) {
		projectDir := t.TempDir()
		args, err := buildPrepareArgsForSelectedProject(projectDir, []string{"guide.md"})

		assert.NoErr(t, err)
		assert.Eq(t, []string{projectDir, "guide.md"}, args)
	})

	t.Run("rejects multiple positional arguments", func(t *testing.T) {
		projectDir := t.TempDir()
		_, err := buildPrepareArgsForSelectedProject(projectDir, []string{"docs", "guide.md"})

		assert.Err(t, err)
	})
}

func TestShouldUseProjectPortRegistry(t *testing.T) {
	origPortSource := config.Cfg.PortSource
	origPortInt := config.Cfg.PortInt
	t.Cleanup(func() {
		config.Cfg.PortSource = origPortSource
		config.Cfg.PortInt = origPortInt
	})

	tests := []struct {
		name      string
		source    config.PortSource
		port      int
		shouldUse bool
	}{
		{name: "unset port uses registry", source: config.PortSourceUnset, port: 6100, shouldUse: true},
		{name: "registry port uses registry", source: config.PortSourceRegistry, port: 6100, shouldUse: true},
		{name: "CLI random uses registry", source: config.PortSourceCLI, port: -1, shouldUse: true},
		{name: "CLI fixed skips registry", source: config.PortSourceCLI, port: 8080, shouldUse: false},
		{name: "ENV fixed skips registry", source: config.PortSourceEnv, port: 8080, shouldUse: false},
		{name: "ENV random skips registry", source: config.PortSourceEnv, port: -1, shouldUse: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Cfg.PortSource = tt.source
			config.Cfg.PortInt = tt.port

			assert.Eq(t, tt.shouldUse, shouldUseProjectPortRegistry())
		})
	}
}

func TestProjectFlagKeepsPortRegistryRules(t *testing.T) {
	origSelectedProject := selectedProject
	origPortSource := config.Cfg.PortSource
	origPortInt := config.Cfg.PortInt
	t.Cleanup(func() {
		selectedProject = origSelectedProject
		config.Cfg.PortSource = origPortSource
		config.Cfg.PortInt = origPortInt
	})

	tests := []struct {
		name      string
		args      []string
		envPort   string
		shouldUse bool
	}{
		{name: "project without explicit port uses registry", args: []string{"-P", "docs"}, shouldUse: true},
		{name: "project with CLI random uses registry", args: []string{"-P", "docs", "-p", "-1"}, shouldUse: true},
		{name: "project with CLI fixed port skips registry", args: []string{"-P", "docs", "-p", "8080"}, shouldUse: false},
		{name: "project with ENV fixed port skips registry", args: []string{"-P", "docs"}, envPort: "8080", shouldUse: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedProject = ""
			config.Cfg.PortSource = config.PortSourceUnset
			config.Cfg.PortInt = 0
			t.Setenv(config.EnvPort, tt.envPort)
			cmd := newCommand(testOptions())
			cmd.Func = nil

			err := cmd.Parse(tt.args)
			assert.NoErr(t, err)
			markPortFlagVisited(cmd)
			if tt.envPort != "" && config.Cfg.PortSource != config.PortSourceCLI {
				config.Cfg.PortSource = config.PortSourceEnv
			}

			assert.Eq(t, tt.shouldUse, shouldUseProjectPortRegistry())
		})
	}
}

func TestPrepareLoadsDotenvFromTargetDir(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".env"), []byte("MKVIEW_PORT=6222\nMKVIEW_WATCH=false\n"), 0644))
	t.Setenv(config.EnvPort, "")
	t.Setenv(config.EnvWatch, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{}

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		err := prepare([]string{targetDir}, testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, 6222, config.Cfg.PortInt)
		assert.Eq(t, config.PortSourceEnv, config.Cfg.PortSource)
		assert.False(t, config.Cfg.EnableWatch)
	})
}

func TestPrepareDotenvDoesNotLeakBetweenProjects(t *testing.T) {
	projectA := t.TempDir()
	projectB := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(projectA, "README.md"), []byte("# A"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(projectA, ".env"), []byte("MKVIEW_PORT=6222\n"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(projectB, "README.md"), []byte("# B"), 0644))
	t.Setenv(config.EnvPort, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		config.Cfg = config.Config{}
		err := prepare([]string{projectA}, testContentFS())
		assert.NoErr(t, err)
		assert.Eq(t, 6222, config.Cfg.PortInt)

		config.Cfg = config.Config{}
		err = prepare([]string{projectB}, testContentFS())
		assert.NoErr(t, err)
		assert.NotEq(t, 6222, config.Cfg.PortInt)
	})
}

func TestPrepareProjectConfigPortSkipsProjectRegistryMode(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".markview.json"), []byte(`{"server":{"port":6223}}`), 0644))
	t.Setenv(config.EnvPort, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{}

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		err := prepare([]string{targetDir}, testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, 6223, config.Cfg.PortInt)
		assert.Eq(t, config.PortSourceConfig, config.Cfg.PortSource)
		assert.False(t, shouldUseProjectPortRegistry())
	})
}

func TestPrepareIsolationClearsStaleCliPortFlag(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".markview.json"), []byte(`{"server":{"port":6223}}`), 0644))
	t.Setenv(config.EnvPort, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{PortInt: 6333, PortSource: config.PortSourceCLI}
	cliPortFlagVisited = true

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		err := prepare([]string{targetDir}, testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, 6223, config.Cfg.PortInt)
		assert.Eq(t, config.PortSourceConfig, config.Cfg.PortSource)
	})
}

func TestPrepareProjectConfigPrivateCanBeOverriddenByExplicitCliFalse(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".markview.json"), []byte(`{"server":{"private":true}}`), 0644))

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		cmd := newCommand(testOptions())
		cmd.Func = nil
		assert.NoErr(t, cmd.Parse([]string{"--private=false", targetDir}))
		markPortFlagVisited(cmd)

		err := prepare(cmd.RemainArgs(), testContentFS())

		assert.NoErr(t, err)
		assert.False(t, config.Cfg.Private)
	})
}

func TestPrepareNoBrowserCliFlagSurvivesConfigMerge(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		cmd := newCommand(testOptions())
		cmd.Func = nil
		assert.NoErr(t, cmd.Parse([]string{"--no-browser", targetDir}))
		markPortFlagVisited(cmd)

		err := prepare(cmd.RemainArgs(), testContentFS())

		assert.NoErr(t, err)
		assert.True(t, config.Cfg.NoBrowser)
	})
}

func TestPrepareCliPortOverridesProjectConfigPort(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".markview.json"), []byte(`{"server":{"port":6223}}`), 0644))

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		cmd := newCommand(testOptions())
		cmd.Func = nil
		assert.NoErr(t, cmd.Parse([]string{"--port", "6333", targetDir}))
		markPortFlagVisited(cmd)

		err := prepare(cmd.RemainArgs(), testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, 6333, config.Cfg.PortInt)
		assert.Eq(t, config.PortSourceCLI, config.Cfg.PortSource)
	})
}

func TestPrepareDoesNotReusePreviousCliPortSource(t *testing.T) {
	projectA := t.TempDir()
	projectB := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(projectA, "README.md"), []byte("# A"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(projectB, "README.md"), []byte("# B"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(projectB, ".markview.json"), []byte(`{"server":{"port":6225}}`), 0644))
	t.Setenv(config.EnvPort, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		cmd := newCommand(testOptions())
		cmd.Func = nil
		assert.NoErr(t, cmd.Parse([]string{"--port", "6333", projectA}))
		markPortFlagVisited(cmd)
		assert.NoErr(t, prepare(cmd.RemainArgs(), testContentFS()))
		assert.Eq(t, config.PortSourceCLI, config.Cfg.PortSource)

		cmd = newCommand(testOptions())
		cmd.Func = nil
		assert.NoErr(t, cmd.Parse([]string{projectB}))
		markPortFlagVisited(cmd)

		err := prepare(cmd.RemainArgs(), testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, 6225, config.Cfg.PortInt)
		assert.Eq(t, config.PortSourceConfig, config.Cfg.PortSource)
	})
}

func TestPrepareUsesRegistryPortWhenNoHigherPrecedencePortExists(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, registryForTest(t, targetDir, "markview", 6224), func(_ string) {
		config.Cfg = config.Config{}

		err := prepare([]string{targetDir}, testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, 6224, config.Cfg.PortInt)
		assert.Eq(t, config.PortSourceRegistry, config.Cfg.PortSource)
		assert.True(t, shouldUseProjectPortRegistry())
	})
}

func TestPrepareDotenvDebugEnablesDebugWithoutLeakingEnv(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".env"), []byte("MKVIEW_DEBUG=true\n"), 0644))
	t.Setenv(config.EnvDebug, "")

	origCfg := config.Cfg
	origUtilsDebug := utils.EnableDebug
	origConfigDebug := config.EnableDebug
	t.Cleanup(func() {
		config.Cfg = origCfg
		utils.EnableDebug = origUtilsDebug
		config.EnableDebug = origConfigDebug
	})
	config.Cfg = config.Config{}
	utils.EnableDebug = false
	config.EnableDebug = false

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		err := prepare([]string{targetDir}, testContentFS())

		assert.NoErr(t, err)
		assert.True(t, utils.EnableDebug)
		assert.True(t, config.EnableDebug)
		assert.Eq(t, "", os.Getenv(config.EnvDebug))
	})
}

func TestPrepareEnvEntryAppliesWithoutPositionalEntry(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "guide.md"), []byte("# Guide"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".env"), []byte("MKVIEW_ENTRY=guide.md\n"), 0644))
	t.Setenv(config.EnvEntry, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		config.Cfg = config.Config{}

		err := prepare([]string{targetDir}, testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, "guide.md", config.Cfg.EntryFile)
	})
}

func TestPreparePositionalEntryOverridesDotenvEntry(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "guide.md"), []byte("# Guide"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".env"), []byte("MKVIEW_ENTRY=README.md\n"), 0644))
	t.Setenv(config.EnvEntry, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })

	withIsolatedPrepareFiles(t, projects.Registry{}, func(_ string) {
		config.Cfg = config.Config{}

		err := prepare([]string{targetDir, "guide.md"}, testContentFS())

		assert.NoErr(t, err)
		assert.Eq(t, "guide.md", config.Cfg.EntryFile)
	})
}

func TestListenNextAvailable(t *testing.T) {
	ports, closePorts := reserveConsecutivePorts(t, 5)
	closePorts()
	occupied, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ports[0]))
	assert.NoErr(t, err)
	defer occupied.Close()
	reservedPorts := map[int]struct{}{ports[1]: {}}

	listener, actualPort, err := listenNextAvailable("127.0.0.1", ports[0], len(ports), reservedPorts)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.NotEq(t, ports[0], actualPort)
	assert.NotEq(t, ports[1], actualPort)
	assert.True(t, actualPort >= ports[0])
	assert.True(t, actualPort < ports[0]+len(ports))
}

func TestListenProjectPortFromRegistryUsesSavedPort(t *testing.T) {
	targetDir := t.TempDir()
	seed, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	savedPort := seed.Addr().(*net.TCPAddr).Port
	seed.Close()

	registry := projects.Registry{}
	err = projects.Upsert(registry, targetDir, savedPort, nowForTest())
	assert.NoErr(t, err)

	listener, actualPort, err := listenProjectPortFromRegistry("127.0.0.1", targetDir, registry, true, true)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, savedPort, actualPort)
}

func TestListenProjectPortFromRegistrySkipsPortsSavedByOtherProjects(t *testing.T) {
	ports, closePorts := reserveConsecutivePorts(t, 2)
	closePorts()
	targetDir := t.TempDir()
	otherDir := t.TempDir()

	registry := projects.Registry{}
	assert.NoErr(t, projects.Upsert(registry, targetDir, ports[0], nowForTest()))
	assert.NoErr(t, projects.Upsert(registry, otherDir, ports[0], nowForTest()))

	listener, actualPort, err := listenProjectPortFromRegistry("127.0.0.1", targetDir, registry, true, true)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, ports[1], actualPort)
}

func TestReservedProjectPortsCanIncludeCurrentProjectSavedPort(t *testing.T) {
	targetDir := t.TempDir()
	otherDir := t.TempDir()

	registry := projects.Registry{}
	assert.NoErr(t, projects.Upsert(registry, targetDir, 6226, nowForTest()))
	assert.NoErr(t, projects.Upsert(registry, otherDir, 6227, nowForTest()))

	excludingCurrent := reservedProjectPorts(registry, targetDir, false)
	_, hasCurrentWhenExcluded := excludingCurrent[6226]
	_, hasOtherWhenExcluded := excludingCurrent[6227]
	assert.False(t, hasCurrentWhenExcluded)
	assert.True(t, hasOtherWhenExcluded)

	includingCurrent := reservedProjectPorts(registry, targetDir, true)
	_, hasCurrentWhenIncluded := includingCurrent[6226]
	_, hasOtherWhenIncluded := includingCurrent[6227]
	assert.True(t, hasCurrentWhenIncluded)
	assert.True(t, hasOtherWhenIncluded)
}

func TestListenProjectPortFromRegistryFallsThroughFromDefaultPort(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:6100")
	if err != nil {
		t.Skipf("port 6100 is already occupied by another process: %v", err)
	}
	defer occupied.Close()

	listener, actualPort, err := listenProjectPortFromRegistry("127.0.0.1", t.TempDir(), projects.Registry{}, true, true)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, 6101, actualPort)
}

func TestListenAndRememberProjectPortUsesHookRegistryPath(t *testing.T) {
	targetDir := t.TempDir()
	ports, closePorts := reserveConsecutivePorts(t, 1)
	defer closePorts()
	savedPort := ports[0]

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{PortSource: config.PortSourceRegistry, Private: true}
	setUserRegistryHome(t)

	withTempProjectRegistry(t, registryForTest(t, targetDir, "markview", savedPort), func(path string) {
		listener, actualPort, err := listenAndRememberProjectPort(targetDir)
		assert.NoErr(t, err)
		defer listener.Close()

		assert.True(t, actualPort > savedPort)
		assert.True(t, actualPort < savedPort+100)
		loaded, err := projects.Load(path)
		assert.NoErr(t, err)
		record, ok := loaded[projectsMustKey(t, targetDir)]
		assert.True(t, ok)
		assert.Eq(t, actualPort, record.Port)
	})
}

func TestListenAndRememberProjectPortKeepsCliRandomFromReusingSavedPort(t *testing.T) {
	targetDir := t.TempDir()
	ports, closePorts := reserveConsecutivePorts(t, 1)
	closePorts()
	savedPort := ports[0]

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{PortInt: -1, PortSource: config.PortSourceCLI, Private: true}
	setUserRegistryHome(t)

	withTempProjectRegistry(t, registryForTest(t, targetDir, "markview", savedPort), func(path string) {
		listener, actualPort, err := listenAndRememberProjectPort(targetDir)
		assert.NoErr(t, err)
		defer listener.Close()

		assert.NotEq(t, savedPort, actualPort)
		loaded, err := projects.Load(path)
		assert.NoErr(t, err)
		record, ok := loaded[projectsMustKey(t, targetDir)]
		assert.True(t, ok)
		assert.Eq(t, actualPort, record.Port)
	})
}

func reserveConsecutivePorts(t *testing.T, count int) ([]int, func()) {
	t.Helper()

	for start := 20000; start < 65000-count; start++ {
		listeners := make([]net.Listener, 0, count)
		ok := true
		for port := start; port < start+count; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				ok = false
				break
			}
			listeners = append(listeners, listener)
		}
		if !ok {
			for _, listener := range listeners {
				assert.NoErr(t, listener.Close())
			}
			continue
		}

		ports := make([]int, 0, count)
		for _, listener := range listeners {
			ports = append(ports, listener.Addr().(*net.TCPAddr).Port)
		}
		return ports, func() {
			for _, listener := range listeners {
				assert.NoErr(t, listener.Close())
			}
		}
	}

	t.Fatal("failed to reserve consecutive ports for test")
	return nil, nil
}

func setUserRegistryHome(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("APPDATA", filepath.Join(homeDir, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
}

func projectsMustKey(t *testing.T, targetDir string) string {
	t.Helper()

	key, err := projects.ProjectKey(targetDir)
	assert.NoErr(t, err)
	return key
}

func nowForTest() time.Time {
	return time.Date(2026, 5, 14, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
}

func testOptions() options {
	return options{
		Content:   testContentFS(),
		Version:   "test",
		GitHash:   "test",
		BuildTime: "test",
	}
}

func testContentFS() fstest.MapFS {
	return fstest.MapFS{
		"web/dist/app.css": {Data: []byte("body{}")},
	}
}

func withIsolatedPrepareFiles(t *testing.T, registry projects.Registry, run func(path string)) {
	t.Helper()

	origCliPortFlagVisited := cliPortFlagVisited
	origCliPrivateFlagVisited := cliPrivateFlagVisited
	t.Cleanup(func() {
		cliPortFlagVisited = origCliPortFlagVisited
		cliPrivateFlagVisited = origCliPrivateFlagVisited
	})
	cliPortFlagVisited = false
	cliPrivateFlagVisited = false

	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", t.TempDir())
	} else {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("HOME", t.TempDir())
	}
	withTempProjectRegistry(t, registry, run)
}
