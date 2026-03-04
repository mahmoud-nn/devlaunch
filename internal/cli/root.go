package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/mahmoud-nn/devlaunch/internal/runtime"
	"github.com/mahmoud-nn/devlaunch/internal/skill"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "devlaunch",
		Short: "Local project launcher for Windows",
	}
	root.AddCommand(newInitCommand())
	root.AddCommand(newProjectCommand())
	root.AddCommand(newUICommand())
	root.AddCommand(newSkillCommand())
	return root
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize devlaunch in the current repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			doc, _, err := runtime.InitProject(root)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized %s\n", doc.Project.Name)
			return nil
		},
	}
}

func newProjectCommand() *cobra.Command {
	project := &cobra.Command{
		Use:   "project",
		Short: "Manage configured projects",
	}

	project.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List registered projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := runtime.ListProjects()
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(projects)
		},
	})

	project.AddCommand(targetedProjectCommand("start", "Start a project", runtime.Start))
	project.AddCommand(targetedProjectCommand("stop", "Stop a project", runtime.Stop))
	project.AddCommand(targetedProjectCommand("status", "Show project status", runtime.Status))
	project.AddCommand(&cobra.Command{
		Use:   "open [project-id]",
		Short: "Open the project folder",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			target, err := runtime.ResolveTarget(id)
			if err != nil {
				return err
			}
			return runtime.OpenProjectFolder(target)
		},
	})

	return project
}

func targetedProjectCommand(use, short string, run func(runtime.ProjectTarget) (runtime.ProjectStatus, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [project-id]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			target, err := runtime.ResolveTarget(id)
			if err != nil {
				return err
			}
			result, err := run(target)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		},
	}
}

func newUICommand() *cobra.Command {
	ui := &cobra.Command{
		Use:   "ui",
		Short: "Control the local web UI",
	}

	ui.AddCommand(newUIStartCommand())
	ui.AddCommand(newUIStopCommand())
	return ui
}

func newUIStartCommand() *cobra.Command {
	var detach bool
	var port int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the local web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if detach {
				exe, err := os.Executable()
				if err != nil {
					return err
				}
				child := exec.Command(exe, "ui", "start", fmt.Sprintf("--port=%d", port))
				child.Stdout = os.Stdout
				child.Stderr = os.Stderr
				return child.Start()
			}

			return serveUI(cmd, port)
		},
	}

	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Start the UI in the background")
	cmd.Flags().IntVar(&port, "port", runtime.DefaultUIPort, "Port for the local web UI")
	return cmd
}

func newUIStopCommand() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the local web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return callControlEndpoint(port, "/__control/stop")
		},
	}

	cmd.Flags().IntVar(&port, "port", runtime.DefaultUIPort, "Port for the local web UI")
	return cmd
}

func serveUI(cmd *cobra.Command, port int) error {
	server := NewUIServer(runtime.DefaultUIHost, port)
	fmt.Fprintf(cmd.OutOrStdout(), "Web UI listening on http://%s:%d\n", runtime.DefaultUIHost, port)
	return server.ListenAndServe()
}

func newSkillCommand() *cobra.Command {
	skillCmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the embedded skill",
	}

	skillCmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install the local skill and proxy npx skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return skill.Install()
		},
	})

	return skillCmd
}

func callControlEndpoint(port int, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s:%d%s", runtime.DefaultUIHost, port, path), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ui control request failed: %s", resp.Status)
	}
	return nil
}
