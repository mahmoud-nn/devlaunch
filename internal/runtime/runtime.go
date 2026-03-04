package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mahmoud-nn/devlaunch/internal/manifest"
	"github.com/mahmoud-nn/devlaunch/internal/registry"
	"github.com/mahmoud-nn/devlaunch/internal/state"
	"github.com/mahmoud-nn/devlaunch/internal/windows"
)

const (
	DefaultUIHost = "127.0.0.1"
	DefaultUIPort = 38473
)

type ProjectTarget struct {
	ID   string
	Root string
}

type ResourceStatus struct {
	ID      string       `json:"id"`
	Type    string       `json:"type"`
	Status  state.Status `json:"status"`
	Managed bool         `json:"managed"`
}

type ProjectStatus struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	RootPath    string           `json:"rootPath"`
	Status      string           `json:"status"`
	LastStartAt *time.Time       `json:"lastStartAt"`
	LastStopAt  *time.Time       `json:"lastStopAt"`
	Resources   []ResourceStatus `json:"resources"`
}

func ResolveTarget(id string) (ProjectTarget, error) {
	if id == "" {
		root, err := os.Getwd()
		if err != nil {
			return ProjectTarget{}, err
		}
		return ProjectTarget{Root: root}, nil
	}
	doc, err := registry.Load()
	if err != nil {
		return ProjectTarget{}, err
	}
	record, ok := registry.FindByID(doc, id)
	if !ok {
		return ProjectTarget{}, fmt.Errorf("project %q not found in registry", id)
	}
	return ProjectTarget{ID: record.ID, Root: record.RootPath}, nil
}

func InitProject(root string) (manifest.Manifest, state.State, error) {
	root = filepath.Clean(root)
	projectName := filepath.Base(root)

	doc, err := ensureManifest(root, projectName)
	if err != nil {
		return manifest.Manifest{}, state.State{}, err
	}

	st, err := ensureState(root, doc.Project.Name)
	if err != nil {
		return manifest.Manifest{}, state.State{}, err
	}

	if _, err := Register(root, registry.ProjectStatusStopped); err != nil {
		return manifest.Manifest{}, state.State{}, err
	}

	return doc, st, nil
}

func ensureManifest(root, projectName string) (manifest.Manifest, error) {
	path := manifest.ManifestPath(root)
	if _, err := os.Stat(path); err == nil {
		return manifest.Load(root)
	} else if !errors.Is(err, os.ErrNotExist) {
		return manifest.Manifest{}, err
	}

	doc := manifest.Manifest{
		Version: 1,
		Project: manifest.Project{
			Name:     projectName,
			RootPath: root,
			Platform: "windows",
		},
		Terminal: manifest.Terminal{
			Engine:      "windows-terminal",
			ReuseWindow: false,
		},
		Apps:     detectApps(),
		Services: detectServices(root),
	}
	if err := manifest.Save(root, doc); err != nil {
		return manifest.Manifest{}, err
	}
	return doc, nil
}

func ensureState(root, projectName string) (state.State, error) {
	path := manifest.StatePath(root)
	if _, err := os.Stat(path); err == nil {
		st := state.Load(root, projectName)
		if st.ProjectName == "" {
			st.ProjectName = projectName
			if err := state.Save(root, st); err != nil {
				return state.State{}, err
			}
		}
		return st, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return state.State{}, err
	}

	st := state.Default(projectName)
	if err := state.Save(root, st); err != nil {
		return state.State{}, err
	}
	return st, nil
}

func Register(root string, status registry.ProjectStatus) (registry.ProjectRecord, error) {
	doc, err := manifest.Load(root)
	if err != nil {
		return registry.ProjectRecord{}, err
	}

	record := registry.ProjectRecord{
		ID:              doc.Project.Name,
		Name:            doc.Project.Name,
		RootPath:        doc.Project.RootPath,
		ManifestPath:    manifest.ManifestPath(root),
		LastSeenAt:      time.Now().UTC(),
		LastKnownStatus: status,
	}

	reg, err := registry.Load()
	if err != nil {
		return registry.ProjectRecord{}, err
	}
	reg = registry.Upsert(reg, record)
	if err := registry.Save(reg); err != nil {
		return registry.ProjectRecord{}, err
	}
	return record, nil
}

func ListProjects() ([]registry.ProjectRecord, error) {
	doc, err := registry.Load()
	if err != nil {
		return nil, err
	}
	sort.Slice(doc.Projects, func(i, j int) bool {
		return strings.ToLower(doc.Projects[i].Name) < strings.ToLower(doc.Projects[j].Name)
	})
	return doc.Projects, nil
}

func Status(target ProjectTarget) (ProjectStatus, error) {
	doc, st, err := loadAndReconcile(target)
	if err != nil {
		return ProjectStatus{}, err
	}
	current := summarize(doc, st)
	if _, err := Register(doc.Project.RootPath, registry.ProjectStatus(current.Status)); err != nil {
		return ProjectStatus{}, err
	}
	return current, nil
}

func Start(target ProjectTarget) (ProjectStatus, error) {
	doc, st, err := loadAndReconcile(target)
	if err != nil {
		return ProjectStatus{}, err
	}

	appOrder, serviceOrder, err := dependencyOrder(doc)
	if err != nil {
		return ProjectStatus{}, err
	}

	for _, app := range appOrder {
		if !app.Enabled || st.Resources[app.ID].Status == state.StatusRunning {
			continue
		}
		result, err := startApp(doc.Project.RootPath, app)
		if err != nil {
			st.Resources[app.ID] = failedResource("app")
			_ = state.Save(doc.Project.RootPath, st)
			return ProjectStatus{}, fmt.Errorf("start app %s: %w", app.ID, err)
		}
		st.Resources[app.ID] = state.ResourceState{
			Type:             "app",
			Status:           state.StatusRunning,
			StartedByRuntime: true,
			LastKnownPID:     result.PID,
			LastSeenAt:       time.Now().UTC(),
		}
	}

	for _, service := range serviceOrder {
		if st.Resources[service.ID].Status == state.StatusRunning {
			continue
		}
		result, err := startService(service)
		if err != nil {
			st.Resources[service.ID] = failedResource("service")
			_ = state.Save(doc.Project.RootPath, st)
			return ProjectStatus{}, fmt.Errorf("start service %s: %w", service.ID, err)
		}
		resource := state.ResourceState{
			Type:             "service",
			Status:           state.StatusRunning,
			StartedByRuntime: true,
			LastKnownPID:     result.PID,
			LastSeenAt:       time.Now().UTC(),
		}
		if service.TabName != nil {
			resource.TerminalTabName = *service.TabName
		}
		st.Resources[service.ID] = resource
	}

	now := time.Now().UTC()
	st.LastStartAt = &now
	if err := state.Save(doc.Project.RootPath, st); err != nil {
		return ProjectStatus{}, err
	}
	if _, err := Register(doc.Project.RootPath, registry.ProjectStatusRunning); err != nil {
		return ProjectStatus{}, err
	}
	return summarize(doc, st), nil
}

func Stop(target ProjectTarget) (ProjectStatus, error) {
	doc, st, err := loadAndReconcile(target)
	if err != nil {
		return ProjectStatus{}, err
	}

	for i := len(doc.Services) - 1; i >= 0; i-- {
		service := doc.Services[i]
		if err := stopService(service); err != nil {
			st.Resources[service.ID] = failedResource("service")
			_ = state.Save(doc.Project.RootPath, st)
			return ProjectStatus{}, fmt.Errorf("stop service %s: %w", service.ID, err)
		}
		st.Resources[service.ID] = state.ResourceState{
			Type:             "service",
			Status:           state.StatusStopped,
			StartedByRuntime: st.Resources[service.ID].StartedByRuntime,
			LastSeenAt:       time.Now().UTC(),
		}
	}

	now := time.Now().UTC()
	st.LastStopAt = &now
	if err := state.Save(doc.Project.RootPath, st); err != nil {
		return ProjectStatus{}, err
	}
	if _, err := Register(doc.Project.RootPath, registry.ProjectStatusStopped); err != nil {
		return ProjectStatus{}, err
	}
	return summarize(doc, st), nil
}

func OpenProjectFolder(target ProjectTarget) error {
	if target.Root == "" {
		return errors.New("missing project root")
	}
	return windows.OpenFolder(target.Root)
}

func loadAndReconcile(target ProjectTarget) (manifest.Manifest, state.State, error) {
	root := filepath.Clean(target.Root)
	doc, err := manifest.Load(root)
	if err != nil {
		return manifest.Manifest{}, state.State{}, err
	}
	st := reconcile(doc, state.Load(root, doc.Project.Name))
	if err := state.Save(root, st); err != nil {
		return manifest.Manifest{}, state.State{}, err
	}
	return doc, st, nil
}

func summarize(doc manifest.Manifest, st state.State) ProjectStatus {
	statusValue := "stopped"
	resources := make([]ResourceStatus, 0, len(doc.Apps)+len(doc.Services))

	for _, app := range doc.Apps {
		resource := st.Resources[app.ID]
		resources = append(resources, ResourceStatus{
			ID:      app.ID,
			Type:    "app",
			Status:  resource.Status,
			Managed: resource.StartedByRuntime,
		})
		if resource.Status == state.StatusRunning {
			statusValue = "running"
		}
	}

	for _, service := range doc.Services {
		resource := st.Resources[service.ID]
		resources = append(resources, ResourceStatus{
			ID:      service.ID,
			Type:    "service",
			Status:  resource.Status,
			Managed: resource.StartedByRuntime,
		})
		if resource.Status == state.StatusRunning {
			statusValue = "running"
		}
	}

	if len(resources) == 0 {
		statusValue = "unknown"
	}

	return ProjectStatus{
		ID:          doc.Project.Name,
		Name:        doc.Project.Name,
		RootPath:    doc.Project.RootPath,
		Status:      statusValue,
		LastStartAt: st.LastStartAt,
		LastStopAt:  st.LastStopAt,
		Resources:   resources,
	}
}

func reconcile(doc manifest.Manifest, st state.State) state.State {
	if st.Resources == nil {
		st.Resources = map[string]state.ResourceState{}
	}

	for _, app := range doc.Apps {
		resource := st.Resources[app.ID]
		if resource.LastKnownPID > 0 && windows.IsProcessRunning(resource.LastKnownPID) {
			resource.Status = state.StatusRunning
		} else if resource.Status == state.StatusRunning {
			resource.Status = state.StatusUnknown
		} else if resource.Status == "" {
			resource.Status = state.StatusStopped
		}
		resource.Type = "app"
		resource.LastSeenAt = time.Now().UTC()
		st.Resources[app.ID] = resource
	}

	for _, service := range doc.Services {
		resource := st.Resources[service.ID]
		if !service.Interactive && resource.LastKnownPID > 0 && windows.IsProcessRunning(resource.LastKnownPID) {
			resource.Status = state.StatusRunning
		} else if !service.Interactive && resource.Status == state.StatusRunning {
			resource.Status = state.StatusUnknown
		} else if resource.Status == "" {
			resource.Status = state.StatusStopped
		}
		resource.Type = "service"
		resource.LastSeenAt = time.Now().UTC()
		st.Resources[service.ID] = resource
	}

	return st
}

func dependencyOrder(doc manifest.Manifest) ([]manifest.App, []manifest.Service, error) {
	apps := map[string]manifest.App{}
	services := map[string]manifest.Service{}
	for _, app := range doc.Apps {
		apps[app.ID] = app
	}
	for _, service := range doc.Services {
		services[service.ID] = service
	}

	order := []string{}
	visited := map[string]bool{}
	visiting := map[string]bool{}
	var walk func(string) error
	walk = func(id string) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("dependency cycle detected at %q", id)
		}
		visiting[id] = true
		if app, ok := apps[id]; ok {
			for _, dep := range app.DependsOn {
				if err := walk(dep); err != nil {
					return err
				}
			}
		}
		if service, ok := services[id]; ok {
			for _, dep := range service.DependsOn {
				if err := walk(dep); err != nil {
					return err
				}
			}
		}
		visiting[id] = false
		visited[id] = true
		order = append(order, id)
		return nil
	}

	for _, app := range doc.Apps {
		if err := walk(app.ID); err != nil {
			return nil, nil, err
		}
	}
	for _, service := range doc.Services {
		if err := walk(service.ID); err != nil {
			return nil, nil, err
		}
	}

	appOrder := []manifest.App{}
	serviceOrder := []manifest.Service{}
	for _, id := range order {
		if app, ok := apps[id]; ok {
			appOrder = append(appOrder, app)
		}
		if service, ok := services[id]; ok {
			serviceOrder = append(serviceOrder, service)
		}
	}
	return appOrder, serviceOrder, nil
}

func startApp(root string, app manifest.App) (windows.CommandResult, error) {
	switch app.Launch.Strategy {
	case "executable":
		result, err := windows.LaunchExecutable(app.Launch.Path)
		if err != nil {
			return windows.CommandResult{}, err
		}
		if err := waitReadiness(root, app.Readiness); err != nil {
			return windows.CommandResult{}, err
		}
		return result, nil
	case "command":
		result, err := windows.RunBackground(root, app.Launch.Command)
		if err != nil {
			return windows.CommandResult{}, err
		}
		if err := waitReadiness(root, app.Readiness); err != nil {
			return windows.CommandResult{}, err
		}
		return result, nil
	default:
		return windows.CommandResult{}, fmt.Errorf("unsupported app launch strategy %q", app.Launch.Strategy)
	}
}

func startService(service manifest.Service) (windows.CommandResult, error) {
	if service.StartWhen.DelayMS > 0 {
		time.Sleep(time.Duration(service.StartWhen.DelayMS) * time.Millisecond)
	}
	if service.Interactive {
		tabName := service.ID
		if service.TabName != nil && *service.TabName != "" {
			tabName = *service.TabName
		}
		if err := windows.LaunchInteractiveTab(service.WorkingDirectory, tabName, service.Command); err != nil {
			return windows.CommandResult{}, err
		}
		if err := waitReadiness(service.WorkingDirectory, service.Readiness); err != nil {
			return windows.CommandResult{}, err
		}
		return windows.CommandResult{}, nil
	}

	result, err := windows.RunBackground(service.WorkingDirectory, service.Command)
	if err != nil {
		return windows.CommandResult{}, err
	}
	if err := waitReadiness(service.WorkingDirectory, service.Readiness); err != nil {
		return windows.CommandResult{}, err
	}
	return result, nil
}

func stopService(service manifest.Service) error {
	if service.StopPolicy.Command == "" {
		return nil
	}
	return windows.RunForeground(service.WorkingDirectory, service.StopPolicy.Command)
}

func waitReadiness(workingDir string, readiness manifest.Readiness) error {
	attempts := readiness.Retry.MaxAttempts
	if attempts == 0 {
		attempts = 1
	}
	delay := time.Duration(readiness.Retry.DelayMS) * time.Millisecond
	if delay == 0 {
		delay = 2 * time.Second
	}

	switch readiness.Type {
	case "", "fixed-delay":
		duration := readiness.DelayMS
		if duration == 0 {
			duration = 1000
		}
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return nil
	case "command":
		for i := 0; i < attempts; i++ {
			if err := windows.RunCheck(workingDir, readiness.Command); err == nil {
				return nil
			}
			time.Sleep(delay)
		}
		return fmt.Errorf("readiness command failed: %s", readiness.Command)
	case "port":
		return windows.WaitForPort(readiness.Port, attempts, delay)
	case "process":
		for i := 0; i < attempts; i++ {
			if windows.ProcessExistsByName(readiness.Process) {
				return nil
			}
			time.Sleep(delay)
		}
		return fmt.Errorf("process readiness failed: %s", readiness.Process)
	default:
		return fmt.Errorf("unsupported readiness type %q", readiness.Type)
	}
}

func detectApps() []manifest.App {
	dockerPath := `C:\Program Files\Docker\Docker\Docker Desktop.exe`
	if _, err := os.Stat(dockerPath); err == nil {
		return []manifest.App{
			{
				ID:        "docker-desktop",
				Kind:      "desktop-app",
				Enabled:   true,
				DependsOn: []string{},
				Launch: manifest.Launch{
					Strategy: "executable",
					Path:     dockerPath,
				},
				Readiness: manifest.Readiness{
					Type:    "command",
					Command: "docker info",
					Retry: manifest.Retry{
						MaxAttempts: 60,
						DelayMS:     2000,
					},
				},
				StopPolicy: manifest.StopPolicy{DefaultAction: "ask"},
			},
		}
	}
	return []manifest.App{}
}

func detectServices(root string) []manifest.Service {
	if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		tab := "dev"
		return []manifest.Service{
			{
				ID:               "dev",
				Kind:             "project-command",
				Interactive:      true,
				TabName:          &tab,
				WorkingDirectory: root,
				Command:          "pnpm dev",
				DependsOn:        []string{},
				Readiness:        manifest.Readiness{Type: "fixed-delay", DelayMS: 2000},
				StartWhen:        manifest.StartWhen{DelayMS: 0},
				StopPolicy:       manifest.StopPolicy{DefaultAction: "ask"},
			},
		}
	}

	if _, err := os.Stat(filepath.Join(root, "docker-compose.yml")); err == nil {
		return []manifest.Service{
			{
				ID:               "docker-compose",
				Kind:             "project-command",
				Interactive:      false,
				TabName:          nil,
				WorkingDirectory: root,
				Command:          "docker compose up -d",
				DependsOn:        []string{"docker-desktop"},
				Readiness:        manifest.Readiness{Type: "command", Command: "docker compose ps"},
				StartWhen:        manifest.StartWhen{DelayMS: 0},
				StopPolicy: manifest.StopPolicy{
					DefaultAction: "ask",
					Command:       "docker compose down",
				},
			},
		}
	}

	return []manifest.Service{}
}

func failedResource(kind string) state.ResourceState {
	return state.ResourceState{
		Type:       kind,
		Status:     state.StatusFailed,
		LastSeenAt: time.Now().UTC(),
	}
}
