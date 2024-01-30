package main

// Quick and dirty exercise of the github adapter
// Auth token is in the environment variable GITHUB_AUTH_TOKEN
// go run github_main.go <org> <repo> [up,down]
// up creates a repo and pushes a release
// down deletes the repo

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	adapters "nhs.uk/coyoteadapters"
	core "nhs.uk/coyotecore"
)

func up(repo string, org string, sourceControl core.IProvideSourceControl) {
	//Check if the repo name is available
	available, err := sourceControl.IsNameAvailable(repo, org)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking if repo name is available: %s\n", err)
		os.Exit(1)
	}
	if !available {
		fmt.Fprintf(os.Stderr, "Repo name %s is not available\n", repo)
		os.Exit(1)
	}
	// Create the repo
	err = sourceControl.CreateRepo(repo, org)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating repo: %s\n", err)
		os.Exit(1)
	}

	// Sleep for 5 seconds to give the repo time to be created
	time.Sleep(5 * time.Second)

	// Create a local repo so that we've got something we can push
	repoDir, err := os.MkdirTemp("", repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary directory: %s\n", err)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating local repo: %s\n", err)
		os.Exit(1)
	}
	cwd := os.Getenv("PWD")
	err = os.Chdir(repoDir)
	defer os.Chdir(cwd)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to local repo: %s\n", err)
		os.Exit(1)
	}
	cmd := exec.Command("git", "init")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initialising local repo: %s\n", err)
		os.Exit(1)
	}
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/"+org+"/"+repo+".git")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding remote to local repo: %s\n", err)
		os.Exit(1)
	}
	err = os.WriteFile("README.md", []byte("# "+repo), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating README.md: %s\n", err)
		os.Exit(1)
	}
	cmd = exec.Command("git", "add", "README.md")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding README.md: %s\n", err)
		os.Exit(1)
	}
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error committing README.md: %s\n", err)
		os.Exit(1)
	}
	cmd = exec.Command("git", "push", "-u", "origin", "HEAD")
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error pushing to remote: %s\n", err)
		os.Exit(1)
	}
	// Make a new file that we can push as a release asset
	err = os.WriteFile("test.txt", []byte("Hello world"), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating test.txt: %s\n", err)
		os.Exit(1)
	}
	// Create a release
	assetURLs, err := sourceControl.CreateRelease(repo, org, "v0.0.1", []string{"test.txt"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating release: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(assetURLs)
}

func down(repo string, org string, sourceControl core.IProvideSourceControl) {
	// Delete the release
	err := sourceControl.DeleteRelease(repo, org, "v0.0.1")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting release: %s\n", err)
		os.Exit(1)
	}
	// Delete the repo
	err = sourceControl.DeleteRepo(repo, org)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting repo: %s\n", err)
		os.Exit(1)
	}
	// Delete the local repo
	err = os.RemoveAll(repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting local repo: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run github_main.go <org> <repo> [up,down]\n")
		os.Exit(1)
	}

	org := args[0]
	repo := args[1]
	cmd := args[2]
	token := os.Getenv("GITHUB_AUTH_TOKEN")

	if token == "" {
		fmt.Fprintf(os.Stderr, "GITHUB_AUTH_TOKEN environment variable not set\n")
		os.Exit(1)
	}

	sourceControl := adapters.NewGithubSourceControl(token)

	if cmd == "up" {
		up(repo, org, sourceControl)
	}
	if cmd == "down" {
		down(repo, org, sourceControl)
	}
}
