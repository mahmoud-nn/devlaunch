package devlaunchassets

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed scripts/ps1/*.ps1 skills/devlaunch/references/*.json
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

func ReadEmbeddedFile(path string) ([]byte, error) {
	return embeddedFiles.ReadFile(filepath.ToSlash(path))
}
