package cobra_cli

import (
	"github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var packageDeleteCmd = &cobra.Command{
	Use:   "delete <package-name>",
	Short: "Delete a package from github",
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.PackageDelete(&Context, args[0])
	},
}

func init() {
	packageCmd.AddCommand(packageDeleteCmd)
}
