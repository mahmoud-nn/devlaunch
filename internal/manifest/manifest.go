package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirName       = ".devlaunch"
	FileName      = "manifest.json"
	StateFileName = "state.local.json"
)

type Manifest struct {
	Version  int       `json:"version"`
	Project  Project   `json:"project"`
	Terminal Terminal  `json:"terminal"`
	Apps     []App     `json:"apps"`
	Services []Service `json:"services"`
}

type Project struct {
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
	Platform string `json:"platform"`
}

type Terminal struct {
	Engine      string `json:"engine"`
	ReuseWindow bool   `json:"reuseWindow"`
}

type Retry struct {
	MaxAttempts int `json:"maxAttempts"`
	DelayMS     int `json:"delayMs"`
}

type Readiness struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	Port    int    `json:"port,omitempty"`
	Process string `json:"process,omitempty"`
	DelayMS int    `json:"delayMs,omitempty"`
	Retry   Retry  `json:"retry,omitempty"`
}

type StopPolicy struct {
	DefaultAction string `json:"defaultAction"`
	Command       string `json:"command,omitempty"`
}

type Launch struct {
	Strategy string `json:"strategy"`
	Path     string `json:"path,omitempty"`
	Command  string `json:"command,omitempty"`
}

type StartWhen struct {
	DelayMS int `json:"delayMs"`
}

type App struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Enabled    bool       `json:"enabled"`
	DependsOn  []string   `json:"dependsOn"`
	Launch     Launch     `json:"launch"`
	Readiness  Readiness  `json:"readiness"`
	StopPolicy StopPolicy `json:"stopPolicy"`
}

type Service struct {
	ID               string     `json:"id"`
	Kind             string     `json:"kind"`
	Interactive      bool       `json:"interactive"`
	TabName          *string    `json:"tabName"`
	WorkingDirectory string     `json:"workingDirectory"`
	Command          string     `json:"command"`
	DependsOn        []string   `json:"dependsOn"`
	Readiness        Readiness  `json:"readiness"`
	StartWhen        StartWhen  `json:"startWhen"`
	StopPolicy       StopPolicy `json:"stopPolicy"`
}

func ManifestPath(root string) string {
	return filepath.Join(root, DirName, FileName)
}

func StatePath(root string) string {
	return filepath.Join(root, DirName, StateFileName)
}

func Load(root string) (Manifest, error) {
	data, err := os.ReadFile(ManifestPath(root))
	if err != nil {
		return Manifest{}, err
	}

	var doc Manifest
	if err := json.Unmarshal(data, &doc); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	if err := Validate(doc); err != nil {
		return Manifest{}, err
	}

	return doc, nil
}

func Save(root string, doc Manifest) error {
	if err := Validate(doc); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ManifestPath(root), append(data, '\n'), 0o644)
}

func Validate(doc Manifest) error {
	if doc.Version != 1 {
		return errors.New("manifest version must be 1")
	}
	if doc.Project.Name == "" || doc.Project.RootPath == "" || doc.Project.Platform == "" {
		return errors.New("manifest.project.name, rootPath and platform are required")
	}

	ids := map[string]string{}
	for _, app := range doc.Apps {
		if app.ID == "" {
			return errors.New("app id is required")
		}
		if _, ok := ids[app.ID]; ok {
			return fmt.Errorf("duplicate id %q", app.ID)
		}
		ids[app.ID] = "app"
	}
	for _, service := range doc.Services {
		if service.ID == "" {
			return errors.New("service id is required")
		}
		if _, ok := ids[service.ID]; ok {
			return fmt.Errorf("duplicate id %q", service.ID)
		}
		ids[service.ID] = "service"
	}

	graph := map[string][]string{}
	for _, app := range doc.Apps {
		graph[app.ID] = append([]string{}, app.DependsOn...)
		for _, dep := range app.DependsOn {
			if ids[dep] != "app" {
				return fmt.Errorf("app %q depends on unknown or non-app %q", app.ID, dep)
			}
		}
	}
	for _, service := range doc.Services {
		graph[service.ID] = append([]string{}, service.DependsOn...)
		for _, dep := range service.DependsOn {
			if _, ok := ids[dep]; !ok {
				return fmt.Errorf("service %q depends on unknown dependency %q", service.ID, dep)
			}
		}
	}

	visiting := map[string]bool{}
	visited := map[string]bool{}
	var walk func(string) error
	walk = func(node string) error {
		if visited[node] {
			return nil
		}
		if visiting[node] {
			return fmt.Errorf("dependency cycle detected at %q", node)
		}
		visiting[node] = true
		for _, dep := range graph[node] {
			if err := walk(dep); err != nil {
				return err
			}
		}
		visiting[node] = false
		visited[node] = true
		return nil
	}
	for id := range graph {
		if err := walk(id); err != nil {
			return err
		}
	}
	return nil
}
