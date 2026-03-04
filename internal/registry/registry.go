package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ProjectStatus string

const (
	ProjectStatusRunning ProjectStatus = "running"
	ProjectStatusStopped ProjectStatus = "stopped"
	ProjectStatusUnknown ProjectStatus = "unknown"
)

type ProjectRecord struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	RootPath        string        `json:"rootPath"`
	ManifestPath    string        `json:"manifestPath"`
	LastSeenAt      time.Time     `json:"lastSeenAt"`
	LastKnownStatus ProjectStatus `json:"lastKnownStatus"`
}

type Registry struct {
	Version  int             `json:"version"`
	Projects []ProjectRecord `json:"projects"`
}

func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".devlaunch"), nil
}

func filePath() (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "registry", "projects.json"), nil
}

func Load() (Registry, error) {
	path, err := filePath()
	if err != nil {
		return Registry{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Registry{Version: 1, Projects: []ProjectRecord{}}, nil
		}
		return Registry{}, err
	}
	var doc Registry
	if err := json.Unmarshal(data, &doc); err != nil {
		return Registry{Version: 1, Projects: []ProjectRecord{}}, nil
	}
	if doc.Version == 0 {
		doc.Version = 1
	}
	return doc, nil
}

func Save(doc Registry) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sort.Slice(doc.Projects, func(i, j int) bool {
		return strings.ToLower(doc.Projects[i].Name) < strings.ToLower(doc.Projects[j].Name)
	})
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func Upsert(doc Registry, record ProjectRecord) Registry {
	for i := range doc.Projects {
		if samePath(doc.Projects[i].RootPath, record.RootPath) {
			doc.Projects[i] = record
			return doc
		}
	}
	doc.Projects = append(doc.Projects, record)
	return doc
}

func FindByID(doc Registry, id string) (ProjectRecord, bool) {
	for _, project := range doc.Projects {
		if strings.EqualFold(project.ID, id) {
			return project, true
		}
	}
	return ProjectRecord{}, false
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
