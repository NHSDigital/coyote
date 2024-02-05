package cobra_cli

import (
	"fmt"

	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var outdir string
var packageBuildCmd = &cobra.Command{
	Use:   "build <package-name> [version-to-build]",
	Short: "Build a package from the current repository",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]
		version := ""
		if len(args) > 1 {
			version = args[1]
		}
		filename, err := core.PackageBuild(&Context, packageName, outdir, version)
		if err == nil {
			fmt.Println(filename)
		}
		return err
	},
}

func init() {
	packageBuildCmd.Flags().StringVarP(&outdir, "output", "o", ".", "Output directory for the package")
	packageCmd.AddCommand(packageBuildCmd)
}
