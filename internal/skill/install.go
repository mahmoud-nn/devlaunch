package skill

import (
	"os"
	"os/exec"
)

const (
	skillRepoSource    = "mahmoud-nn/devlaunch"
	devlaunchSkillName = "devlaunch"
)

func Install() error {
	cmd := exec.Command(
		"npx",
		"skills",
		"add",
		skillRepoSource,
		"-g",
		"--skill", devlaunchSkillName,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
