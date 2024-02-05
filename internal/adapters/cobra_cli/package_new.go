package cobra_cli

import (
	"github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var packageNewCmd = &cobra.Command{
	Use:   "new <package-name>",
	Short: "Make and upload a new package repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.PackageNew(&Context, args[0])
	},
}

func init() {
	packageCmd.AddCommand(packageNewCmd)
}
