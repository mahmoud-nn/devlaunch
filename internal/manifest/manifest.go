package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mahmoud-nn/devlaunch/internal/schema"
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

type Policy struct {
	DefaultAction string `json:"defaultAction"`
	Command       string `json:"command,omitempty"`
}

type CheckItem struct {
	Type             string `json:"type"`
	Command          string `json:"command,omitempty"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	Port             int    `json:"port,omitempty"`
	Host             string `json:"host,omitempty"`
	Name             string `json:"name,omitempty"`
	URL              string `json:"url,omitempty"`
	ExpectStatus     []int  `json:"expectStatus,omitempty"`
	TimeoutMS        int    `json:"timeoutMs,omitempty"`
	DelayMS          int    `json:"delayMs,omitempty"`
	Retry            *Retry `json:"retry,omitempty"`
}

type CheckGroup struct {
	Mode  string      `json:"mode"`
	Items []CheckItem `json:"items"`
}

type Checks struct {
	Start  CheckGroup `json:"start"`
	Status CheckGroup `json:"status"`
}

type Launch struct {
	Strategy string `json:"strategy"`
	Path     string `json:"path,omitempty"`
	Command  string `json:"command,omitempty"`
}

type App struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	DependsOn   []string `json:"dependsOn"`
	Launch      Launch   `json:"launch"`
	StartPolicy Policy   `json:"startPolicy"`
	StopPolicy  Policy   `json:"stopPolicy"`
	Checks      Checks   `json:"checks"`
}

type Service struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind"`
	Interactive      bool     `json:"interactive"`
	TabName          *string  `json:"tabName"`
	WorkingDirectory string   `json:"workingDirectory"`
	Command          string   `json:"command"`
	DependsOn        []string `json:"dependsOn"`
	StartPolicy      Policy   `json:"startPolicy"`
	StopPolicy       Policy   `json:"stopPolicy"`
	Checks           Checks   `json:"checks"`
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
	if _, err := schema.ValidateManifestBytes(data); err != nil {
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
	if _, err := schema.ValidateManifestBytes(data); err != nil {
		return err
	}
	return os.WriteFile(ManifestPath(root), append(data, '\n'), 0o644)
}

func Validate(doc Manifest) error {
	if doc.Version != 1 {
		return errors.New("manifest version must be 1")
	}
	if doc.Project.Name == "" || doc.Project.RootPath == "" || doc.Project.Platform != "windows" {
		return errors.New("manifest.project.name, rootPath and platform=windows are required")
	}
	if doc.Terminal.Engine != "windows-terminal" {
		return errors.New("manifest.terminal.engine must be windows-terminal")
	}

	ids := map[string]string{}
	for _, app := range doc.Apps {
		if app.ID == "" {
			return errors.New("app id is required")
		}
		if _, ok := ids[app.ID]; ok {
			return fmt.Errorf("duplicate id %q", app.ID)
		}
		if err := validatePolicy(app.StartPolicy, "app", app.ID, "startPolicy"); err != nil {
			return err
		}
		if err := validatePolicy(app.StopPolicy, "app", app.ID, "stopPolicy"); err != nil {
			return err
		}
		if err := validateLaunch(app); err != nil {
			return err
		}
		if err := validateChecks(app.Checks, "app", app.ID); err != nil {
			return err
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
		if service.Interactive && (service.TabName == nil || *service.TabName == "") {
			return fmt.Errorf("service %q requires a non-empty tabName when interactive", service.ID)
		}
		if !service.Interactive && service.TabName != nil && *service.TabName == "" {
			return fmt.Errorf("service %q tabName must be null or non-empty", service.ID)
		}
		if err := validatePolicy(service.StartPolicy, "service", service.ID, "startPolicy"); err != nil {
			return err
		}
		if err := validatePolicy(service.StopPolicy, "service", service.ID, "stopPolicy"); err != nil {
			return err
		}
		if err := validateChecks(service.Checks, "service", service.ID); err != nil {
			return err
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

func validateLaunch(app App) error {
	switch app.Launch.Strategy {
	case "executable":
		if app.Launch.Path == "" {
			return fmt.Errorf("app %q requires launch.path for executable strategy", app.ID)
		}
	case "command":
		if app.Launch.Command == "" {
			return fmt.Errorf("app %q requires launch.command for command strategy", app.ID)
		}
	default:
		return fmt.Errorf("app %q has unsupported launch strategy %q", app.ID, app.Launch.Strategy)
	}
	return nil
}

func validatePolicy(policy Policy, kind, id, field string) error {
	switch policy.DefaultAction {
	case "always", "ask", "never":
	default:
		return fmt.Errorf("%s %q has unsupported %s.defaultAction %q", kind, id, field, policy.DefaultAction)
	}
	return nil
}

func validateChecks(checks Checks, kind, id string) error {
	if err := validateCheckGroup(checks.Start, kind, id, "start", true); err != nil {
		return err
	}
	if err := validateCheckGroup(checks.Status, kind, id, "status", false); err != nil {
		return err
	}
	return nil
}

func validateCheckGroup(group CheckGroup, kind, id, name string, allowFixedDelayOnly bool) error {
	if group.Mode != "all" && group.Mode != "any" {
		return fmt.Errorf("%s %q checks.%s.mode must be all or any", kind, id, name)
	}
	if len(group.Items) == 0 {
		return fmt.Errorf("%s %q checks.%s.items must not be empty", kind, id, name)
	}
	onlyFixedDelay := true
	for _, item := range group.Items {
		if item.Type != "fixed-delay" {
			onlyFixedDelay = false
		}
		if err := validateCheckItem(item, kind, id, name); err != nil {
			return err
		}
	}
	if !allowFixedDelayOnly && onlyFixedDelay {
		return fmt.Errorf("%s %q checks.%s cannot contain only fixed-delay items", kind, id, name)
	}
	return nil
}

func validateCheckItem(item CheckItem, kind, id, group string) error {
	switch item.Type {
	case "command":
		if item.Command == "" {
			return fmt.Errorf("%s %q checks.%s command item requires command", kind, id, group)
		}
	case "port":
		if item.Port <= 0 {
			return fmt.Errorf("%s %q checks.%s port item requires a valid port", kind, id, group)
		}
	case "process":
		if item.Name == "" {
			return fmt.Errorf("%s %q checks.%s process item requires name", kind, id, group)
		}
	case "http":
		if item.URL == "" {
			return fmt.Errorf("%s %q checks.%s http item requires url", kind, id, group)
		}
	case "fixed-delay":
		if item.DelayMS < 0 {
			return fmt.Errorf("%s %q checks.%s fixed-delay requires delayMs >= 0", kind, id, group)
		}
	default:
		return fmt.Errorf("%s %q checks.%s has unsupported check type %q", kind, id, group, item.Type)
	}
	return nil
}
