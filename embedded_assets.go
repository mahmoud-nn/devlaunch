package devlaunchassets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed scripts/ps1/*.ps1 skill/**/*
var embeddedFiles embed.FS

func RuntimeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".devlaunch", "runtime"), nil
}

func EnsurePowerShellScript(name string) (string, error) {
	root, err := RuntimeDir()
	if err != nil {
		return "", err
	}
	targetDir := filepath.Join(root, "scripts", "ps1")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	src := filepath.ToSlash(filepath.Join("scripts", "ps1", name))
	dst := filepath.Join(targetDir, name)
	if err := writeEmbeddedFile(src, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func EnsureSkillDir() (string, error) {
	root, err := RuntimeDir()
	if err != nil {
		return "", err
	}
	targetDir := filepath.Join(root, "skill")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	if err := extractTree("skill", targetDir); err != nil {
		return "", err
	}
	return targetDir, nil
}

func extractTree(sourceRoot, targetRoot string) error {
	return fs.WalkDir(embeddedFiles, sourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == sourceRoot {
			return nil
		}

		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return writeEmbeddedFile(path, target)
	})
}

func writeEmbeddedFile(sourcePath, targetPath string) error {
	data, err := embeddedFiles.ReadFile(filepath.ToSlash(sourcePath))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}
