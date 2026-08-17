package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/github"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var runsCmd = &cobra.Command{
	Use:   "runs [repo]",
	Short: "Stream recent CI/CD workflow runs across repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		client := github.NewClient(cfg.Token)
		owner := cfg.Space
		if owner == "" {
			owner = "AndreaPallotta"
		}

		var runs []github.Run

		if len(args) > 0 {
			repo := args[0]
			fmt.Printf("🔍 Fetching runs for %s/%s...\n\n", owner, repo)
			rList, err := client.ListWorkflowRuns(context.Background(), owner, repo, 25)
			if err != nil {
				return err
			}
			runs = rList
		} else {
			fmt.Printf("🔍 Fetching recent space runs for %s...\n\n", owner)
			overview, err := client.GetSpaceOverview(context.Background(), owner, cfg.IncludeForks, cfg.IncludeArchived)
			if err != nil {
				return err
			}
			runs = overview.RecentRuns
		}

		if len(runs) == 0 {
			fmt.Println("No recent workflow runs found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tWORKFLOW\tBRANCH\tSTATUS\tDURATION\tCOMMIT MESSAGE\tAUTHOR")
		fmt.Fprintln(w, "----------\t--------\t------\t------\t--------\t--------------\t------")

		for _, r := range runs {
			statusStr := r.Status
			if r.Status == "completed" {
				if r.Conclusion == "success" {
					statusStr = color.GreenString("✓ success")
				} else if r.Conclusion == "failure" {
					statusStr = color.RedString("✗ failure")
				} else {
					statusStr = r.Conclusion
				}
			} else {
				statusStr = color.YellowString("● " + r.Status)
			}

			durStr := fmt.Sprintf("%ds", r.DurationSec)
			if r.DurationSec >= 60 {
				durStr = fmt.Sprintf("%dm %ds", r.DurationSec/60, r.DurationSec%60)
			}

			msg := r.CommitMsg
			if len(msg) > 30 {
				msg = msg[:27] + "..."
			}
			msg = strings.ReplaceAll(msg, "\n", " ")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
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
