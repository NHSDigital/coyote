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
		sourceControl = adapters.NewGithubSourceControl(os.Getenv("GITHUB_AUTH_TOKEN"))
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

/*
func RunRelease(context *core.Context, pkgname string, version string) {

	assetURL, err := core.PackageRelease(context, pkgname, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error releasing package: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(assetURL)
}

func RunPackage(context *core.Context, args []string) {
	subcmd := args[0]
	pkgname := args[1]

	outdir := "."
	for i, arg := range args {
		if arg == "--output" {
			outdir = args[i+1]
			args = append(args[:i], args[i+2:]...)
		}
	}

	switch subcmd {
	case "init":
		core.PackageInit(context, pkgname)
	case "build":
		// By default we build the latest version
		version := ""
		if len(args) > 2 {
			version = args[2]
		}
		RunPackageBuild(context, pkgname, outdir, version)
	case "new":
		core.PackageNew(context, pkgname)
	case "delete":
		core.PackageDelete(context, pkgname)
	case "release":
		version := args[2] // not optional here
		RunRelease(context, pkgname, version)
	case "test":
		core.PackageTest(context, pkgname)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: package %s\n", subcmd)
		os.Exit(1)
	}
}


func RunApply(context *core.Context, args []string) {
	filename := args[0]
	core.Apply(context, filename)
}


func Run(context *core.Context, args []string) {
	cmd := args[0]
	switch cmd {
	case "package":
		RunPackage(context, args[1:])
	case "apply":
		RunApply(context, args[1:])
	case "init":
		RunInit(context, args[1:])
	case "index":
		RunIndex(context, args[1:])
	case "install":
		RunInstall(context, args[1:])
	case "config":
		RunConfig(context, args[1:])
	case "open":
		RunOpen(context)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func main() {
	// pull --config out of args, because everything else needs it
	// and we don't want to pass it to them
	config, err := getConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	// Here we pre-parse the args to pull out any arguments that change the context
	// In the first instance that's to allow us to insert a null source control
	// adapter for testing purposes

	var sourceControl core.IProvideSourceControl
	for i, arg := range os.Args {
		if arg == "--fake-github" {
			sourceControl = core.NewNullSourceControl()
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			break
		}
	}
	if sourceControl == nil {
		sourceControl = adapters.NewGithubSourceControl(os.Getenv("GITHUB_AUTH_TOKEN"))
	}

	context := core.Context{
		Config:        config,
		PackageFiles:  adapters.NewPackageTarFileProvider(),
		SourceControl: sourceControl,
		Platform:      adapters.NewPlatform(),
	}

	//copy os.Args to a new slice, because we don't want to pass the first
	//argument or --config to Run
	args := make([]string, len(os.Args)-1)
	copy(args, os.Args[1:])
	//now we can cut out the --config argument
	for i, arg := range args {
		if arg == "--config" {
			args = append(args[:i], args[i+2:]...)
		}
	}

	Run(&context, args)
}
*/
