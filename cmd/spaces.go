package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/models"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var spacesCmd = &cobra.Command{
	Use:   "spaces",
	Short: "List all configured GitHub and GitLab spaces/namespaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		color.New(color.Bold, color.FgHiWhite).Println("CONFIGURED VANTAGE SPACES:")
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SPACE ID\tNAME\tPLATFORM\tNAMESPACE / GROUP\tBASE URL\tAUTH TOKEN")
		fmt.Fprintln(w, "--------\t----\t--------\t-----------------\t--------\t----------")

		for _, s := range cfg.Spaces {
			tokenResolved := config.ResolveToken(s)
			tokenStatus := color.GreenString("configured")
			if tokenResolved == "" {
				tokenStatus = color.YellowString("missing token")
			}

			platStr := color.MagentaString("GitHub")
			if s.Platform == models.PlatformGitLab {
				platStr = color.YellowString("GitLab")
			}

			baseURL := s.BaseURL
			if baseURL == "" {
				if s.Platform == models.PlatformGitLab {
					baseURL = "https://gitlab.com"
				} else {
					baseURL = "https://api.github.com"
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				s.ID,
				s.Name,
				platStr,
				s.Namespace,
				baseURL,
				tokenStatus,
			)
		}

		w.Flush()
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(spacesCmd)
}
