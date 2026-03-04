package skill

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"

	devlaunchassets "github.com/mahmoud-nn/devlaunch"
)

func Install() error {
	sourceDir, err := devlaunchassets.EnsureSkillDir()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	targetDir := filepath.Join(home, ".devlaunch", "skills", "devlaunch")
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := copyDir(sourceDir, targetDir); err != nil {
		return err
	}

	cmd := exec.Command("npx", "skills")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return out.Chmod(info.Mode())
	})
}
