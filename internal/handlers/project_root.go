package handlers

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var ErrPathOutsideProject = errors.New("path outside project")

type ProjectRoot struct {
	DisplayPath string
	RealPath    string
}

func NewProjectRoot(path string) (ProjectRoot, error) {
	displayPath, err := filepath.Abs(path)
	if err != nil {
		return ProjectRoot{}, err
	}
	displayPath = filepath.Clean(displayPath)
	realPath, err := resolveExistingPath(displayPath)
	if err != nil {
		return ProjectRoot{}, err
	}
	return ProjectRoot{DisplayPath: displayPath, RealPath: filepath.Clean(realPath)}, nil
}

func (root ProjectRoot) Resolve(urlPath string) (string, error) {
	if err := validateProjectURLPath(urlPath); err != nil {
		return "", err
	}
	candidate := filepath.Join(root.DisplayPath, filepath.FromSlash(strings.TrimPrefix(urlPath, "/")))
	realPath, err := resolveExistingPath(candidate)
	if err == nil {
		if !pathWithinRoot(root.RealPath, realPath) {
			return "", outsideProjectError(urlPath)
		}
		return filepath.Clean(realPath), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err := root.validateExistingParent(candidate, urlPath); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s: %w", candidate, fs.ErrNotExist)
}

func (root ProjectRoot) validateExistingParent(candidate, urlPath string) error {
	for parent := filepath.Dir(candidate); ; parent = filepath.Dir(parent) {
		realParent, err := resolveExistingPath(parent)
		if err == nil {
			if !pathWithinRoot(root.RealPath, realParent) {
				return outsideProjectError(urlPath)
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if filepath.Dir(parent) == parent {
			return err
		}
	}
}

func validateProjectURLPath(urlPath string) error {
	if !strings.HasPrefix(urlPath, "/") || strings.ContainsRune(urlPath, '\x00') || strings.ContainsRune(urlPath, '\\') {
		return outsideProjectError(urlPath)
	}
	for _, segment := range strings.Split(strings.TrimPrefix(urlPath, "/"), "/") {
		if segment == "." || segment == ".." {
			return outsideProjectError(urlPath)
		}
	}
	return nil
}

func pathWithinRoot(rootPath, targetPath string) bool {
	rel, err := filepath.Rel(rootPath, targetPath)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func outsideProjectError(urlPath string) error {
	return fmt.Errorf("%w: %q", ErrPathOutsideProject, urlPath)
}
