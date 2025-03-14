package cobra_cli

import (
	"fmt"

	"github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var packageVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version that would be built",
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := core.PackageVersion(&Context)
		if err == nil {
			fmt.Println(version)
		}
		return err
	},
}

func init() {
	packageCmd.AddCommand(packageVersionCmd)
}
