package handlers

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestProjectRootResolveAllowsProjectFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "docs", "guide.md")
	assert.NoErr(t, os.MkdirAll(filepath.Dir(file), 0755))
	assert.NoErr(t, os.WriteFile(file, []byte("# Guide"), 0644))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)

	resolved, err := root.Resolve("/docs/guide.md")

	assert.NoErr(t, err)
	want, err := filepath.EvalSymlinks(file)
	assert.NoErr(t, err)
	assert.Eq(t, want, resolved)
}

func TestProjectRootResolveRejectsTraversal(t *testing.T) {
	root, err := NewProjectRoot(t.TempDir())
	assert.NoErr(t, err)

	for _, urlPath := range []string{
		"../secret.md",
		"/../secret.md",
		"/a/../../secret.md",
		"/a/./secret.md",
		"/a\\..\\secret.md",
		"/x\x00.md",
	} {
		t.Run(urlPath, func(t *testing.T) {
			_, err := root.Resolve(urlPath)
			assert.True(t, errors.Is(err, ErrPathOutsideProject))
		})
	}
}

func TestProjectRootResolveDoesNotDecodeAgain(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "%2e%2e", "safe.md")
	assert.NoErr(t, os.MkdirAll(filepath.Dir(file), 0755))
	assert.NoErr(t, os.WriteFile(file, []byte("safe"), 0644))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)

	resolved, err := root.Resolve("/%2e%2e/safe.md")

	assert.NoErr(t, err)
	want, err := filepath.EvalSymlinks(file)
	assert.NoErr(t, err)
	assert.Eq(t, want, resolved)
}

func TestProjectRootResolveAllowsDotDotPrefixName(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "..foo", "safe.md")
	assert.NoErr(t, os.MkdirAll(filepath.Dir(file), 0755))
	assert.NoErr(t, os.WriteFile(file, []byte("safe"), 0644))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)

	resolved, err := root.Resolve("/..foo/safe.md")

	assert.NoErr(t, err)
	want, err := filepath.EvalSymlinks(file)
	assert.NoErr(t, err)
	assert.Eq(t, want, resolved)
}

func TestProjectRootResolveMissingFileKeepsNotExist(t *testing.T) {
	root, err := NewProjectRoot(t.TempDir())
	assert.NoErr(t, err)

	_, err = root.Resolve("/docs/missing.md")

	assert.True(t, errors.Is(err, fs.ErrNotExist))
	assert.False(t, errors.Is(err, ErrPathOutsideProject))
}

func TestProjectRootResolveSymlinkBoundary(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "inside")
	outside := t.TempDir()
	assert.NoErr(t, os.Mkdir(inside, 0755))
	assert.NoErr(t, os.WriteFile(filepath.Join(inside, "safe.md"), []byte("safe"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0644))
	insideLink := filepath.Join(dir, "inside-link")
	outsideLink := filepath.Join(dir, "outside-link")
	if err := os.Symlink(inside, insideLink); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	assert.NoErr(t, os.Symlink(outside, outsideLink))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)

	resolved, err := root.Resolve("/inside-link/safe.md")
	assert.NoErr(t, err)
	want, err := filepath.EvalSymlinks(filepath.Join(inside, "safe.md"))
	assert.NoErr(t, err)
	assert.Eq(t, want, resolved)

	_, err = root.Resolve("/outside-link/secret.md")
	assert.True(t, errors.Is(err, ErrPathOutsideProject))
}

func TestProjectRootResolveJunctionBoundary(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junction test is Windows-only")
	}
	dir := t.TempDir()
	outside := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0644))
	junction := filepath.Join(dir, "outside-junction")
	output, err := exec.Command("cmd", "/c", "mklink", "/J", junction, outside).CombinedOutput()
	if err != nil {
		t.Skipf("junction unavailable: %v: %s", err, output)
	}
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)

	_, err = root.Resolve("/outside-junction/secret.md")

	assert.True(t, errors.Is(err, ErrPathOutsideProject))
}
