package cmd

import (
	"context"
	"fmt"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/github"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	flagTriggerRef string
)

var triggerCmd = &cobra.Command{
	Use:   "trigger <repo> <workflow>",
	Short: "Dispatch a GitHub Actions workflow run for a repository",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		repo := args[0]
		workflow := args[1]

		client := github.NewClient(cfg.Token)
		owner := cfg.Space
		if owner == "" {
			owner = "AndreaPallotta"
		}

		ref := flagTriggerRef
		if ref == "" {
			ref = "main"
		}

		fmt.Printf("🚀 Dispatching workflow %s for %s/%s on ref %s...\n",
			color.CyanString(workflow),
			color.YellowString(owner),
			color.YellowString(repo),
			color.GreenString(ref),
		)

		if err := client.DispatchWorkflow(context.Background(), owner, repo, workflow, ref, nil); err != nil {
			return fmt.Errorf("failed to dispatch workflow: %w", err)
		}

		color.Green("✓ Successfully dispatched workflow! Check status with: vantage runs %s\n", repo)
		return nil
	},
}

func init() {
	triggerCmd.Flags().StringVarP(&flagTriggerRef, "ref", "r", "main", "Git branch, tag, or SHA ref to run on")
	rootCmd.AddCommand(triggerCmd)
}
