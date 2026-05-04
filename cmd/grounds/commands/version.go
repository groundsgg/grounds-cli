package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/version"
)

var versionCheckHTTPClient = &http.Client{Timeout: 5 * time.Second}

func NewVersionCommand() *cobra.Command {
	var check bool
	var releaseAPIURL string

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"grounds version %s\n  commit: %s\n  built:  %s\n",
				version.Version, version.Commit, version.BuildAt); err != nil {
				return err
			}
			if !check {
				return nil
			}

			report, err := version.CheckLatest(context.Background(), version.CheckOptions{
				Current:    version.Version,
				APIBaseURL: releaseAPIURL,
				HTTPClient: versionCheckHTTPClient,
			})
			if err != nil {
				return err
			}

			status := "up to date"
			if !report.Comparable {
				status = "local build"
			} else if report.UpdateAvailable {
				status = "update available"
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"  latest: %s\n  status: %s\n",
				report.Latest, status); err != nil {
				return err
			}

			if report.UpdateAvailable {
				printUpdateHint(cmd, report)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "check whether a newer release is available")
	cmd.Flags().StringVar(&releaseAPIURL, "release-api-url", version.DefaultReleaseAPIBaseURL, "GitHub API base URL for release checks")
	_ = cmd.Flags().MarkHidden("release-api-url")
	return cmd
}

func printUpdateHint(cmd *cobra.Command, report version.CheckReport) {
	executable, err := os.Executable()
	if err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
	}

	install := version.DetectInstallMethod(executable, os.Getenv("HOME"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  install method: %s\n", install.Method)
	if install.UpdateCommand != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  update command: `%s`\n", install.UpdateCommand)
		return
	}
	if report.ReleaseURL != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  release: %s\n", report.ReleaseURL)
	}
}
