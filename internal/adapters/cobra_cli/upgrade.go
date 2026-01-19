package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [package-name...]",
	Short: "Upgrade installed packages to the latest version",
	Long: `Upgrade installed packages to the latest version available in the index.

If one or more package names are specified, only those packages are upgraded.
If no package names are specified, all installed packages are upgraded.

If a specified package is not installed, this command will fail. If the
installed version is already the latest, no action is taken. The upgrade
will re-run the on-install script if one exists.

Packages that are installed but not present in the index are skipped when
upgrading all packages.

Examples:
  coyote upgrade                       # Upgrade all installed packages
  coyote upgrade my-package            # Upgrade only my-package
  coyote upgrade pkg-a pkg-b pkg-c     # Upgrade multiple specific packages`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.Upgrade(&Context, args)
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
