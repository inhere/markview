package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gookit/cliui/show/table"
	"github.com/inhere/markview/internal/projects"
)

var (
	projectsAction      string
	selectedProject     string
	projectRegistryPath = projects.RegistryPath
)

func runProjectsAction(action string, args []string, out io.Writer) error {
	registryPath, registry, err := loadProjectRegistry()
	if err != nil {
		return err
	}

	switch action {
	case "list":
		renderProjectsList(out, projects.List(registry))
		return nil
	case "show":
		selector, err := requireProjectSelector(args)
		if err != nil {
			return err
		}
		entry, err := projects.Resolve(registry, selector)
		if err != nil {
			return err
		}
		renderProjectInfo(out, entry)
		return nil
	case "remove":
		selector, err := requireProjectSelector(args)
		if err != nil {
			return err
		}
		entry, err := projects.Remove(registry, selector)
		if err != nil {
			return err
		}
		if err := projects.Save(registryPath, registry); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Removed project: %s (%s)\n", entry.Record.Name, entry.Path)
		return nil
	case "prune":
		removed := projects.PruneMissing(registry)
		if err := projects.Save(registryPath, registry); err != nil {
			return err
		}
		if len(removed) == 0 {
			_, _ = fmt.Fprintln(out, "No missing project records.")
			return nil
		}
		_, _ = fmt.Fprintf(out, "Removed %d missing project records.\n", len(removed))
		return nil
	default:
		return fmt.Errorf("unknown projects action: %s", action)
	}
}

func resolveSelectedProjectTarget(selector string) (string, error) {
	_, registry, err := loadProjectRegistry()
	if err != nil {
		return "", err
	}
	entry, err := projects.Resolve(registry, selector)
	if err != nil {
		return "", err
	}
	return entry.Path, nil
}

func loadProjectRegistry() (string, projects.Registry, error) {
	registryPath, err := projectRegistryPath()
	if err != nil {
		return "", nil, err
	}
	registry, err := projects.Load(registryPath)
	if err != nil {
		return "", nil, err
	}
	return registryPath, registry, nil
}

func requireProjectSelector(args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("project selector is required")
	}
	return args[0], nil
}

func renderProjectsList(out io.Writer, entries []projects.ProjectEntry) {
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(out, "No saved projects.")
		return
	}

	tb := table.New("Saved projects")
	tb.SetHeads("NAME", "PORT", "ADDED", "PATH")
	for _, entry := range entries {
		tb.AddRow(entry.Record.Name, entry.Record.Port, entry.Record.Added, entry.Path)
	}
	_, _ = fmt.Fprint(out, tb.Render())
}

func renderProjectInfo(out io.Writer, entry projects.ProjectEntry) {
	exists := "no"
	if info, err := os.Stat(entry.Path); err == nil && info.IsDir() {
		exists = "yes"
	}

	tb := table.New("Project")
	tb.SetHeads("Name", "Value")
	tb.AddRow("Name", entry.Record.Name)
	tb.AddRow("Path", entry.Path)
	tb.AddRow("Port", entry.Record.Port)
	tb.AddRow("Added", entry.Record.Added)
	tb.AddRow("Exists", exists)
	_, _ = fmt.Fprint(out, tb.Render())
}
