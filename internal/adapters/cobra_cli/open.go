package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the current repository in a browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.Open(&Context)
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
