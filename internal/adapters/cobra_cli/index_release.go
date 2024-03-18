package cobra_cli

import (
	"fmt"

	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

// TODO: This doesn't feel quite right, in that we ask for an index source file but we
// operate on the version tag, which is repository-wide.  That means we really only
// support a single index file per index repo, which is a bit limiting.
// It'll do for now though.
var indexReleaseCmd = &cobra.Command{
	Use:   "release <index-src-input> <version-to-release-as>",
	Short: "Build an index file and upload it as a release",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		url, err := core.ReleaseIndex(&Context, args[0], args[1])
		if err == nil {
			fmt.Println(url)
		}
		return err
	},
}

func init() {
	indexCmd.AddCommand(indexReleaseCmd)
}
