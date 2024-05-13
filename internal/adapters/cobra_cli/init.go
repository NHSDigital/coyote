package cobra_cli

import (
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <tech-stack> <project-name>",
	Short: "Initialise a new project",
	Long: "Initialise a new project with the <tech-stack> tech stack, in a new directory named <project-name>. " +
		"The tech stack must be a package name, or \"empty\" for an empty project.",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		techStack := args[0]
		projectName := args[1]
		return core.Init(&Context, techStack, projectName)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
