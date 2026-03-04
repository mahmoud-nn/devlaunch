package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mahmoud-nn/devlaunch/internal/manifest"
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

func Load(root, projectName string) State {
	data, err := os.ReadFile(manifest.StatePath(root))
	if err != nil {
		return Default(projectName)
	}

	var doc State
	if err := json.Unmarshal(data, &doc); err != nil {
		return Default(projectName)
	}
	if doc.Version == 0 {
		doc.Version = 1
	}
	if doc.ProjectName == "" {
		doc.ProjectName = projectName
	}
	if doc.Resources == nil {
		doc.Resources = map[string]ResourceState{}
	}
	return doc
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
	return os.WriteFile(manifest.StatePath(root), append(data, '\n'), 0o644)
}
