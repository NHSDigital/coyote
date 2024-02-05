package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var packageInitCmd = &cobra.Command{
	Use:   "init <package-name>",
	Short: "Create the local file tree for a new package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.PackageInit(&Context, args[0])
	},
}

func init() {
	packageCmd.AddCommand(packageInitCmd)
}
