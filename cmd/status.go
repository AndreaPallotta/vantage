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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print a CLI fleet summary table across all repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if flagSpace != "" {
			cfg.Space = flagSpace
		}
		if flagToken != "" {
			cfg.Token = flagToken
		}

		client := github.NewClient(cfg.Token)

		owner := cfg.Space
		if owner == "" {
			if user, err := client.GetAuthenticatedUser(context.Background()); err == nil && user.Login != "" {
				owner = user.Login
			} else {
				owner = "AndreaPallotta"
			}
		}

		fmt.Printf("🔍 Scanning space: %s...\n\n", color.CyanString(owner))

		overview, err := client.GetSpaceOverview(context.Background(), owner, cfg.IncludeForks, cfg.IncludeArchived)
		if err != nil {
			return err
		}

		// Print Summary Banner
		color.New(color.Bold, color.FgHiWhite).Printf("VANTAGE FLEET SUMMARY: %s\n", owner)
		fmt.Printf("Repositories: %s  |  Stars: %s  |  Active CI: %s  |  Failed CI: %s  |  Success Rate: %s\n\n",
			color.CyanString("%d", overview.TotalRepos),
			color.YellowString("%d", overview.TotalStars),
			color.YellowString("%d", overview.ActivePipelines),
			color.RedString("%d", overview.FailedPipelines),
			color.GreenString("%.0f%%", overview.SuccessRate),
		)

		// Render Table with text/tabwriter
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tLANGUAGE\tSTARS\tBRANCH\tLATEST COMMIT\tRELEASE TAG\tPIPELINE STATUS")
		fmt.Fprintln(w, "----------\t--------\t-----\t------\t-------------\t-----------\t---------------")

		for _, r := range overview.Repositories {
			commitStr := "-"
			if r.LatestCommit != nil {
				shortMsg := r.LatestCommit.Message
				if len(shortMsg) > 28 {
					shortMsg = shortMsg[:25] + "..."
				}
				shortMsg = strings.ReplaceAll(shortMsg, "\n", " ")
				commitStr = fmt.Sprintf("[%s] %s", r.LatestCommit.ShortSHA, shortMsg)
			}

			tagStr := "-"
			if r.LatestRelease != nil {
				tagStr = r.LatestRelease.TagName
			}

			ciStr := color.HiBlackString("no runs")
			if len(r.WorkflowRuns) > 0 {
				lastRun := r.WorkflowRuns[0]
				if lastRun.Status == "in_progress" {
					ciStr = color.YellowString("● running")
				} else if lastRun.Status == "queued" {
					ciStr = color.YellowString("● queued")
				} else if lastRun.Conclusion == "success" {
					ciStr = color.GreenString("✓ passing")
				} else if lastRun.Conclusion == "failure" {
					ciStr = color.RedString("✗ failed")
				} else {
					ciStr = lastRun.Conclusion
				}
			}

			lang := r.Language
			if lang == "" {
				lang = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
				r.Name,
				lang,
				r.Stars,
				r.DefaultBranch,
				commitStr,
				tagStr,
				ciStr,
			)
		}

		w.Flush()
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
