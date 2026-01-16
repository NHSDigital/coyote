package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var doReinstall bool
var installCmd = &cobra.Command{
	Use:   "install <package-name>[@<version>]",
	Short: "Install a package to the current project",
	Long: `Install a package and its dependencies to the current project.

If the index contains multiple versions of a package, the most recent version
(using semantic versioning) is installed by default. You can specify a particular
version by appending @<version> to the package name.

Examples:
  coyote install my-package          # Install latest version
  coyote install my-package@v1.0.0   # Install specific version`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgname := args[0]

		return core.Install(&Context, pkgname, doReinstall)
	},
}

func init() {
	installCmd.Flags().BoolVarP(&doReinstall, "reinstall", "r", false, "Reinstall the package if it is already installed")
	rootCmd.AddCommand(installCmd)
}
