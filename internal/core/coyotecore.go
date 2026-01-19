package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"
)

func readInstalledPackages() ([]string, error) {
	installedFilename := ".coyote/installed"
	if _, err := os.Stat(installedFilename); os.IsNotExist(err) {
		// No file means nothing installed, so nothing to conflict with
		return []string{}, nil
	}
	installedPackages, err := os.ReadFile(installedFilename)
	if err != nil {
		return []string{}, fmt.Errorf("error reading installed packages: %v", err)
	}
	installedArr := strings.Split(string(installedPackages), "\n")
	for i, installed := range installedArr {
		installedArr[i] = strings.Split(installed, "=")[0]
	}
	return installedArr, nil
}

func conflictingInstalledPackages(pkg PackageFile) []string {
	result := []string{}
	installedPackages, err := readInstalledPackages()
	if err != nil {
		panic(err)
	}

	if len(installedPackages) == 0 {
		return result
	}

	conflicts := pkg.ReadMetadata("CONFLICTS")
	conflicts = strings.TrimSpace(conflicts)
	conflictsArr := strings.Split(conflicts, "\n")

	for _, conflict := range conflictsArr {
		for _, installedName := range installedPackages {
			if conflict == installedName {
				result = append(result, conflict)
			}
		}
	}
	return result
}

func extractPackage(project Project, packageFiles IProvidePackageFiles, filename string) PackageFile {
	// This function extracts the package to the project. We record the installation
	// in .coyote/installed.

	var vars PackageTemplateVars
	vars.ProjectName = project.GetName()

	pkg := packageFiles.Open(filename)

	pkg.Apply(vars)

	err := project.RecordInstalledPackage(pkg)
	if err != nil {
		panic(err)
	}

	return pkg
}

func runOnInstall(pkg PackageFile) error {
	onInstall := pkg.ReadMetadata("on-install")
	if onInstall == "" {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "coyote")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.Write([]byte(onInstall))
	tmpFile.Close()

	cmd := exec.Command("bash", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func Apply(context *Context, filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("package file not found: %v", err)
	}
	project := context.Projects.MaybeProject(".")
	if project == nil {
		return fmt.Errorf("not in a Coyote project")
	}
	pkg := context.PackageFiles.Open(filename)
	conflicts := conflictingInstalledPackages(pkg)
	if len(conflicts) == 0 {
		pkg := extractPackage(project, context.PackageFiles, filename)
		err := runOnInstall(pkg)

		if err != nil {
			return fmt.Errorf("error running on-install script: %v", err)
		} else {
			return nil
		}
	} else {
		return fmt.Errorf("conflicts found: %s", strings.TrimSuffix(strings.Join(conflicts, ", "), "\n"))
	}
}

func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func Init(context *Context, techStack string, projectName string) error {
	// This function creates a new Coyote project, named projectName.
	// The project root dir will be created in the current directory.
	// The name will be stored in projectName/.coyote/projectName.

	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("project %s already exists", projectName)
	}

	cwd := os.Getenv("PWD")
	projectPath := path.Join(cwd, projectName)
	newProject := context.Projects.NewProject(projectPath, projectName)

	if techStack != "empty" {
		os.Chdir(newProject.GetPath())
		defer os.Chdir(cwd)

		index, err := openIndex(context, context.Config.GetIndex())
		if err != nil {
			return fmt.Errorf("error opening index file: %v", err)
		}

		return installPackageTree(newProject,
			context.SourceControl,
			context.PackageFiles,
			techStack,
			index,
			false)
	} else {
		return nil
	}
}

func installPackageTree(project Project, sourceControl IProvideSourceControl, packageFiles IProvidePackageFiles, pkgSpec string, index Index, reinstall bool) error {
	// reinstall *only* applies to the named package. Nothing else is reinstalled.
	// pkgSpec can be "package" or "package@version"

	depQueue := []string{pkgSpec}
	depsToInstall := []string{}

	for len(depQueue) > 0 {
		dep := depQueue[0]
		depQueue = depQueue[1:]
		newDeps, err := index.ReadPackageDependencies(dep)
		if err != nil {
			return fmt.Errorf("error getting dependencies for %s from index file %s: %v",
				dep, index.Describe(), err)
		}
		for _, newDep := range newDeps {
			// Dependencies don't include version specs, just package names
			if !stringInSlice(newDep, depQueue) && !stringInSlice(newDep, depsToInstall) {
				depQueue = append(depQueue, newDep)
			}
		}
		depsToInstall = append(depsToInstall, dep)
	}
	// We have appended the dependencies to the depsToInstall slice as we went through, so
	// now we need to reverse it so that dependencies are installed before the packages that
	// depend on them.
	for i, j := 0, len(depsToInstall)-1; i < j; i, j = i+1, j-1 {
		depsToInstall[i], depsToInstall[j] = depsToInstall[j], depsToInstall[i]
	}

	// We now have the list of dependencies to install, but we need to check for conflicts.
	// We do this by reading the list of installed packages, and then checking each dependency
	// against that list.
	installedPackages, err := readInstalledPackages()
	if err != nil {
		return fmt.Errorf("error reading installed packages: %v", err)
	}

	// Here note that we only check conflicts in one direction.  The dependency needs to be
	// the one that declares the conflict against the thing that is installed, not the other
	// way around.  The practical effect of this is that conflicts need to be declared in
	// both directions.  That could be done at index build time.
	conflictMap := make(map[string][]string)
	for _, dep := range depsToInstall {
		conflicts, err := index.ReadPackageConflicts(dep)
		if err != nil {
			return fmt.Errorf("error getting conflicts for %s from index file %s: %v",
				dep, index.Describe(), err)
		}
		for _, conflict := range conflicts {
			if stringInSlice(conflict, installedPackages) {
				conflictMap[dep] = append(conflictMap[dep], conflict)
			}
		}
	}

	if len(conflictMap) > 0 {
		return fmt.Errorf("conflicts found: %v", conflictMap)
	}

	// Now we have the list of dependencies to install, so we can install them.
	for _, dep := range depsToInstall {
		// we just re-read the list of installed packages each time. It's simpler than managing the list

		installedPackages, err := readInstalledPackages()
		if err != nil {
			return fmt.Errorf("error reading installed packages: %v", err)
		}

		// Parse the package spec to get just the name for comparison with installed packages
		depName, _ := ParsePackageSpec(dep)
		pkgName, _ := ParsePackageSpec(pkgSpec)

		if stringInSlice(depName, installedPackages) && !(reinstall && depName == pkgName) {
			continue
		}

		location, err := index.ReadPackageLocation(dep)
		if err != nil {
			return fmt.Errorf("error getting package location: %v", err)
		}

		if locationIsRemote(location) {
			location, err = sourceControl.DownloadReleaseFile(location)
			if err != nil {
				return fmt.Errorf("error downloading package: %v", err)
			}
			defer os.Remove(location)
		}

		if _, err := os.Stat(location); err != nil {
			return fmt.Errorf("package file missing: %v", err)
		}

		pkg := extractPackage(project, packageFiles, location)
		err = runOnInstall(pkg)

		if err != nil {
			return fmt.Errorf("error running on-install script: %v", err)
		}
	}
	return nil
}

func Install(context *Context, pkgName string, reinstall bool) error {
	project := context.Projects.MaybeProject(".")
	if project == nil {
		return fmt.Errorf("not in a Coyote project")
	}
	indexLocation := context.Config.GetIndex()

	index, err := openIndex(context, indexLocation)
	if err != nil {
		return fmt.Errorf("error opening index file: %v", err)
	}

	return installPackageTree(project, context.SourceControl, context.PackageFiles, pkgName, index, reinstall)
}

func Upgrade(context *Context, pkgNames []string) error {
	project := context.Projects.MaybeProject(".")
	if project == nil {
		return fmt.Errorf("not in a Coyote project")
	}

	// Get all installed packages
	installedPackages, err := project.ReadInstalledPackages()
	if err != nil {
		return fmt.Errorf("error reading installed packages: %v", err)
	}

	// Open the index
	indexLocation := context.Config.GetIndex()
	index, err := openIndex(context, indexLocation)
	if err != nil {
		return fmt.Errorf("error opening index file: %v", err)
	}

	// Determine which packages to upgrade
	var packagesToCheck []string
	if len(pkgNames) == 0 {
		// Upgrade all installed packages
		for _, pkg := range installedPackages {
			if len(pkg) >= 2 {
				packagesToCheck = append(packagesToCheck, pkg[0])
			}
		}
	} else {
		packagesToCheck = pkgNames
	}

	if len(packagesToCheck) == 0 {
		// Nothing to upgrade
		return nil
	}

	// Phase 1: Collect all upgrade targets and validate
	type upgradeTarget struct {
		pkgName          string
		installedVersion string
		targetVersion    PackageVersionEntry
	}
	var upgrades []upgradeTarget

	for _, pkgName := range packagesToCheck {
		// Find the installed version
		var installedVersion string
		for _, pkg := range installedPackages {
			if len(pkg) >= 2 && pkg[0] == pkgName {
				installedVersion = pkg[1]
				break
			}
		}

		if installedVersion == "" {
			// If specific packages were requested, error if not installed
			if len(pkgNames) > 0 {
				return fmt.Errorf("package %s is not installed", pkgName)
			}
			// Otherwise skip (upgrading all, package not installed)
			continue
		}

		// Get the package entry from the index
		pkgEntry, err := index.indexFile.GetPackage(pkgName)
		if err != nil {
			if len(pkgNames) > 0 {
				return fmt.Errorf("package %s not found in index: %v", pkgName, err)
			}
			// Skip packages not in index when upgrading all
			continue
		}

		// Get the latest version
		latestVersion, err := pkgEntry.GetLatestVersion()
		if err != nil {
			if len(pkgNames) > 0 {
				return fmt.Errorf("error getting latest version for %s: %v", pkgName, err)
			}
			continue
		}

		// Compare versions - if already at latest, skip
		if CompareSemanticVersions(installedVersion, latestVersion.Version) >= 0 {
			continue
		}

		upgrades = append(upgrades, upgradeTarget{
			pkgName:          pkgName,
			installedVersion: installedVersion,
			targetVersion:    latestVersion,
		})
	}

	if len(upgrades) == 0 {
		// Nothing to upgrade
		return nil
	}

	// Phase 2: Check all conflicts BEFORE making any changes
	// Build a set of packages that will be upgraded (to exclude from conflict checks)
	upgradeSet := make(map[string]bool)
	for _, u := range upgrades {
		upgradeSet[u.pkgName] = true
	}

	// Build the list of packages that will remain installed (not being upgraded)
	var remainingPackages []string
	for _, pkg := range installedPackages {
		if len(pkg) >= 1 && !upgradeSet[pkg[0]] {
			remainingPackages = append(remainingPackages, pkg[0])
		}
	}

	// Check each upgrade target for conflicts
	for _, u := range upgrades {
		// Check conflicts with packages that will remain installed
		for _, conflict := range u.targetVersion.Conflicts {
			if stringInSlice(conflict, remainingPackages) {
				return fmt.Errorf("cannot upgrade %s to %s: conflicts with installed package %s",
					u.pkgName, u.targetVersion.Version, conflict)
			}
		}

		// Check conflicts with other packages being upgraded
		for _, other := range upgrades {
			if other.pkgName == u.pkgName {
				continue
			}
			if stringInSlice(other.pkgName, u.targetVersion.Conflicts) {
				return fmt.Errorf("cannot upgrade %s to %s: conflicts with %s",
					u.pkgName, u.targetVersion.Version, other.pkgName)
			}
		}
	}

	// Phase 3: All checks passed, perform the actual upgrades
	for _, u := range upgrades {
		location := u.targetVersion.Location
		if locationIsRemote(location) {
			location, err = context.SourceControl.DownloadReleaseFile(location)
			if err != nil {
				return fmt.Errorf("error downloading package %s: %v", u.pkgName, err)
			}
			defer os.Remove(location)
		}

		if _, err := os.Stat(location); err != nil {
			return fmt.Errorf("package file missing for %s: %v", u.pkgName, err)
		}

		pkg := extractPackage(project, context.PackageFiles, location)
		err = runOnInstall(pkg)
		if err != nil {
			return fmt.Errorf("error running on-install script for %s: %v", u.pkgName, err)
		}
	}

	return nil
}

func locationIsRemote(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

func removeComments(body string) []string {
	lines := strings.Split(body, "\n")
	result := []string{}
	for _, line := range lines {
		line = strings.Split(line, "#")[0]
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func readIndexEntry(packageFiles IProvidePackageFiles, packageLocation string) PackageVersionEntry {
	file := packageFiles.Open(packageLocation)

	return PackageVersionEntry{
		Version:      file.ReadMetadata("VERSION"),
		Conflicts:    removeComments(file.ReadMetadata("CONFLICTS")),
		Dependencies: removeComments(file.ReadMetadata("DEPENDS")),
	}
}

func readPackageName(packageFiles IProvidePackageFiles, packageLocation string) string {
	file := packageFiles.Open(packageLocation)
	return file.ReadMetadata("NAME")
}

func sortVersionsDescending(versions []PackageVersionEntry) {
	sort.Slice(versions, func(i, j int) bool {
		return CompareSemanticVersions(versions[i].Version, versions[j].Version) > 0 // Descending order
	})
}

func BuildIndex(context *Context, indexSourceFilename string, indexFilename string) error {
	// This function reads an index source file, and outputs an index file.
	// The index file is a map of package names to locations and dependencies.
	// Multiple versions of the same package can be present; the latest version
	// (by asciibetic sort) is used as the default.
	// Locations in the index source file can be relative to the file being indexed, or absolute.
	// Index file locations will always be absolute.
	//
	// It outputs to stdout.
	// It functions by opening each package, reading its metadata, and then
	// writing it to the index file.

	indexSource, err := os.Open(indexSourceFilename)
	if err != nil {
		return fmt.Errorf("error opening index source file: %v", err)
	}
	defer indexSource.Close()

	// The index file is a json file. At the top level we have `version` and `packages`.
	// `version` is the version of the index file format.
	// `packages` is a map of package names to package metadata.
	// Each package metadata contains a list of versions with their metadata.

	packages := make(map[string]PackageIndexEntry)
	indexSourceScanner := bufio.NewScanner(indexSource)
	for indexSourceScanner.Scan() {
		packageLocation := indexSourceScanner.Text()
		var localLocation string

		// packageLocation can be remote: http:// or https:// mean that we need to download the package.
		if locationIsRemote(packageLocation) {
			localLocation, err = context.SourceControl.DownloadReleaseFile(packageLocation)
			if err != nil {
				return fmt.Errorf("error downloading package %s: %v", packageLocation, err)
			}
			defer os.Remove(localLocation)
		} else {
			// now a local packageLocation is either relative to the index file, or absolute.  We need to
			// force it to be absolute so that the index file can be moved around, and so that readIndexEntry can find it.
			// Note that it is *not* relative to the current directory, it is relative to the index file.
			if !strings.HasPrefix(packageLocation, "/") {
				//Get the directory of the index file, and append the packageLocation to it.
				dir := path.Dir(indexSourceFilename)
				packageLocation = path.Clean(dir + "/" + packageLocation)
			}
			localLocation = packageLocation
		}
		if _, err := os.Stat(localLocation); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("package file missing: %s", localLocation)
		}

		pkgName := readPackageName(context.PackageFiles, localLocation)
		if pkgName == "" {
			// This is a proper "this should never happen" - if we can't get the name of the package
			// then something has gone wrong upstream.
			panic(fmt.Sprintf("package name in %s is empty.", localLocation))
		}

		versionEntry := readIndexEntry(context.PackageFiles, localLocation)
		versionEntry.Location = packageLocation

		if existing, ok := packages[pkgName]; ok {
			// Add this version to the existing package entry
			existing.Versions = append(existing.Versions, versionEntry)
			packages[pkgName] = existing
		} else {
			// Create a new package entry with this version
			packages[pkgName] = PackageIndexEntry{
				Name:     pkgName,
				Versions: []PackageVersionEntry{versionEntry},
			}
		}
	}

	// Sort versions and set the latest version info for each package
	for name, pkg := range packages {
		sortVersionsDescending(pkg.Versions)
		latest := pkg.Versions[0]
		pkg.Version = latest.Version
		pkg.Location = latest.Location
		pkg.Dependencies = latest.Dependencies
		pkg.Conflicts = latest.Conflicts
		packages[name] = pkg
	}

	// Now for each package, we need to check that the conflicts are reflected both ways.
	// We do this by iterating over the conflicts, and then adding the package to the conflicts
	// field of the conflicting package.
	// Note: We only check conflicts for the latest version of each package
	for _, pkg := range packages {
		for _, conflict := range pkg.Conflicts {
			if conflictPkg, ok := packages[conflict]; ok {
				if !stringInSlice(pkg.Name, conflictPkg.Conflicts) {
					// Add to the package-level conflicts
					conflictPkg.Conflicts = append(conflictPkg.Conflicts, pkg.Name)
					// Also add to each version's conflicts
					for i := range conflictPkg.Versions {
						if !stringInSlice(pkg.Name, conflictPkg.Versions[i].Conflicts) {
							conflictPkg.Versions[i].Conflicts = append(conflictPkg.Versions[i].Conflicts, pkg.Name)
						}
					}
					packages[conflict] = conflictPkg
				}
			} else {
				// We don't have the named package in the index. The most dangerous case is that
				// it's a typo and someone will install incompatible packages, so we need to barf.
				return fmt.Errorf("package %s conflicts with %s, but %s is not in the index",
					pkg.Name, conflict, conflict)
			}
		}
	}

	indexFile, err := os.Create(indexFilename)
	if err != nil {
		return fmt.Errorf("error creating index file: %v", err)
	}
	defer indexFile.Close()

	index := IndexData{}
	index.Version = "1"
	index.Packages = packages

	json.NewEncoder(indexFile).Encode(index)
	return nil
}

func PackageInit(context *Context, pkgName string) error {
	return context.PackageFiles.Init(pkgName)
}

func PackageBuild(context *Context, pkgName string, outDir string, version string) (string, error) {
	return context.PackageFiles.Build(pkgName, outDir, version)
}

func PackageBuildAll(context *Context, outDir string, version string) ([]string, error) {
	packages, err := context.PackageFiles.ListPackages()
	if err != nil {
		return nil, err
	}

	var results []string
	for _, pkg := range packages {
		filename, err := context.PackageFiles.Build(pkg, outDir, version)
		if err != nil {
			return nil, fmt.Errorf("error building package %s: %v", pkg, err)
		}
		results = append(results, filename)
	}
	return results, nil
}

func PackageVersion(context *Context) (string, error) {
	// This function returns the version of the package that would be built
	// by a raw `coyote package build` command.
	// It does this by reading the version from the git tags, the same way
	// that `package build` would if you didn't specify a version.

	// Barf if we're not in a git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return "", fmt.Errorf("not in a git repository")
	}

	// Get the current version from the git tags
	output, err := context.PackageFiles.Version()
	if err != nil {
		return "", fmt.Errorf("error getting version: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// coyote package new <pkgName>
// This creates a new package in the current directory, and pushes it
// to github.
// At some point we will undoubtedly want to have a template for this,
// but for now we just create the files.
func PackageNew(context *Context, pkgName string) error {
	// First we make the new dir.  The name of the dir will match our
	// remote.  We need to check that the name is available first.

	sourceControl := context.SourceControl
	intendedName := "cypkg-" + pkgName
	packageOrg := context.Config.GetPackageOrg()
	available, err := sourceControl.IsNameAvailable(intendedName, packageOrg)
	if err != nil {
		return fmt.Errorf("error checking name availability: %v", err)
	}
	if !available {
		return fmt.Errorf("name %s is already taken", intendedName)
	}

	actualName := intendedName

	// Now we create the local dir, and initialise it as a git repo.
	os.MkdirAll(actualName, 0777)
	cwd := os.Getenv("PWD")
	err = os.Chdir(actualName)
	if err != nil {
		return fmt.Errorf("error changing to new directory: %v", err)
	}
	defer os.Chdir(cwd)

	err = exec.Command("git", "init").Run()
	if err != nil {
		return fmt.Errorf("error initialising git repo: %v", err)
	}
	// Force the main branch to be called main.
	err = exec.Command("git", "branch", "-M", "main").Run()
	if err != nil {
		return fmt.Errorf("error setting main branch: %v", err)
	}
	// Now we can run Init to create the package files.
	context.PackageFiles.Init(pkgName)
	// git add, git commit...
	err = exec.Command("git", "add", ".").Run()
	if err != nil {
		return fmt.Errorf("error adding files to git repo: %v", err)
	}
	err = exec.Command("git", "commit", "-m", "Initial commit").Run()
	if err != nil {
		return fmt.Errorf("error committing files to git repo: %v", err)
	}
	// Now we can create the remote repo.
	err = sourceControl.CreateRepo(actualName, packageOrg)
	if err != nil {
		return fmt.Errorf("error creating remote repo: %v", err)
	}
	// We need to loop here until the remote repo is actually created, which
	// we check by seeing if the name is available
	for {
		available, err = sourceControl.IsNameAvailable(actualName, packageOrg)
		if err != nil {
			return fmt.Errorf("error checking name availability: %v", err)
		}
		if !available {
			break
		} else {
			time.Sleep(time.Duration(sourceControl.GetRateLimitDelayMilliseconds()))
		}
	}

	remoteURL := sourceControl.GetRemoteURL(actualName, packageOrg)

	// Now we can set the remote and push
	err = exec.Command("git", "remote", "add", "origin", remoteURL).Run()
	if err != nil {
		return fmt.Errorf("error adding remote: %v", err)
	}

	return context.SourceControl.Push(actualName, packageOrg)
}

func PackageDelete(context *Context, pkgName string) error {
	// This function deletes the named package from github.
	// It does not delete the local copy of the package.
	// It does not check that the package is not in use.
	// It does not check that the package is not a dependency of another package.
	// It does not check that the package is not a dependency of the project.
	// It will make you sad if you use it wrong.

	sourceControl := context.SourceControl
	packageOrg := context.Config.GetPackageOrg()
	err := sourceControl.DeleteRepo("cypkg-"+pkgName, packageOrg)
	if err != nil {
		return fmt.Errorf("error deleting remote repo: %v", err)
	}
	return nil
}

func repoHasOriginSet(origin string) (bool, error) {
	remotes, err := exec.Command("git", "remote").Output()
	if err != nil {
		return false, fmt.Errorf("error getting remote list: %v", err)
	}
	return strings.Contains(string(remotes)+"\n", origin), nil
}

func Open(context *Context) error {
	// If we're in a github repo, open the origin remote repo in the browser.
	remoteToOpen := "origin"
	remoteExists, err := repoHasOriginSet(remoteToOpen)
	if err != nil {
		return fmt.Errorf("error checking for remote: %v", err)
	} else if !remoteExists {
		return fmt.Errorf("no %s remote found", remoteToOpen)
	}

	remote, err := exec.Command("git", "remote", "get-url", remoteToOpen).Output()
	if err != nil {
		return fmt.Errorf("error getting remote url: %v", err)
	}

	platform := context.Platform
	return platform.OpenURL(strings.TrimSpace(string(remote)))
}

func pushTagsToOrigin() error {
	// TODO this is potentially hazardous, because it pushes all tags and ignores whether
	// what's checked out matches the version we're pushing.  Ok for a demo though.
	return exec.Command("git", "push", "origin", "--follow-tags").Run()
}

func commitForTag(tag string) (string, error) {
	commitHashBuf, err := exec.Command("git", "rev-list", "-n", "1", tag).Output()
	if err != nil {
		return "", fmt.Errorf("error getting commit hash for tag: %v", err)
	}
	return strings.TrimSpace(string(commitHashBuf)), nil
}

func commitExistsAtOrigin(context *Context, pkgName string, tag string) (bool, error) {
	commitHash, err := commitForTag(tag)
	if err != nil {
		return false, err
	}

	// Now check if the commit hash exists on any branch at the origin
	output, err := exec.Command("git", "branch", "-r", "--contains", commitHash).Output()
	if err != nil {
		return false, fmt.Errorf("error checking for commit at origin: %v", err)
	}
	return strings.Contains(string(output), "origin/"), nil
}

func PackageRelease(context *Context, pkgName string, version string) (string, error) {
	//Barf if we're not in a coyote package
	if _, err := os.Stat(".cypkg"); os.IsNotExist(err) {
		return "", fmt.Errorf("not in a Coyote package")
	}

	// Bad things will happen if we get version=="HEAD" here, so don't do that
	if version == "HEAD" {
		return "", fmt.Errorf("cannot release HEAD version")
	}

	tag, err := tagForRelease(version, context, pkgName)
	if err != nil {
		return "", err
	}

	commitHash, err := commitForTag(tag)
	if err != nil {
		return "", fmt.Errorf("error getting commit hash for tag: %v", err)
	}

	// Check that the tag exists at the origin. If we don't do this, then trying to push
	// the tag will fail, the release won't run, and we'll have to clean up the tag locally.
	commitExists, err := commitExistsAtOrigin(context, pkgName, commitHash)
	if err != nil {
		return "", fmt.Errorf("error checking for tag at origin: %v", err)
	}
	if !commitExists {
		return "", fmt.Errorf("the release commit %s does not exist at the origin", commitHash)
	}

	// We know the tag exists in the repo, so we can now build the tag.

	packagePath, err := context.PackageFiles.Build(pkgName, ".", version)
	if err != nil {
		return "", fmt.Errorf("error building package: %v", err)
	}

	err = pushTagsToOrigin()
	if err != nil {
		return "", fmt.Errorf("error pushing tag to remote: %v", err)
	}

	return releaseFiles(context, "cypkg-"+pkgName, tag, []string{packagePath})
}

// This function does all the preflight checks to ensure that the version tag we're
// asking for exists in the repository, and hasn't been released already.  This includes
// tagging locally.
// It returns the tag that was actually written to the repo.
func tagForRelease(version string, context *Context, pkgName string) (string, error) {
	tag := "coyote-" + version

	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return "", fmt.Errorf("not in a git repository")
	}

	remoteToOpen := "origin"
	remoteExists, err := repoHasOriginSet(remoteToOpen)
	if err != nil {
		return "", fmt.Errorf("error checking for remote: %v", err)
	} else if !remoteExists {
		return "", fmt.Errorf("no %s remote found", remoteToOpen)
	}

	err = exec.Command("git", "check-ref-format", "--allow-onelevel", tag).Run()
	if err != nil {
		return "", fmt.Errorf("invalid version: %v", err)
	}

	sourceControl := context.SourceControl
	packageOrg := context.Config.GetPackageOrg()

	releaseExists, err := sourceControl.DoesReleaseExist(pkgName, packageOrg, version)
	if err != nil {
		return "", fmt.Errorf("error checking if release exists: %v", err)
	}
	if releaseExists {
		return "", fmt.Errorf("release %s already exists", version)
	}

	output, err := exec.Command("git", "tag", "--list", tag).Output()
	if err != nil {
		return "", fmt.Errorf("error checking for existing tag: %v", err)
	}
	if strings.TrimSpace(string(output)) != tag {
		// We use an annotated tag here so that we keep authorship
		// TODO: check whether tag signing can work here
		// TODO: what can usefully go in the tag message?
		cmd := exec.Command("git", "tag", "--annotate", "-m", "No tag message", tag)
		_, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("error creating tag %v: %v", tag, err)
		}
	}
	return tag, nil
}

func PackageTest(context *Context, pkgName string) error {
	//All this test does is make a temp dir, build the package into it, and then apply it.
	// It will barf if there are any errors in the templates.
	tempDir, err := os.MkdirTemp("", "coyote-test")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	packagePath, err := context.PackageFiles.Build(pkgName, tempDir, "HEAD")
	if err != nil {
		return fmt.Errorf("error building package: %v", err)
	}

	// Now we apply the package to the temp dir.
	// We need to change to the temp dir first, and then change back.
	cwd := os.Getenv("PWD")
	err = os.Chdir(tempDir)
	if err != nil {
		return fmt.Errorf("error changing to temp dir: %v", err)
	}
	defer os.Chdir(cwd)
	// Make ourselves a coyote project dir
	os.MkdirAll(".coyote", 0777)
	os.WriteFile(".coyote/project-name", []byte("test"), 0777)
	// Now we can apply the package.
	// NOTE: THIS WILL RUN on-install IF IT EXISTS.
	return Apply(context, packagePath)
}

// url, err = core.ReleaseIndex(&Context, args[0], args[1])
func ReleaseIndex(context *Context, indexSrcInput string, versionToReleaseAs string) (string, error) {
	// This function builds an index file and uploads it as a release.
	// It returns the URL of the release, or an error.

	// TODO: hardcode for now, figure out how to support more than one repo index later
	repoName := "coyote-index"

	// Barf if the input file doesn't exist
	if _, err := os.Stat(indexSrcInput); os.IsNotExist(err) {
		return "", fmt.Errorf("index source file not found: %v", indexSrcInput)
	}

	tag, err := tagForRelease(versionToReleaseAs, context, repoName)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "coyote")
	if err != nil {
		return "", fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	indexFilePath := path.Join(tempDir, "index")
	indexFile, err := os.Create(indexFilePath)
	if err != nil {
		return "", fmt.Errorf("error creating index file: %v", err)
	}
	defer indexFile.Close()
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(indexFile.Name())

	err = BuildIndex(context, indexSrcInput, indexFile.Name())
	if err != nil {
		return "", fmt.Errorf("error building index: %v", err)
	}

	return releaseFiles(context, repoName, tag, []string{indexFile.Name()})
}

func releaseFiles(context *Context, repoName string, tag string, filesToRelease []string) (string, error) {
	sourceControl := context.SourceControl
	indexOrg := context.Config.GetPackageOrg()

	assetURLs, err := sourceControl.CreateRelease(repoName, indexOrg, tag, filesToRelease)
	if err != nil {
		return "", fmt.Errorf("error creating release: %v", err)
	}

	return assetURLs[0], nil
}

func Release(context *Context, version string, description string, remoteName string, filenames []string) ([]string, error) {
	// This function releases the named files. It is not only for releasing packages, it can
	// release anything.
	// It uploads the passed files to github as a release, using the `origin` git remote as the target repository.
	// It returns the URL of the release, or an error.

	// Barf if we're not in a git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return nil, fmt.Errorf("not in a git repository")
	}

	// Barf if the version is HEAD
	if version == "HEAD" {
		return nil, fmt.Errorf("cannot release HEAD version")
	}

	// Gather the information we need to create the release
	// get the org name and the repo name from the git remote
	remoteExists, err := repoHasOriginSet(remoteName)
	if err != nil {
		return nil, fmt.Errorf("error checking for remote: %v", err)
	} else if !remoteExists {
		return nil, fmt.Errorf("no %s remote found", remoteName)
	}

	remote, err := exec.Command("git", "remote", "get-url", remoteName).Output()
	if err != nil {
		return nil, fmt.Errorf("error getting remote url: %v", err)
	}

	// double-check that the remote is a github repo
	if !strings.Contains(string(remote), "github.com") {
		return nil, fmt.Errorf("remote is not a github repo")
	}

	// get the org and repo names
	remoteParts := strings.Split(strings.TrimSpace(string(remote)), "/")
	repoName := strings.TrimSuffix(remoteParts[len(remoteParts)-1], ".git")
	orgName := remoteParts[len(remoteParts)-2]

	// Now just do the release: if any of these params are wrong then the API will barf for us
	assetURLs, err := context.SourceControl.CreateRelease(repoName, orgName, version, filenames)
	return assetURLs, err
}
