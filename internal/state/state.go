package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mahmoud-nn/devlaunch/internal/manifest"
	"github.com/mahmoud-nn/devlaunch/internal/schema"
)

type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusUnknown Status = "unknown"
	StatusFailed  Status = "failed"
)

type ResourceState struct {
	Type             string    `json:"type"`
	Status           Status    `json:"status"`
	StartedByRuntime bool      `json:"startedByRuntime"`
	LastKnownPID     int       `json:"lastKnownPid,omitempty"`
	TerminalTabName  string    `json:"terminalTabName,omitempty"`
	LastSeenAt       time.Time `json:"lastSeenAt,omitempty"`
}

type State struct {
	Version     int                      `json:"version"`
	ProjectName string                   `json:"projectName"`
	LastStartAt *time.Time               `json:"lastStartAt"`
	LastStopAt  *time.Time               `json:"lastStopAt"`
	Resources   map[string]ResourceState `json:"resources"`
}

func Default(projectName string) State {
	return State{
		Version:     1,
		ProjectName: projectName,
		Resources:   map[string]ResourceState{},
	}
}

func Load(root, projectName string) (State, []string, error) {
	path := manifest.StatePath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(projectName), nil, nil
		}
		return State{}, nil, err
	}
	if _, err := schema.ValidateStateBytes(data); err != nil {
		warnings := []string{fmt.Sprintf("State file %s is invalid and was ignored: %s", path, err.Error())}
		return Default(projectName), warnings, nil
	}

	var doc State
	if err := json.Unmarshal(data, &doc); err != nil {
		warnings := []string{fmt.Sprintf("State file %s could not be parsed and was ignored: %s", path, err.Error())}
		return Default(projectName), warnings, nil
	}
	if doc.ProjectName == "" {
		doc.ProjectName = projectName
	}
	if doc.Resources == nil {
		doc.Resources = map[string]ResourceState{}
	}
	return doc, nil, nil
}

func Save(root string, doc State) error {
	if doc.Version == 0 {
		doc.Version = 1
	}
	if doc.Resources == nil {
		doc.Resources = map[string]ResourceState{}
	}
	if err := os.MkdirAll(filepath.Join(root, manifest.DirName), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if _, err := schema.ValidateStateBytes(data); err != nil {
		return err
	}
	return os.WriteFile(manifest.StatePath(root), append(data, '\n'), 0o644)
}
