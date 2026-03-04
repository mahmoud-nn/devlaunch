package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mahmoud-nn/devlaunch/internal/registry"
	"github.com/mahmoud-nn/devlaunch/internal/runtime"
	"github.com/mahmoud-nn/devlaunch/internal/schema"
	"github.com/mahmoud-nn/devlaunch/internal/skill"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "devlaunch",
		Short: "Local project launcher for Windows",
	}
	root.AddCommand(newInitCommand())
	root.AddCommand(newValidateCommand())
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

func newValidateCommand() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate [manifest|state]",
		Short: "Validate devlaunch files against the strict schema",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			target := runtime.ProjectTarget{Root: root}
			kind := ""
			if len(args) == 1 {
				kind = args[0]
			}
			reports, err := runtime.ValidateProject(target)
			if err != nil {
				var validationErr schema.ValidationError
				if errors.As(err, &validationErr) {
					printValidationReport(cmd, validationErr.Report)
					return err
				}
				if len(reports) > 0 {
					for _, report := range reports {
						printValidationReport(cmd, report)
					}
				}
				return err
			}
			for _, report := range reports {
				if kind != "" && report.DocumentType != kind {
					continue
				}
				printValidationReport(cmd, report)
			}
			if kind == "state" && len(reports) == 1 && reports[0].DocumentType == "manifest" {
				fmt.Fprintln(cmd.OutOrStdout(), "State file not found. This is allowed.")
			}
			return nil
		},
	}
	return validateCmd
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
			printProjectList(cmd, projects)
			return nil
		},
	})

	project.AddCommand(&cobra.Command{
		Use:   "start [project-id]",
		Short: "Start a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveProjectTarget(args)
			if err != nil {
				return err
			}
			result, err := runtime.Start(target, runtime.ExecutionOptions{Interactive: true})
			if err != nil {
				return err
			}
			printProjectStatus(cmd, result)
			return nil
		},
	})

	project.AddCommand(&cobra.Command{
		Use:   "stop [project-id]",
		Short: "Stop a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveProjectTarget(args)
			if err != nil {
				return err
			}
			result, err := runtime.Stop(target, runtime.ExecutionOptions{Interactive: true})
			if err != nil {
				return err
			}
			printProjectStatus(cmd, result)
			return nil
		},
	})

	project.AddCommand(&cobra.Command{
		Use:   "status [project-id]",
		Short: "Show project status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveProjectTarget(args)
			if err != nil {
				return err
			}
			result, err := runtime.Status(target)
			if err != nil {
				return err
			}
			printProjectStatus(cmd, result)
			return nil
		},
	})

	project.AddCommand(&cobra.Command{
		Use:   "open [project-id]",
		Short: "Open the project folder",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveProjectTarget(args)
			if err != nil {
				return err
			}
			return runtime.OpenProjectFolder(target)
		},
	})

	return project
}

func resolveProjectTarget(args []string) (runtime.ProjectTarget, error) {
	id := ""
	if len(args) == 1 {
		id = args[0]
	}
	return runtime.ResolveTarget(id)
}

func printValidationReport(cmd *cobra.Command, report schema.ValidationReport) {
	out := cmd.OutOrStdout()
	if report.Valid {
		fmt.Fprintf(out, "%s validation OK\n", strings.Title(report.DocumentType))
		fmt.Fprintf(out, "  version: %d\n", report.Version)
		fmt.Fprintf(out, "  schema: %s\n", report.SchemaRef)
		return
	}
	fmt.Fprintf(out, "%s validation failed\n", strings.Title(report.DocumentType))
	fmt.Fprintf(out, "  version: %d\n", report.Version)
	fmt.Fprintf(out, "  schema: %s\n", report.SchemaRef)
	fmt.Fprintln(out, "  issues:")
	for _, issue := range report.Issues {
		fmt.Fprintf(out, "  - %s: %s\n", issue.Path, issue.Message)
	}
}

func printProjectList(cmd *cobra.Command, projects []registry.ProjectRecord) {
	out := cmd.OutOrStdout()
	if len(projects) == 0 {
		fmt.Fprintln(out, "No registered projects.")
		return
	}
	fmt.Fprintln(out, "Registered projects")
	for _, project := range projects {
		fmt.Fprintf(out, "- %s\n", project.Name)
		fmt.Fprintf(out, "  id: %s\n", project.ID)
		fmt.Fprintf(out, "  path: %s\n", project.RootPath)
		fmt.Fprintf(out, "  status: %s\n", project.LastKnownStatus)
		fmt.Fprintf(out, "  last seen: %s\n", formatTime(project.LastSeenAt))
	}
}

func printProjectStatus(cmd *cobra.Command, status runtime.ProjectStatus) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Project %s\n", status.Name)
	fmt.Fprintf(out, "  id: %s\n", status.ID)
	fmt.Fprintf(out, "  path: %s\n", status.RootPath)
	fmt.Fprintf(out, "  status: %s\n", status.Status)
	fmt.Fprintf(out, "  last start: %s\n", formatOptionalTime(status.LastStartAt))
	fmt.Fprintf(out, "  last stop: %s\n", formatOptionalTime(status.LastStopAt))
	if len(status.Warnings) > 0 {
		fmt.Fprintln(out, "  warnings:")
		for _, warning := range status.Warnings {
			fmt.Fprintf(out, "  - %s\n", warning)
		}
	}
	if len(status.Resources) == 0 {
		fmt.Fprintln(out, "  resources: none")
		return
	}
	fmt.Fprintln(out, "  resources:")
	for _, resource := range status.Resources {
		managed := "manual"
		if resource.Managed {
			managed = "managed"
		}
		line := fmt.Sprintf("  - %s [%s] %s (%s)", resource.ID, resource.Type, resource.Status, managed)
		if resource.Diverged {
			line += " (state mismatch)"
		}
		fmt.Fprintln(out, line)
		if resource.ObservedBy != "" {
			fmt.Fprintf(out, "    observed by: %s\n", resource.ObservedBy)
		}
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "never"
	}
	return formatTime(*value)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Local().Format("2006-01-02 15:04:05")
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
