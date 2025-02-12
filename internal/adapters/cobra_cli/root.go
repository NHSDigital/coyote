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

var overridePackageOrg string

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
	rootCmd.PersistentFlags().StringVar(&overridePackageOrg, "package-org", "", "The github org where we publish packages, overriding the config file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops, something went wrong: %v", err)
		os.Exit(1)
	}
}

type FlagConfig struct {
	Index        string
	PackageOrg   string
	NestedConfig *core.Config
}

func NewFlagConfig(index string, packageOrg string, nested *core.Config) *FlagConfig {
	if nested == nil {
		panic("nested config must not be nil")
	}
	return &FlagConfig{
		Index:        index,
		PackageOrg:   packageOrg,
		NestedConfig: nested,
	}
}

func (c *FlagConfig) GetIndex() string {
	if c.Index != "" {
		return c.Index
	}
	return (*c.NestedConfig).GetIndex()
}

func (c *FlagConfig) GetPackageOrg() string {
	if c.PackageOrg != "" {
		return c.PackageOrg
	}
	return (*c.NestedConfig).GetPackageOrg()
}

func (c *FlagConfig) GetPath() string {
	return (*c.NestedConfig).GetPath()
}

func wrapWithFlagConfig(nested core.Config, index string, packageOrg string) (core.Config, error) {
	return NewFlagConfig(index, packageOrg, &nested), nil
}

func configFromPathOptions() (core.Config, error) {
	// The config file is loaded in the following order:
	// If you have set a flag, and that file does not exist, that's an error. Otherwise use that path.
	// If you have set the environment variable, and that file does not exist, that's an error.  Otherwise use that path.
	// If you have not set either, and the default file does not exist, that's not an error: you just
	// get the null config.  The wrapFlagConfig function will provide values or error if nothing is set for a required field.
	// If you have not set either, and the default file does exist, use that file.

	defaultConfigPath := os.ExpandEnv("${HOME}/.coyoterc")
	envConfigPath := os.ExpandEnv("${COYOTE_CONFIG}")
	flagConfigPath := ConfigPath

	if flagConfigPath != "" {
		if _, err := os.Stat(flagConfigPath); err != nil {
			return nil, fmt.Errorf("--config (-c) is set to %s, but the file does not exist", flagConfigPath)
		}
		return adapters.NewTomlConfig(flagConfigPath)
	} else if envConfigPath != "" {
		if _, err := os.Stat(envConfigPath); err != nil {
			return nil, fmt.Errorf("COYOTE_CONFIG is set to %s, but the file does not exist", envConfigPath)
		}
		return adapters.NewTomlConfig(envConfigPath)

	} else if _, err := os.Stat(defaultConfigPath); err == nil {
		return adapters.NewTomlConfig(defaultConfigPath)
	} else {
		return &core.NullConfig{}, nil
	}
}

func getTomlConfig(overrideIndexPath string) (core.Config, error) {

	parsedConfig, err := configFromPathOptions()

	if err != nil {
		return nil, err
	}
	return wrapWithFlagConfig(parsedConfig, overrideIndexPath, overridePackageOrg)
}

func initContext() {
	config, err := getTomlConfig(overrideIndexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v", err)
		os.Exit(1)
	}

	var sourceControl core.IProvideSourceControl
	if UseFakeGithub {
		sourceControl = core.NewNullSourceControl()
	} else {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			fmt.Fprintf(os.Stderr, "GITHUB_TOKEN environment variable is not set.  This is required for Github operations.")
			os.Exit(1)
		}
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
