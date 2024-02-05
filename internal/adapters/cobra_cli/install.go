package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var doReinstall bool
var installCmd = &cobra.Command{
	Use:   "install <package-name>",
	Short: "Install a package to the current project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgname := args[0]

		return core.Install(&Context, pkgname, doReinstall)
	},
}

func init() {
	installCmd.Flags().BoolVarP(&doReinstall, "reinstall", "r", false, "Reinstall the package if it is already installed")
	rootCmd.AddCommand(installCmd)
}
