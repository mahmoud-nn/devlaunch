package windows

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	devlaunchassets "github.com/mahmoud-nn/devlaunch"
)

type CommandResult struct {
	PID int
}

func RunBackground(workingDir, command string) (CommandResult, error) {
	output, err := runScript("run-background.ps1", workingDir, command)
	if err != nil {
		return CommandResult{}, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return CommandResult{}, fmt.Errorf("parse background pid: %w", err)
	}
	return CommandResult{PID: pid}, nil
}

func RunForeground(workingDir, command string) error {
	_, err := runScriptWithStreaming("run-foreground.ps1", workingDir, command)
	return err
}

func RunCheck(workingDir, command string) error {
	_, err := runScript("run-check.ps1", workingDir, command)
	return err
}

func OpenFolder(path string) error {
	_, err := runScript("open-folder.ps1", path)
	return err
}

func LaunchExecutable(path string) (CommandResult, error) {
	output, err := runScript("launch-executable.ps1", path)
	if err != nil {
		return CommandResult{}, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return CommandResult{}, fmt.Errorf("parse executable pid: %w", err)
	}
	return CommandResult{PID: pid}, nil
}

func LaunchInteractiveTab(workingDir, tabName, command string) error {
	_, err := runScript("launch-interactive-tab.ps1", workingDir, tabName, command)
	return err
}

func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func WaitForPort(port int, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		output, err := runScript("wait-for-port.ps1", strconv.Itoa(port), strconv.Itoa(int(delay.Milliseconds())))
		if err == nil && strings.EqualFold(strings.TrimSpace(output), "ready") {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("port %d not ready", port)
}

func ProcessExistsByName(name string) bool {
	output, err := runScript("process-exists.ps1", strings.TrimSpace(strings.TrimSuffix(name, ".exe")))
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(output), "true")
}

func runScript(script string, args ...string) (string, error) {
	cmd, err := buildCommand(script, args...)
	if err != nil {
		return "", err
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", script, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func runScriptWithStreaming(script string, args ...string) (string, error) {
	cmd, err := buildCommand(script, args...)
	if err != nil {
		return "", err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return "", nil
}

func buildCommand(script string, args ...string) (*exec.Cmd, error) {
	path, err := devlaunchassets.EnsurePowerShellScript(script)
	if err != nil {
		return nil, err
	}
	baseArgs := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", path,
	}
	baseArgs = append(baseArgs, args...)
	return exec.Command("powershell", baseArgs...), nil
}
