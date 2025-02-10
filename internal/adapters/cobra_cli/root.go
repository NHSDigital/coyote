package cobra_cli

import (
	"fmt"
	"os"

	adapters "github.com/nhsdigital/coyote/internal/adapters"
	core "github.com/nhsdigital/coyote/internal/core"
	"github.com/spf13/cobra"
)

var ConfigPath string
var UseFakeGithub bool
var overrideIndexPath string

var Context core.Context

var rootCmd = &cobra.Command{
	Use:   "coyote [flags] [command]",
	Short: "Package management for repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Usage()
		return nil
	},
}

func Execute() {
	cobra.OnInitialize(initContext)

	rootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "", "Path to the .coyoterc config file")
	rootCmd.PersistentFlags().BoolVar(&UseFakeGithub, "fake-github", false, "If set, will not communicate with the real Github.")
	rootCmd.PersistentFlags().StringVarP(&overrideIndexPath, "index", "i", "", "Location of the index file, overriding the config file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops, something went wrong: %v", err)
		os.Exit(1)
	}
}

func getConfig(overrideIndexPath string) (core.Config, error) {
	configPath := ConfigPath
	if configPath == "" {
		configPath = os.ExpandEnv("${COYOTE_CONFIG}")
	}
	if configPath == "" {
		configPath = os.ExpandEnv("${HOME}/.coyoterc")
	}

	parsedConfig, err := adapters.NewTomlConfig(configPath)
	if err != nil {
		return nil, err
	}
	if overrideIndexPath != "" {
		parsedConfig.Index = overrideIndexPath
	}
	return parsedConfig, nil
}

func initContext() {
	// TODO: Overrides want handling in a better way than
	// one by one, but this is a start
	config, err := getConfig(overrideIndexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	var sourceControl core.IProvideSourceControl
	if UseFakeGithub {
		sourceControl = core.NewNullSourceControl()
	} else {
		sourceControl = adapters.NewGithubSourceControl(os.Getenv("GITHUB_TOKEN"))
	}

	Context = core.Context{
		Config:        config,
		PackageFiles:  adapters.NewPackageTarFileProvider(),
		SourceControl: sourceControl,
		Platform:      adapters.NewPlatform(),
		Projects:      adapters.NewProjectProvider(),
		IndexFiles:    adapters.NewIndexFileProvider(),
	}
}
