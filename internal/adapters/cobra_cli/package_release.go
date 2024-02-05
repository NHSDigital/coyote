package cobra_cli

import (
	"fmt"

	"github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var packageReleaseCmd = &cobra.Command{
	Use:   "release <package-name> <version-to-release-as>",
	Short: "Build a package and upload the released package file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		url, err := core.PackageRelease(&Context, args[0], args[1])
		if err == nil {
			fmt.Println(url)
		}
		return err
	},
}

func init() {
	packageCmd.AddCommand(packageReleaseCmd)
}
