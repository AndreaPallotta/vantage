package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/manager"
	"github.com/AndreaPallotta/vantage/internal/models"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var runsCmd = &cobra.Command{
	Use:   "runs [repo]",
	Short: "Stream recent CI/CD workflow and pipeline runs across spaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		spaceID := flagSpace
		if spaceID == "" {
			spaceID = cfg.ActiveSpace
		}

		mgr := manager.New(cfg)

		var runs []models.Run

		if len(args) > 0 {
			repo := args[0]
			fmt.Printf("Fetching runs for %s...\n\n", repo)
			if spaceID != "all" {
				if prov, err := mgr.GetProvider(spaceID); err == nil {
					runs, _ = prov.ListPipelines(context.Background(), repo, 25)
				}
			}
		} else {
			fmt.Printf("Fetching space pipeline runs for: %s...\n\n", color.CyanString(spaceID))
			overview, err := mgr.GetOverview(context.Background(), spaceID, cfg.IncludeForks, cfg.IncludeArchived)
			if err != nil {
				return err
			}
			runs = overview.RecentRuns
		}

		if len(runs) == 0 {
			fmt.Println("No recent pipeline runs found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PLATFORM\tPROJECT\tPIPELINE\tBRANCH\tSTATUS\tDURATION\tCOMMIT MESSAGE\tACTOR")
		fmt.Fprintln(w, "--------\t-------\t--------\t------\t------\t--------\t--------------\t-----")

		for _, r := range runs {
			platStr := color.MagentaString("GitHub")
			if r.Platform == models.PlatformGitLab {
				platStr = color.YellowString("GitLab")
			}

			statusStr := r.Status
			if r.Status == "completed" {
				if r.Conclusion == "success" {
					statusStr = color.GreenString("success")
				} else if r.Conclusion == "failure" || r.Conclusion == "failed" {
					statusStr = color.RedString("failure")
				} else {
					statusStr = r.Conclusion
				}
			} else {
				statusStr = color.YellowString(r.Status)
			}

			durStr := fmt.Sprintf("%ds", r.DurationSec)
			if r.DurationSec >= 60 {
				durStr = fmt.Sprintf("%dm %ds", r.DurationSec/60, r.DurationSec%60)
			}

			msg := r.CommitMsg
			if len(msg) > 28 {
				msg = msg[:25] + "..."
			}
			msg = strings.ReplaceAll(msg, "\n", " ")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				platStr,
				r.RepoName,
				r.Name,
				r.HeadBranch,
				statusStr,
				durStr,
				msg,
				r.Actor,
			)
		}

		w.Flush()
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runsCmd)
}
