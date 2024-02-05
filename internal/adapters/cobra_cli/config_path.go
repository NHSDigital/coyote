package cobra_cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the path to the config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(Context.Config.GetPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configPathCmd)
}
