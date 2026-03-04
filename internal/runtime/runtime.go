package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mahmoud-nn/devlaunch/internal/manifest"
	"github.com/mahmoud-nn/devlaunch/internal/registry"
	"github.com/mahmoud-nn/devlaunch/internal/schema"
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

type ExecutionOptions struct {
	Interactive bool            `json:"interactive"`
	Decisions   map[string]bool `json:"decisions,omitempty"`
}

type ResourceStatus struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Status      state.Status `json:"status"`
	StateStatus state.Status `json:"stateStatus"`
	Managed     bool         `json:"managed"`
	Diverged    bool         `json:"diverged"`
	ObservedBy  string       `json:"observedBy,omitempty"`
}

type ProjectStatus struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	RootPath    string           `json:"rootPath"`
	Status      string           `json:"status"`
	LastStartAt *time.Time       `json:"lastStartAt"`
	LastStopAt  *time.Time       `json:"lastStopAt"`
	Resources   []ResourceStatus `json:"resources"`
	Warnings    []string         `json:"warnings,omitempty"`
}

type loadContext struct {
	doc      manifest.Manifest
	state    state.State
	warnings []string
}

type observedStatus struct {
	status     state.Status
	observedBy string
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
	st, _, err := ensureState(root, doc.Project.Name)
	if err != nil {
		return manifest.Manifest{}, state.State{}, err
	}
	if _, err := Register(root, registry.ProjectStatusStopped); err != nil {
		return manifest.Manifest{}, state.State{}, err
	}
	return doc, st, nil
}

func ValidateProject(target ProjectTarget) ([]schema.ValidationReport, error) {
	root := filepath.Clean(target.Root)
	reports := []schema.ValidationReport{}

	manifestBytes, err := os.ReadFile(manifest.ManifestPath(root))
	if err != nil {
		return nil, err
	}
	manifestReport, err := schema.ValidateManifestBytes(manifestBytes)
	reports = append(reports, manifestReport)
	if err != nil {
		return reports, err
	}

	stateBytes, err := os.ReadFile(manifest.StatePath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return reports, nil
		}
		return reports, err
	}
	stateReport, err := schema.ValidateStateBytes(stateBytes)
	reports = append(reports, stateReport)
	if err != nil {
		return reports, err
	}
	return reports, nil
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
	ctx, err := loadProject(target)
	if err != nil {
		return ProjectStatus{}, err
	}
	current := summarize(ctx.doc, ctx.state, ctx.warnings)
	if _, err := Register(ctx.doc.Project.RootPath, registry.ProjectStatus(current.Status)); err != nil {
		return ProjectStatus{}, err
	}
	return current, nil
}

func Start(target ProjectTarget, options ExecutionOptions) (ProjectStatus, error) {
	ctx, err := loadProject(target)
	if err != nil {
		return ProjectStatus{}, err
	}

	appOrder, serviceOrder, err := dependencyOrder(ctx.doc)
	if err != nil {
		return ProjectStatus{}, err
	}

	for _, app := range appOrder {
		resource := ctx.state.Resources[app.ID]
		if resource.Status == state.StatusRunning {
			continue
		}
		shouldStart, err := resolveAction("start", "app", app.ID, app.StartPolicy, options, &ctx.warnings)
		if err != nil {
			return ProjectStatus{}, err
		}
		if !shouldStart {
			continue
		}
		result, err := startApp(ctx.doc.Project.RootPath, app)
		if err != nil {
			ctx.state.Resources[app.ID] = failedResource("app")
			_ = state.Save(ctx.doc.Project.RootPath, ctx.state)
			return ProjectStatus{}, fmt.Errorf("start app %s: %w", app.ID, err)
		}
		ctx.state.Resources[app.ID] = state.ResourceState{
			Type:             "app",
			Status:           state.StatusRunning,
			StartedByRuntime: true,
			LastKnownPID:     result.PID,
			LastSeenAt:       time.Now().UTC(),
		}
	}

	for _, service := range serviceOrder {
		resource := ctx.state.Resources[service.ID]
		if resource.Status == state.StatusRunning {
			continue
		}
		shouldStart, err := resolveAction("start", "service", service.ID, service.StartPolicy, options, &ctx.warnings)
		if err != nil {
			return ProjectStatus{}, err
		}
		if !shouldStart {
			continue
		}
		result, err := startService(service)
		if err != nil {
			ctx.state.Resources[service.ID] = failedResource("service")
			_ = state.Save(ctx.doc.Project.RootPath, ctx.state)
			return ProjectStatus{}, fmt.Errorf("start service %s: %w", service.ID, err)
		}
		next := state.ResourceState{
			Type:             "service",
			Status:           state.StatusRunning,
			StartedByRuntime: true,
			LastKnownPID:     result.PID,
			LastSeenAt:       time.Now().UTC(),
		}
		if service.TabName != nil {
			next.TerminalTabName = *service.TabName
		}
		ctx.state.Resources[service.ID] = next
	}

	now := time.Now().UTC()
	ctx.state.LastStartAt = &now
	if err := state.Save(ctx.doc.Project.RootPath, ctx.state); err != nil {
		return ProjectStatus{}, err
	}
	result := summarize(ctx.doc, ctx.state, ctx.warnings)
	if _, err := Register(ctx.doc.Project.RootPath, registry.ProjectStatus(result.Status)); err != nil {
		return ProjectStatus{}, err
	}
	return result, nil
}

func Stop(target ProjectTarget, options ExecutionOptions) (ProjectStatus, error) {
	ctx, err := loadProject(target)
	if err != nil {
		return ProjectStatus{}, err
	}

	for i := len(ctx.doc.Services) - 1; i >= 0; i-- {
		service := ctx.doc.Services[i]
		resource := ctx.state.Resources[service.ID]
		if resource.Status == state.StatusStopped {
			continue
		}
		shouldStop, err := resolveAction("stop", "service", service.ID, service.StopPolicy, options, &ctx.warnings)
		if err != nil {
			return ProjectStatus{}, err
		}
		if !shouldStop {
			continue
		}
		if err := stopService(service, resource, &ctx.warnings); err != nil {
			ctx.state.Resources[service.ID] = failedResource("service")
			_ = state.Save(ctx.doc.Project.RootPath, ctx.state)
			return ProjectStatus{}, fmt.Errorf("stop service %s: %w", service.ID, err)
		}
		ctx.state.Resources[service.ID] = state.ResourceState{
			Type:             "service",
			Status:           state.StatusStopped,
			StartedByRuntime: resource.StartedByRuntime,
			TerminalTabName:  resource.TerminalTabName,
			LastSeenAt:       time.Now().UTC(),
		}
	}

	for i := len(ctx.doc.Apps) - 1; i >= 0; i-- {
		app := ctx.doc.Apps[i]
		resource := ctx.state.Resources[app.ID]
		if resource.Status == state.StatusStopped {
			continue
		}
		shouldStop, err := resolveAction("stop", "app", app.ID, app.StopPolicy, options, &ctx.warnings)
		if err != nil {
			return ProjectStatus{}, err
		}
		if !shouldStop {
			continue
		}
		if err := stopApp(app, resource, &ctx.warnings); err != nil {
			ctx.state.Resources[app.ID] = failedResource("app")
			_ = state.Save(ctx.doc.Project.RootPath, ctx.state)
			return ProjectStatus{}, fmt.Errorf("stop app %s: %w", app.ID, err)
		}
		ctx.state.Resources[app.ID] = state.ResourceState{
			Type:             "app",
			Status:           state.StatusStopped,
			StartedByRuntime: resource.StartedByRuntime,
			LastSeenAt:       time.Now().UTC(),
		}
	}

	now := time.Now().UTC()
	ctx.state.LastStopAt = &now
	if err := state.Save(ctx.doc.Project.RootPath, ctx.state); err != nil {
		return ProjectStatus{}, err
	}
	result := summarize(ctx.doc, ctx.state, ctx.warnings)
	if _, err := Register(ctx.doc.Project.RootPath, registry.ProjectStatus(result.Status)); err != nil {
		return ProjectStatus{}, err
	}
	return result, nil
}

func OpenProjectFolder(target ProjectTarget) error {
	if target.Root == "" {
		return errors.New("missing project root")
	}
	return windows.OpenFolder(target.Root)
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

func ensureState(root, projectName string) (state.State, []string, error) {
	path := manifest.StatePath(root)
	if _, err := os.Stat(path); err == nil {
		return state.Load(root, projectName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return state.State{}, nil, err
	}
	st := state.Default(projectName)
	if err := state.Save(root, st); err != nil {
		return state.State{}, nil, err
	}
	return st, nil, nil
}

func loadProject(target ProjectTarget) (loadContext, error) {
	root := filepath.Clean(target.Root)
	doc, err := manifest.Load(root)
	if err != nil {
		return loadContext{}, err
	}
	st, warnings, err := state.Load(root, doc.Project.Name)
	if err != nil {
		return loadContext{}, err
	}
	st, warnings = reconcile(doc, st, warnings)
	if err := state.Save(root, st); err != nil {
		return loadContext{}, err
	}
	return loadContext{doc: doc, state: st, warnings: warnings}, nil
}

func summarize(doc manifest.Manifest, st state.State, warnings []string) ProjectStatus {
	statusValue := "stopped"
	resources := make([]ResourceStatus, 0, len(doc.Apps)+len(doc.Services))

	for _, app := range doc.Apps {
		resource := st.Resources[app.ID]
		observed := observeApp(doc.Project.RootPath, app, resource)
		resources = append(resources, ResourceStatus{
			ID:          app.ID,
			Type:        "app",
			Status:      observed.status,
			StateStatus: resource.Status,
			Managed:     resource.StartedByRuntime,
			Diverged:    observed.status != resource.Status,
			ObservedBy:  observed.observedBy,
		})
		if observed.status == state.StatusRunning {
			statusValue = "running"
		}
	}

	for _, service := range doc.Services {
		resource := st.Resources[service.ID]
		observed := observeService(service, resource)
		resources = append(resources, ResourceStatus{
			ID:          service.ID,
			Type:        "service",
			Status:      observed.status,
			StateStatus: resource.Status,
			Managed:     resource.StartedByRuntime,
			Diverged:    observed.status != resource.Status,
			ObservedBy:  observed.observedBy,
		})
		if observed.status == state.StatusRunning {
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
		Warnings:    warnings,
	}
}

func reconcile(doc manifest.Manifest, st state.State, warnings []string) (state.State, []string) {
	if st.Resources == nil {
		st.Resources = map[string]state.ResourceState{}
	}
	for _, app := range doc.Apps {
		resource := st.Resources[app.ID]
		observed := observeApp(doc.Project.RootPath, app, resource)
		resource.Type = "app"
		resource.Status = observed.status
		resource.LastSeenAt = time.Now().UTC()
		if observed.status != state.StatusRunning {
			resource.LastKnownPID = 0
		}
		st.Resources[app.ID] = resource
	}
	for _, service := range doc.Services {
		resource := st.Resources[service.ID]
		observed := observeService(service, resource)
		resource.Type = "service"
		resource.Status = observed.status
		resource.LastSeenAt = time.Now().UTC()
		if observed.status != state.StatusRunning {
			resource.LastKnownPID = 0
		}
		st.Resources[service.ID] = resource
	}
	return st, warnings
}

func observeApp(root string, app manifest.App, resource state.ResourceState) observedStatus {
	if resource.LastKnownPID > 0 && windows.IsProcessRunning(resource.LastKnownPID) {
		return observedStatus{status: state.StatusRunning, observedBy: "pid"}
	}
	return observeCheckGroup(root, app.Checks.Status)
}

func observeService(service manifest.Service, resource state.ResourceState) observedStatus {
	if resource.LastKnownPID > 0 && windows.IsProcessRunning(resource.LastKnownPID) {
		return observedStatus{status: state.StatusRunning, observedBy: "pid"}
	}
	return observeCheckGroup(service.WorkingDirectory, service.Checks.Status)
}

func observeCheckGroup(workingDir string, group manifest.CheckGroup) observedStatus {
	results := make([]itemResult, 0, len(group.Items))
	for _, item := range group.Items {
		results = append(results, observeCheckItem(workingDir, item))
	}
	return aggregateCheckResults(group.Mode, results)
}

type itemResult struct {
	success    bool
	conclusive bool
	source     string
}

func observeCheckItem(workingDir string, item manifest.CheckItem) itemResult {
	switch item.Type {
	case "command":
		err := runCommandCheck(workingDir, item)
		return itemResult{success: err == nil, conclusive: true, source: "command"}
	case "port":
		err := runPortCheck(item)
		return itemResult{success: err == nil, conclusive: true, source: "port"}
	case "process":
		return itemResult{success: windows.ProcessExistsByName(item.Name), conclusive: true, source: "process"}
	case "http":
		err := runHTTPCheck(item)
		return itemResult{success: err == nil, conclusive: true, source: "http"}
	case "fixed-delay":
		return itemResult{success: false, conclusive: false, source: "fixed-delay"}
	default:
		return itemResult{success: false, conclusive: false, source: item.Type}
	}
}

func aggregateCheckResults(mode string, results []itemResult) observedStatus {
	sources := make([]string, 0, len(results))
	conclusiveFailures := 0
	conclusiveSuccesses := 0
	inconclusive := 0
	for _, result := range results {
		sources = append(sources, result.source)
		if !result.conclusive {
			inconclusive++
			continue
		}
		if result.success {
			conclusiveSuccesses++
		} else {
			conclusiveFailures++
		}
	}
	observedBy := fmt.Sprintf("checks.%s(%s)", mode, strings.Join(sources, ","))
	switch mode {
	case "all":
		if conclusiveFailures > 0 {
			return observedStatus{status: state.StatusStopped, observedBy: observedBy}
		}
		if conclusiveSuccesses == len(results) {
			return observedStatus{status: state.StatusRunning, observedBy: observedBy}
		}
		return observedStatus{status: state.StatusUnknown, observedBy: observedBy}
	case "any":
		if conclusiveSuccesses > 0 {
			return observedStatus{status: state.StatusRunning, observedBy: observedBy}
		}
		if inconclusive > 0 {
			return observedStatus{status: state.StatusUnknown, observedBy: observedBy}
		}
		return observedStatus{status: state.StatusStopped, observedBy: observedBy}
	default:
		return observedStatus{status: state.StatusUnknown, observedBy: "invalid-check-mode"}
	}
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
	var result windows.CommandResult
	var err error
	switch app.Launch.Strategy {
	case "executable":
		result, err = windows.LaunchExecutable(app.Launch.Path)
	case "command":
		result, err = windows.RunBackground(root, app.Launch.Command)
	default:
		return windows.CommandResult{}, fmt.Errorf("unsupported app launch strategy %q", app.Launch.Strategy)
	}
	if err != nil {
		return windows.CommandResult{}, err
	}
	if err := waitForCheckGroup(root, app.Checks.Start); err != nil {
		return windows.CommandResult{}, err
	}
	return result, nil
}

func startService(service manifest.Service) (windows.CommandResult, error) {
	var result windows.CommandResult
	var err error
	if service.Interactive {
		tabName := service.ID
		if service.TabName != nil && *service.TabName != "" {
			tabName = *service.TabName
		}
		result, err = windows.LaunchInteractiveTab(service.WorkingDirectory, tabName, service.Command, service.ID)
	} else {
		result, err = windows.RunBackground(service.WorkingDirectory, service.Command)
	}
	if err != nil {
		return windows.CommandResult{}, err
	}
	if err := waitForCheckGroup(service.WorkingDirectory, service.Checks.Start); err != nil {
		return windows.CommandResult{}, err
	}
	return result, nil
}

func stopApp(app manifest.App, resource state.ResourceState, warnings *[]string) error {
	if app.StopPolicy.Command != "" {
		return windows.RunForeground("", app.StopPolicy.Command)
	}
	if app.Launch.Strategy == "command" && resource.StartedByRuntime && resource.LastKnownPID > 0 {
		if err := windows.KillProcessTree(resource.LastKnownPID); err != nil && windows.IsProcessRunning(resource.LastKnownPID) {
			return err
		}
		return nil
	}
	*warnings = append(*warnings, fmt.Sprintf("App %q has no configured stop command and was left running.", app.ID))
	return nil
}

func stopService(service manifest.Service, resource state.ResourceState, warnings *[]string) error {
	if service.StopPolicy.Command != "" {
		if err := windows.RunForeground(service.WorkingDirectory, service.StopPolicy.Command); err != nil {
			return err
		}
	}
	if service.Interactive && resource.StartedByRuntime && resource.LastKnownPID > 0 {
		if err := windows.KillProcessTree(resource.LastKnownPID); err != nil && windows.IsProcessRunning(resource.LastKnownPID) {
			return err
		}
		return nil
	}
	if service.StopPolicy.Command == "" && !service.Interactive {
		*warnings = append(*warnings, fmt.Sprintf("Service %q has no configured stop command and may still be running.", service.ID))
	}
	return nil
}

func waitForCheckGroup(workingDir string, group manifest.CheckGroup) error {
	switch group.Mode {
	case "all":
		for _, item := range group.Items {
			if err := waitForCheckItem(workingDir, item); err != nil {
				return err
			}
		}
		return nil
	case "any":
		var lastErr error
		for _, item := range group.Items {
			if err := waitForCheckItem(workingDir, item); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		if lastErr == nil {
			lastErr = errors.New("no start checks configured")
		}
		return lastErr
	default:
		return fmt.Errorf("unsupported checks mode %q", group.Mode)
	}
}

func waitForCheckItem(workingDir string, item manifest.CheckItem) error {
	switch item.Type {
	case "command":
		return waitWithRetry(item.Retry, func() error {
			return windows.RunCheck(resolveWorkingDir(workingDir, item.WorkingDirectory), item.Command)
		})
	case "port":
		return runPortCheck(item)
	case "process":
		return waitWithRetry(retryOrDefault(item.Retry), func() error {
			if windows.ProcessExistsByName(item.Name) {
				return nil
			}
			return fmt.Errorf("process %s not ready", item.Name)
		})
	case "http":
		return waitWithRetry(retryOrDefault(item.Retry), func() error { return runHTTPCheck(item) })
	case "fixed-delay":
		time.Sleep(time.Duration(item.DelayMS) * time.Millisecond)
		return nil
	default:
		return fmt.Errorf("unsupported check type %q", item.Type)
	}
}

func runCommandCheck(workingDir string, item manifest.CheckItem) error {
	return windows.RunCheck(resolveWorkingDir(workingDir, item.WorkingDirectory), item.Command)
}

func runPortCheck(item manifest.CheckItem) error {
	retry := retryOrDefault(item.Retry)
	return windows.WaitForPort(item.Port, retry.MaxAttempts, time.Duration(retry.DelayMS)*time.Millisecond)
}

func runHTTPCheck(item manifest.CheckItem) error {
	timeout := 2 * time.Second
	if item.TimeoutMS > 0 {
		timeout = time.Duration(item.TimeoutMS) * time.Millisecond
	}
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(item.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if len(item.ExpectStatus) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		return fmt.Errorf("unexpected http status %d", resp.StatusCode)
	}
	for _, statusCode := range item.ExpectStatus {
		if resp.StatusCode == statusCode {
			return nil
		}
	}
	return fmt.Errorf("unexpected http status %d", resp.StatusCode)
}

func waitWithRetry(retry *manifest.Retry, fn func() error) error {
	actual := retry
	if actual == nil {
		actual = &manifest.Retry{MaxAttempts: 1, DelayMS: 0}
	}
	attempts := actual.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	delay := time.Duration(actual.DelayMS) * time.Millisecond
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < attempts-1 && delay > 0 {
			time.Sleep(delay)
		}
	}
	return lastErr
}

func resolveAction(action, kind, id string, policy manifest.Policy, options ExecutionOptions, warnings *[]string) (bool, error) {
	decision, ok := normalizedDecision(policy.DefaultAction)
	if ok {
		return decision, nil
	}
	if policy.DefaultAction != "ask" {
		return false, fmt.Errorf("unsupported %s policy for %s %q: %s", action, kind, id, policy.DefaultAction)
	}
	if explicit, ok := options.Decisions[id]; ok {
		return explicit, nil
	}
	if !options.Interactive || !isInteractiveSession() {
		*warnings = append(*warnings, fmt.Sprintf("Skipped %s for %s %q because confirmation was required.", action, kind, id))
		return false, nil
	}
	return promptYesNo(fmt.Sprintf("%s %s %q now? [y/N]: ", strings.Title(action), kind, id))
}

func normalizedDecision(action string) (bool, bool) {
	switch action {
	case "", "always":
		return true, true
	case "never":
		return false, true
	default:
		return false, false
	}
}

func promptYesNo(message string) (bool, error) {
	fmt.Fprint(os.Stdout, message)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) && !strings.Contains(err.Error(), "EOF") {
		return false, err
	}
	value := strings.TrimSpace(strings.ToLower(line))
	return value == "y" || value == "yes", nil
}

func isInteractiveSession() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func detectApps() []manifest.App {
	dockerPath := `C:\Program Files\Docker\Docker\Docker Desktop.exe`
	if _, err := os.Stat(dockerPath); err == nil {
		return []manifest.App{
			{
				ID:        "docker-desktop",
				Kind:      "desktop-app",
				DependsOn: []string{},
				Launch: manifest.Launch{
					Strategy: "executable",
					Path:     dockerPath,
				},
				StartPolicy: manifest.Policy{DefaultAction: "always"},
				StopPolicy:  manifest.Policy{DefaultAction: "ask"},
				Checks: manifest.Checks{
					Start: manifest.CheckGroup{
						Mode: "all",
						Items: []manifest.CheckItem{
							{Type: "command", Command: "docker info", Retry: &manifest.Retry{MaxAttempts: 60, DelayMS: 2000}},
						},
					},
					Status: manifest.CheckGroup{
						Mode: "all",
						Items: []manifest.CheckItem{
							{Type: "command", Command: "docker info", Retry: &manifest.Retry{MaxAttempts: 1, DelayMS: 0}},
						},
					},
				},
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
				StartPolicy:      manifest.Policy{DefaultAction: "always"},
				StopPolicy:       manifest.Policy{DefaultAction: "always"},
				Checks: manifest.Checks{
					Start: manifest.CheckGroup{
						Mode: "all",
						Items: []manifest.CheckItem{
							{Type: "fixed-delay", DelayMS: 2000},
						},
					},
					Status: manifest.CheckGroup{
						Mode: "any",
						Items: []manifest.CheckItem{
							{Type: "process", Name: "node"},
						},
					},
				},
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
				StartPolicy:      manifest.Policy{DefaultAction: "always"},
				StopPolicy:       manifest.Policy{DefaultAction: "always", Command: "docker compose down"},
				Checks: manifest.Checks{
					Start: manifest.CheckGroup{
						Mode: "all",
						Items: []manifest.CheckItem{
							{Type: "command", Command: "docker compose ps", Retry: &manifest.Retry{MaxAttempts: 10, DelayMS: 1000}},
						},
					},
					Status: manifest.CheckGroup{
						Mode: "all",
						Items: []manifest.CheckItem{
							{Type: "command", Command: "docker compose ps", Retry: &manifest.Retry{MaxAttempts: 1, DelayMS: 0}},
						},
					},
				},
			},
		}
	}
	return []manifest.Service{}
}

func resolveWorkingDir(defaultDir, override string) string {
	if override != "" {
		return override
	}
	return defaultDir
}

func failedResource(kind string) state.ResourceState {
	return state.ResourceState{
		Type:       kind,
		Status:     state.StatusFailed,
		LastSeenAt: time.Now().UTC(),
	}
}

func retryOrDefault(retry *manifest.Retry) *manifest.Retry {
	if retry != nil {
		return retry
	}
	return &manifest.Retry{MaxAttempts: 1, DelayMS: 0}
}
