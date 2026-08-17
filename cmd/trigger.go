package cmd

import (
	"context"
	"fmt"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/manager"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	flagTriggerRef   string
	flagTriggerSpace string
)

var triggerCmd = &cobra.Command{
	Use:   "trigger <repo> [workflow]",
	Short: "Dispatch a pipeline or workflow run for a project",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		repo := args[0]
		workflow := "release.yml"
		if len(args) > 1 {
			workflow = args[1]
		}

		spaceID := flagTriggerSpace
		if spaceID == "" {
			if flagSpace != "" && flagSpace != "all" {
				spaceID = flagSpace
			} else if len(cfg.Spaces) > 0 {
				spaceID = cfg.Spaces[0].ID
			}
		}

		mgr := manager.New(cfg)
		prov, err := mgr.GetProvider(spaceID)
		if err != nil {
			return err
		}

		ref := flagTriggerRef
		if ref == "" {
			ref = "main"
		}

		fmt.Printf("🚀 Triggering %s pipeline for %s on ref %s [%s: %s]...\n",
			color.CyanString(workflow),
			color.YellowString(repo),
			color.GreenString(ref),
			prov.Platform(),
			prov.Namespace(),
		)

		if err := prov.TriggerPipeline(context.Background(), repo, ref, nil); err != nil {
			return fmt.Errorf("failed to trigger pipeline: %w", err)
		}

		color.Green("✓ Successfully triggered pipeline! Check status with: vantage runs %s\n", repo)
		return nil
	},
}

func init() {
	triggerCmd.Flags().StringVarP(&flagTriggerRef, "ref", "r", "main", "Git branch, tag, or SHA ref to run on")
	triggerCmd.Flags().StringVarP(&flagTriggerSpace, "space", "s", "", "Space ID to target")
	rootCmd.AddCommand(triggerCmd)
}
