package cobra_cli

import (
	"fmt"

	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var outdir string
var buildAll bool

var packageBuildCmd = &cobra.Command{
	Use:   "build [<package-name>] [version-to-build]",
	Short: "Build a package from the current repository",
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if buildAll {
			if len(args) > 1 {
				return fmt.Errorf("cannot specify a package name with --all")
			}
			version := ""
			if len(args) == 1 {
				version = args[0]
			}
			filenames, err := core.PackageBuildAll(&Context, outdir, version)
			if err == nil {
				for _, filename := range filenames {
					fmt.Println(filename)
				}
			}
			return err
		}

		if len(args) < 1 {
			return fmt.Errorf("package name is required (or use --all)")
		}
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
	packageBuildCmd.Flags().BoolVarP(&buildAll, "all", "a", false, "Build all packages in .cypkg/")
	packageCmd.AddCommand(packageBuildCmd)
}
