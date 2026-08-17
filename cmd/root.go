package cmd

import (
	"fmt"
	"os"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/manager"
	"github.com/AndreaPallotta/vantage/internal/server"
	"github.com/spf13/cobra"
)

var (
	flagPort            int
	flagSpace           string
	flagToken           string
	flagNoOpen          bool
	flagIncludeForks    bool
	flagIncludeArchived bool
)

var rootCmd = &cobra.Command{
	Use:   "vantage",
	Short: "Centralized Multi-Platform (GitHub & GitLab) Mission Control Dashboard",
	Long: `Vantage is a unified mission control cockpit for monitoring repositories,
commit vitality, release tags, and CI/CD pipelines across all your GitHub and GitLab spaces.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}

		if flagPort != 0 {
			cfg.Port = flagPort
		}
		if flagSpace != "" {
			cfg.ActiveSpace = flagSpace
		}
		if flagNoOpen {
			cfg.AutoOpen = false
		}
		if flagIncludeForks {
			cfg.IncludeForks = true
		}
		if flagIncludeArchived {
			cfg.IncludeArchived = true
		}

		mgr := manager.New(cfg)
		srv := server.New(cfg, mgr)

		return srv.Start()
	},
}

// Execute is the main entry point for the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "vantage: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&flagPort, "port", "p", 8080, "Port to run the dashboard web server on")
	rootCmd.PersistentFlags().StringVarP(&flagSpace, "space", "s", "all", "Space ID to monitor or 'all' for unified fleet")
	rootCmd.PersistentFlags().BoolVar(&flagNoOpen, "no-open", false, "Do not automatically open browser on start")
	rootCmd.PersistentFlags().BoolVar(&flagIncludeForks, "include-forks", false, "Include forked repositories in space overview")
	rootCmd.PersistentFlags().BoolVar(&flagIncludeArchived, "include-archived", false, "Include archived repositories")
}
