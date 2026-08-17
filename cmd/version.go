package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "0.1.0"
	CommitSHA = "dev"
	BuildDate = "2026-08-17"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Vantage version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vantage version %s (%s) %s\n", Version, CommitSHA, BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
