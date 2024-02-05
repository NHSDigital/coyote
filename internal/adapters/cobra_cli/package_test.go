package cobra_cli

import (
	"github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var packageTestCmd = &cobra.Command{
	Use:   "test <package-name>",
	Short: "Build the package and attempt to install it in a test project.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.PackageTest(&Context, args[0])
	},
}

func init() {
	packageCmd.AddCommand(packageTestCmd)
}
