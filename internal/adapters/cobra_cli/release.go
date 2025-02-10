package cobra_cli

import (
	"os"

	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var version string
var description string
var marker string
var filenames []string
var remoteName string
var releaseCmd = &cobra.Command{
	Use:   "release <version> <filename>...",
	Short: "Release to the current repository",
	Long:  "Release the listed files to the current repository. If HEAD does not yet have the given version tag, it will be created.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		filenames := args[1:]

		if description != "" {
			if description[0] == '@' {
				readDescription, err := os.ReadFile(description[1:])
				if err != nil {
					return err
				}
				description = string(readDescription)
			}
		}

		urls, err := core.Release(&Context, version, description, remoteName, filenames)
		if err != nil {
			return err
		}
		for _, url := range urls {
			cmd.Println(url)
		}
		return nil
	},
}

func init() {
	releaseCmd.Flags().StringVarP(&description, "description", "d", "", "A description for the release.  Can be @file to read from a file.")
	releaseCmd.Flags().StringVarP(&remoteName, "remote", "r", "origin", "The remote to push the release to.")
	rootCmd.AddCommand(releaseCmd)
}
