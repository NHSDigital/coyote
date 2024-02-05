package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply <package-file>",
	Short: "Apply a package file to the current project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		return core.Apply(&Context, filename)
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
