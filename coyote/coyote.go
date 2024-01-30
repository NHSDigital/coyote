package main

import (
	"fmt"
	"os"

	adapters "nhs.uk/coyoteadapters"
	core "nhs.uk/coyotecore"
)

func RunPackage(context *core.Context, args []string) {
	subcmd := args[0]
	pkgname := args[1]

	outdir := "."
	for i, arg := range args {
		if arg == "--output" {
			outdir = args[i+1]
		}
	}

	switch subcmd {
	case "init":
		core.PackageInit(context, pkgname)
	case "build":
		core.PackageBuild(context, pkgname, outdir)
	case "new":
		core.PackageNew(context, pkgname)
	case "delete":
		core.PackageDelete(context, pkgname)

	}
}

func RunApply(context *core.Context, args []string) {
	filename := args[0]
	core.Apply(context, filename)
}

func RunInit(context *core.Context, args []string) {
	projectName := args[1]
	techStack := args[0]
	indexLocation := context.Config.GetIndex()
	for i, arg := range args {
		if arg == "--index" {
			indexLocation = args[i+1]
		}
	}
	err := core.Init(context, techStack, projectName, indexLocation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initialising project: %s\n", err)
		os.Exit(1)
	}
}

func RunIndex(context *core.Context, args []string) {

	// Ignore the first argument, which is the subcommand. Unused for the moment.
	indexSourceFilename := args[1]
	indexFilename := args[2]

	err := core.BuildIndex(context, indexSourceFilename, indexFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building index: %s\n", err)
		os.Exit(1)
	}
}

func RunInstall(context *core.Context, args []string) {
	pkgname := args[0]
	indexLocation := context.Config.GetIndex()
	reinstall := false
	for i, arg := range args {
		if arg == "--index" {
			indexLocation = args[i+1]
		}
		if arg == "--reinstall" {
			reinstall = true
		}
	}
	err := core.Install(context, pkgname, indexLocation, reinstall)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing package: %s\n", err)
		os.Exit(1)
	}
}

func RunConfig(context *core.Context, args []string) {
	// only one subcommand for now, `path`, which prints the path to the config file
	cmd := args[0]
	switch cmd {
	case "path":
		fmt.Println(context.Config.GetPath())
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", cmd)
		os.Exit(1)
	}
}

func getConfig(args []string) (core.Config, error) {
	defaultConfigPath := os.ExpandEnv("${HOME}/.coyoterc")

	configPath := os.ExpandEnv("${COYOTE_CONFIG}")

	if configPath == "" {
		configPath = defaultConfigPath
	}

	for i, arg := range args {
		if arg == "--config" {
			configPath = args[i+1]
		}
	}

	return adapters.NewTomlConfig(configPath)
}

func RunOpen(context *core.Context) {
	err := core.Open(context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening repo: %s\n", err)
		os.Exit(1)
	}
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
