package cmd

import (
	"fmt"
	"strings"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/models"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	flagAddID        string
	flagAddName      string
	flagAddPlatform  string
	flagAddURL       string
	flagAddNamespace string
	flagAddToken     string
	flagAddSubgroups bool
)

var addSpaceCmd = &cobra.Command{
	Use:   "add-space",
	Short: "Add a new GitHub or GitLab namespace/group to Vantage",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagAddID == "" || flagAddNamespace == "" {
			return fmt.Errorf("--id and --namespace are required")
		}

		plat := models.PlatformGitHub
		if strings.ToLower(flagAddPlatform) == "gitlab" {
			plat = models.PlatformGitLab
		}

		name := flagAddName
		if name == "" {
			name = fmt.Sprintf("%s (%s)", strings.ToUpper(string(plat)), flagAddNamespace)
		}

		baseURL := flagAddURL
		if baseURL == "" {
			if plat == models.PlatformGitLab {
				baseURL = "https://gitlab.com/api/v4"
			} else {
				baseURL = "https://api.github.com"
			}
		}

		newSpace := models.SpaceConfig{
			ID:               flagAddID,
			Name:             name,
			Platform:         plat,
			BaseURL:          baseURL,
			Namespace:        flagAddNamespace,
			Token:            flagAddToken,
			IncludeSubgroups: flagAddSubgroups,
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Replace if exists, else append
		replaced := false
		for i, s := range cfg.Spaces {
			if s.ID == flagAddID {
				cfg.Spaces[i] = newSpace
				replaced = true
				break
			}
		}
		if !replaced {
			cfg.Spaces = append(cfg.Spaces, newSpace)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		color.Green("Successfully configured space %s [%s: %s].", flagAddID, plat, flagAddNamespace)
		fmt.Printf("View dashboard with: vantage --space %s\n", flagAddID)
		return nil
	},
}

func init() {
	addSpaceCmd.Flags().StringVar(&flagAddID, "id", "", "Unique space identifier (e.g. gitlab-work)")
	addSpaceCmd.Flags().StringVar(&flagAddName, "name", "", "Display name (e.g. 'GitLab Core Team')")
	addSpaceCmd.Flags().StringVar(&flagAddPlatform, "platform", "github", "Platform type: github or gitlab")
	addSpaceCmd.Flags().StringVar(&flagAddURL, "url", "", "Base API URL (default: https://api.github.com or https://gitlab.com/api/v4)")
	addSpaceCmd.Flags().StringVar(&flagAddNamespace, "namespace", "", "GitHub user/org or GitLab group/namespace")
	addSpaceCmd.Flags().StringVar(&flagAddToken, "token", "", "Access token / API key")
	addSpaceCmd.Flags().BoolVar(&flagAddSubgroups, "subgroups", true, "Include nested GitLab subgroups")

	rootCmd.AddCommand(addSpaceCmd)
}
