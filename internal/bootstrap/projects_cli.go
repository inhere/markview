package bootstrap

import (
	"fmt"
	"io"
	"os"

	"github.com/gookit/cliui/show"
	"github.com/gookit/cliui/show/table"
	"github.com/gookit/goutil/x/ccolor"
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
	case "list", "ls":
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
	case "remove", "rm":
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
		ccolor.Fprintf(out, "<mga>Removed project: %s (%s)</>\n", entry.Record.Name, entry.Path)
		return nil
	case "prune":
		removed := projects.PruneMissing(registry)
		if err := projects.Save(registryPath, registry); err != nil {
			return err
		}
		if len(removed) == 0 {
			ccolor.Fprintln(out, "<info>No missing project records.</>")
			return nil
		}
		ccolor.Fprintf(out, "<info>Removed %d missing project records.</>\n", len(removed))
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
		ccolor.Fprintln(out, "<info>Warning:No saved projects.</>")
		return
	}

	tb := table.New("<info>Saved projects</>")
	tb.SetHeads("NAME", "PORT", "ADDED", "PATH")
	for _, entry := range entries {
		tb.AddRow(entry.Record.Name, entry.Record.Port, entry.Record.Added, entry.Path)
	}
	ccolor.Fprint(out, tb.Render())
}

func renderProjectInfo(out io.Writer, entry projects.ProjectEntry) {
	var exists bool
	if info, err := os.Stat(entry.Path); err == nil && info.IsDir() {
		exists = true
	}

	ccolor.Fprintf(out, "<info>Path</>  : %s\n", entry.Path)
	ccolor.Fprintf(out, "<info>Exists</>: %v\n", exists)
	ls := show.NewList("Information", entry.Record)
	ls.SetOutput(out)
	ls.Println()
}
