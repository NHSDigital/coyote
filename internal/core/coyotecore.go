package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

func readInstalledPackages() ([]string, error) {
	installedFilename := ".coyote/installed"
	if _, err := os.Stat(installedFilename); os.IsNotExist(err) {
		// No file means nothing installed, so nothing to conflict with
		return []string{}, nil
	}
	installeds, err := os.ReadFile(installedFilename)
	if err != nil {
		return []string{}, fmt.Errorf("Error reading installed packages: %v", err)
	}
	installedArr := strings.Split(string(installeds), "\n")
	for i, installed := range installedArr {
		installedArr[i] = strings.Split(installed, "=")[0]
	}
	return installedArr, nil
}

func conflictingInstalledPackages(pkg PackageFile) []string {
	result := []string{}
	installeds, err := readInstalledPackages()
	if err != nil {
		panic(err)
	}

	if len(installeds) == 0 {
		return result
	}

	conflicts := pkg.ReadMetadata("CONFLICTS")
	conflicts = strings.TrimSpace(conflicts)
	conflictsArr := strings.Split(conflicts, "\n")

	for _, conflict := range conflictsArr {
		for _, installedName := range installeds {
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

	project.RecordInstalledPackage(pkg)

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
	project := context.Projects.MaybeProject(".")
	if project == nil {
		return fmt.Errorf("Not in a Coyote project.\n")
	}
	pkg := context.PackageFiles.Open(filename)
	conflicts := conflictingInstalledPackages(pkg)
	if len(conflicts) == 0 {
		pkg := extractPackage(project, context.PackageFiles, filename)
		err := runOnInstall(pkg)

		if err != nil {
			return fmt.Errorf("Error running on-install script: %v\n", err)
		} else {
			return nil
		}
	} else {
		return fmt.Errorf("Conflicts found: %s\n", strings.Join(conflicts, ", "))
	}
}

func checkForCoyoteProject() {
	if _, err := os.Stat(".coyote/project-name"); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Not in a Coyote project.\n")
		os.Exit(1)
	}
}

type IndexFile struct {
	filename string
	contents Index
}

func openIndexFile(filename string) (IndexFile, error) {

	st, err := os.Stat(filename)
	if err != nil {
		return IndexFile{}, fmt.Errorf("Index file %s does not exist.", filename)
	}
	if st.Mode().IsDir() {
		return IndexFile{}, fmt.Errorf("Index file %s is a directory, not a file.", filename)
	}

	indexFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer indexFile.Close()

	indexBytes, err := io.ReadAll(indexFile)
	if err != nil {
		panic(err)
	}

	var indexData Index
	err = json.Unmarshal(indexBytes, &indexData)
	if err != nil {
		return IndexFile{}, fmt.Errorf("Error parsing index file %s: %v", filename, err)
	}

	//Use the absolute path to the index file so we can use it if we change directories.
	absFilename, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	absFilename += "/" + filename

	return IndexFile{filename: absFilename, contents: indexData}, nil
}

func (indexFile IndexFile) readPackageLocation(pkgname string) (string, error) {
	// This function reads the index file, and returns the location of the package named by pkgname.
	// It returns an empty string if the package is not found.

	indexData := indexFile.contents
	if _, ok := indexData.Packages[pkgname]; !ok {
		return "", fmt.Errorf("Package %s not found in index file %s.", pkgname, indexFile.filename)
	}
	return indexData.Packages[pkgname].Location, nil
}

func (indexFile IndexFile) readPackageDependencies(pkgname string) ([]string, error) {
	// This function reads the index file, and returns the dependencies of the package named by pkgname.
	// It returns an empty slice if the package is not found.

	indexData := indexFile.contents
	if _, ok := indexData.Packages[pkgname]; !ok {
		return []string{}, fmt.Errorf("Package %s not found in index file %s.", pkgname, indexFile.filename)
	}
	return indexData.Packages[pkgname].Dependencies, nil
}

func (indexFile IndexFile) readPackageConflicts(pkgname string) ([]string, error) {
	// This function reads the index file, and returns the conflicts of the package named by pkgname.
	// It returns an empty slice if the package is not found.

	indexData := indexFile.contents
	if _, ok := indexData.Packages[pkgname]; !ok {
		return []string{}, fmt.Errorf("Package %s not found in index file %s.", pkgname, indexFile.filename)
	}
	return indexData.Packages[pkgname].Conflicts, nil
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
	// The project will be created in the current directory.
	// The name will be stored in .coyote/project-name.

	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("Project %s already exists.\n", projectName)
	}
	// Get the current working directory so we can return to it later.
	cwd := os.Getenv("PWD")
	// Now we make the project directory and store the name in .coyote/project-name.
	newProject := context.Projects.NewProject(cwd, projectName)
	os.MkdirAll(projectName+"/.coyote", 0777)
	os.WriteFile(projectName+"/.coyote/project-name", []byte(projectName), 0777)

	if techStack != "empty" {

		indexFile, err := openIndexFile(context.Config.GetIndex())
		if err != nil {
			return fmt.Errorf("Error opening index file: %v", err)
		}
		//Now we extract the package to the project directory. After this, index is
		//no longer valid, use indexFile.filename instead.
		os.Chdir(projectName)
		defer os.Chdir("..")

		return installPackageTree(newProject,
			context.SourceControl,
			context.PackageFiles,
			techStack,
			indexFile,
			false)
	} else {
		return nil
	}
}

func installPackageTree(project Project, sourceControl IProvideSourceControl, packageFiles IProvidePackageFiles, pkg string, indexFile IndexFile, reinstall bool) error {
	// reinstall *only* applies to the named package. Nothing else is reinstalled.

	depQueue := []string{pkg}
	depsToInstall := []string{}

	for len(depQueue) > 0 {
		dep := depQueue[0]
		depQueue = depQueue[1:]
		newDeps, err := indexFile.readPackageDependencies(dep)
		if err != nil {
			return fmt.Errorf("Error getting dependencies for %s from index file %s: %v\n",
				dep, indexFile.filename, err)
		}
		for _, newDep := range newDeps {
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
	installeds, err := readInstalledPackages()
	if err != nil {
		return fmt.Errorf("Error reading installed packages: %v", err)
	}

	// Here note that we only check conflicts in one direction.  The dependency needs to be
	// the one that declares the conflict against the thing that is installed, not the other
	// way around.  The practical effect of this is that conflicts need to be declared in
	// both directions.  That could be done at index build time.
	conflictMap := make(map[string][]string)
	for _, dep := range depsToInstall {
		conflicts, err := indexFile.readPackageConflicts(dep)
		if err != nil {
			return fmt.Errorf("Error getting conflicts for %s from index file %s: %v\n",
				dep, indexFile.filename, err)
		}
		for _, conflict := range conflicts {
			if stringInSlice(conflict, installeds) {
				conflictMap[dep] = append(conflictMap[dep], conflict)
			}
		}
	}

	if len(conflictMap) > 0 {
		return fmt.Errorf("Conflicts found: %v\n", conflictMap)
	}

	// Now we have the list of dependencies to install, so we can install them.
	for _, dep := range depsToInstall {
		// we just re-read the list of installed packages each time. It's simpler than managing the list

		installeds, err := readInstalledPackages()
		if err != nil {
			return fmt.Errorf("Error reading installed packages: %v", err)
		}

		if stringInSlice(dep, installeds) && !(reinstall && dep == pkg) {
			continue
		}

		location, err := indexFile.readPackageLocation(dep)
		if err != nil {
			return fmt.Errorf("Error getting package location: %v\n", err)
		}

		if locationIsRemote(location) {
			location, err = sourceControl.DownloadReleaseFile(location)
			if err != nil {
				return fmt.Errorf("Error downloading package: %v\n", err)
			}
			defer os.Remove(location)
		}

		if _, err := os.Stat(location); err != nil {
			return fmt.Errorf("Package file missing: %v\n", err)
		}

		pkg := extractPackage(project, packageFiles, location)
		err = runOnInstall(pkg)

		if err != nil {
			return fmt.Errorf("Error running on-install script: %v\n", err)
		}
	}
	return nil
}

func Install(context *Context, pkgname string, reinstall bool) error {
	project := context.Projects.MaybeProject(".")
	if project == nil {
		return fmt.Errorf("Not in a Coyote project.\n")
	}
	index := context.Config.GetIndex()
	localIndex := index
	if locationIsRemote(index) {
		myLocalIndex, err := context.SourceControl.DownloadReleaseFile(index)
		localIndex = myLocalIndex
		if err != nil {
			return fmt.Errorf("Error downloading index file: %v\n", err)
		}
		defer os.Remove(localIndex)
	}

	indexFile, err := openIndexFile(localIndex)
	if err != nil {
		return fmt.Errorf("Error opening index file: %v\n", err)
	}

	return installPackageTree(project, context.SourceControl, context.PackageFiles, pkgname, indexFile, reinstall)
}

type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Location     string   `json:"location"`
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
}

type Index struct {
	Version  string             `json:"version"`
	Packages map[string]Package `json:"packages"`
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

func readIndexEntry(packageFiles IProvidePackageFiles, packageLocation string) Package {
	file := packageFiles.Open(packageLocation)

	return Package{
		Name:         file.ReadMetadata("NAME"),
		Version:      file.ReadMetadata("VERSION"),
		Conflicts:    removeComments(file.ReadMetadata("CONFLICTS")),
		Dependencies: removeComments(file.ReadMetadata("DEPENDS")),
	}
}

func BuildIndex(context *Context, indexSourceFilename string, indexFilename string) error {
	// This function reads an index source file, and outputs an index file.
	// The index file is a map of package names to locations and dependencies.
	// Locations in the index source file can be relative to the file being indexed, or absolute.
	// Index file locations will always be absolute.
	//
	// It outputs to stdout.
	// It functions by opening each package, reading its metadata, and then
	// writing it to the index file.

	indexSource, err := os.Open(indexSourceFilename)
	if err != nil {
		return fmt.Errorf("Error opening index source file: %v", err)
	}
	defer indexSource.Close()

	// The index file is a json file. At the top level we have `version` and `packages`.
	// `version` is the version of the index file format.
	// `packages` is a map of package names to package metadata.
	// Each package metadata is a map of metadata fields to values.
	// The metadata fields are `name`, `version`, `location`, `conflicts`, and `dependencies`.

	packages := make(map[string]Package)
	indexSourceScanner := bufio.NewScanner(indexSource)
	for indexSourceScanner.Scan() {
		packageLocation := indexSourceScanner.Text()
		var localLocation string

		// packageLocation can be remote: http:// or https:// mean that we need to download the package.
		if locationIsRemote(packageLocation) {
			localLocation, err = context.SourceControl.DownloadReleaseFile(packageLocation)
			if err != nil {
				return fmt.Errorf("Error downloading package %s: %v\n", packageLocation, err)
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
			return fmt.Errorf("Package file missing: %s", localLocation)
		}
		pkg := readIndexEntry(context.PackageFiles, localLocation)
		if pkg.Name == "" {
			// This is a proper "this should never happen" - if we can't get the name of the package
			// then something has gone wrong upstream.
			panic(fmt.Sprintf("Package name in %s is empty.", localLocation))
		}
		pkg.Location = packageLocation
		packages[pkg.Name] = pkg
	}

	// Now for each package, we need to check that the conflicts are reflected both ways.
	// We do this by iterating over the conflicts, and then adding the package to the conflicts
	// field of the conflicting package.
	for _, pkg := range packages {
		for _, conflict := range pkg.Conflicts {
			if _, ok := packages[conflict]; ok {
				if !stringInSlice(pkg.Name, packages[conflict].Conflicts) {
					// Can't assign to pkg.Conflicts, so we have to make a new one.
					packages[conflict] = Package{
						Name:         packages[conflict].Name,
						Version:      packages[conflict].Version,
						Location:     packages[conflict].Location,
						Dependencies: packages[conflict].Dependencies,
						Conflicts:    append(packages[conflict].Conflicts, pkg.Name),
					}
				}
			} else {
				// We don't have the named package in the index. The most dangerous case is that
				// it's a typo and someone will install incompatible packages, so we need to barf.
				return fmt.Errorf("Package %s conflicts with %s, but %s is not in the index.\n",
					pkg.Name, conflict, conflict)
			}
		}
	}

	indexFile, err := os.Create(indexFilename)
	if err != nil {
		return fmt.Errorf("Error creating index file: %v", err)
	}
	defer indexFile.Close()

	index := Index{}
	index.Version = "1"
	index.Packages = packages

	json.NewEncoder(indexFile).Encode(index)
	return nil
}

func PackageInit(context *Context, pkgname string) error {
	return context.PackageFiles.Init(pkgname)
}

func PackageBuild(context *Context, pkgname string, outdir string, version string) (string, error) {
	return context.PackageFiles.Build(pkgname, outdir, version)
}

// coyote package new <pkgname>
// This creates a new package in the current directory, and pushes it
// to github.
// At some point we will undoubtedly want to have a template for this,
// but for now we just create the files.
func PackageNew(context *Context, pkgname string) error {
	// First we make the new dir.  The name of the dir will match our
	// remote.  We need to check that the name is available first.

	sourceControl := context.SourceControl
	intendedName := "cypkg-" + pkgname
	packageOrg := context.Config.GetPackageOrg()
	available, err := sourceControl.IsNameAvailable(intendedName, packageOrg)
	if err != nil {
		return fmt.Errorf("Error checking name availability: %v", err)
	}
	if !available {
		return fmt.Errorf("Name %s is already taken.", intendedName)
	}

	actualName := intendedName

	// Now we create the local dir, and initialise it as a git repo.
	os.MkdirAll(actualName, 0777)
	cwd := os.Getenv("PWD")
	err = os.Chdir(actualName)
	if err != nil {
		return fmt.Errorf("Error changing to new directory: %v", err)
	}
	defer os.Chdir(cwd)

	err = exec.Command("git", "init").Run()
	if err != nil {
		return fmt.Errorf("Error initialising git repo: %v", err)
	}
	// Force the main branch to be called main.
	err = exec.Command("git", "branch", "-M", "main").Run()
	if err != nil {
		return fmt.Errorf("Error setting main branch: %v", err)
	}
	// Now we can run Init to create the package files.
	context.PackageFiles.Init(pkgname)
	// git add, git commit...
	err = exec.Command("git", "add", ".").Run()
	if err != nil {
		return fmt.Errorf("Error adding files to git repo: %v", err)
	}
	err = exec.Command("git", "commit", "-m", "Initial commit").Run()
	if err != nil {
		return fmt.Errorf("Error committing files to git repo: %v", err)
	}
	// Now we can create the remote repo.
	err = sourceControl.CreateRepo(actualName, packageOrg)
	if err != nil {
		return fmt.Errorf("Error creating remote repo: %v", err)
	}
	// We need to loop here until the remote repo is actually created, which
	// we check by seeing if the name is available
	for {
		available, err = sourceControl.IsNameAvailable(actualName, packageOrg)
		if err != nil {
			return fmt.Errorf("Error checking name availability: %v", err)
		}
		if !available {
			break
		} else {
			time.Sleep(time.Duration(sourceControl.GetRateLimitDelayMilliseconds()))
		}
	}

	return context.SourceControl.Push(actualName, packageOrg)
}

func PackageDelete(context *Context, pkgname string) error {
	// This function deletes the named package from github.
	// It does not delete the local copy of the package.
	// It does not check that the package is not in use.
	// It does not check that the package is not a dependency of another package.
	// It does not check that the package is not a dependency of the project.
	// It will make you sad if you use it wrong.

	sourceControl := context.SourceControl
	packageOrg := context.Config.GetPackageOrg()
	err := sourceControl.DeleteRepo("cypkg-"+pkgname, packageOrg)
	if err != nil {
		return fmt.Errorf("Error deleting remote repo: %v", err)
	}
	return nil
}

func repoHasOriginSet(origin string) (bool, error) {
	remotes, err := exec.Command("git", "remote").Output()
	if err != nil {
		return false, fmt.Errorf("Error getting remote list: %v", err)
	}
	return strings.Contains(string(remotes)+"\n", origin), nil
}

func Open(context *Context) error {
	// If we're in a github repo, open the origin remote repo in the browser.
	remoteToOpen := "origin"
	remoteExists, err := repoHasOriginSet(remoteToOpen)
	if err != nil {
		return fmt.Errorf("Error checking for remote: %v", err)
	} else if !remoteExists {
		return fmt.Errorf("No %s remote found.", remoteToOpen)
	}

	remote, err := exec.Command("git", "remote", "get-url", remoteToOpen).Output()
	if err != nil {
		return fmt.Errorf("Error getting remote url: %v", err)
	}

	platform := context.Platform
	return platform.OpenURL(strings.TrimSpace(string(remote)))
}

func pushTagsToOrigin() error {
	// TODO this is potentially hazardous, because it pushes all tags and ignores whether
	// what's checked out matches the version we're pushing.  Ok for a demo though.
	return exec.Command("git", "push", "origin", "--follow-tags").Run()
}

func PackageRelease(context *Context, pkgname string, version string) (string, error) {
	//Barf if we're not in a coyote package
	if _, err := os.Stat(".cypkg"); os.IsNotExist(err) {
		return "", fmt.Errorf("Not in a Coyote package.\n")
	}

	// Bad things will happen if we get version=="HEAD" here, so don't do that
	if version == "HEAD" {
		return "", fmt.Errorf("Cannot release HEAD version.")
	}

	tag, err := tagForRelease(version, context, pkgname)
	if err != nil {
		return "", err
	}

	// We know the tag exists in the repo, so we can now build the tag.

	packagePath, err := context.PackageFiles.Build(pkgname, ".", version)
	if err != nil {
		return "", fmt.Errorf("Error building package: %v\n", err)
	}

	err = pushTagsToOrigin()
	if err != nil {
		return "", fmt.Errorf("Error pushing tag to remote: %v\n", err)
	}

	return releaseFiles(context, "cypkg-"+pkgname, tag, []string{packagePath})
}

// This function does all the preflight checks to ensure that the version tag we're
// asking for exists in the repository, and hasn't been released already.  This includes
// tagging locally.
// It returns the tag that was actually written to the repo.
func tagForRelease(version string, context *Context, pkgname string) (string, error) {
	tag := "coyote-" + version

	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return "", fmt.Errorf("Not in a git repository.\n")
	}

	remoteToOpen := "origin"
	remoteExists, err := repoHasOriginSet(remoteToOpen)
	if err != nil {
		return "", fmt.Errorf("Error checking for remote: %v", err)
	} else if !remoteExists {
		return "", fmt.Errorf("No %s remote found.", remoteToOpen)
	}

	err = exec.Command("git", "check-ref-format", "--allow-onelevel", tag).Run()
	if err != nil {
		return "", fmt.Errorf("Invalid version: %v\n", err)
	}

	sourceControl := context.SourceControl
	packageOrg := context.Config.GetPackageOrg()

	releaseExists, err := sourceControl.DoesReleaseExist(pkgname, packageOrg, version)
	if err != nil {
		return "", fmt.Errorf("Error checking if release exists: %v\n", err)
	}
	if releaseExists {
		return "", fmt.Errorf("Release %s already exists.\n", version)
	}

	output, err := exec.Command("git", "tag", "--list", tag).Output()
	if err != nil {
		return "", fmt.Errorf("Error checking for existing tag: %v\n", err)
	}
	if strings.TrimSpace(string(output)) != tag {
		// We use an annotated tag here so that we keep authorship
		// TODO: check whether tag signing can work here
		// TODO: what can usefully go in the tag message?
		cmd := exec.Command("git", "tag", "--annotate", "-m", "No tag message", tag)
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating tag %v:\n%v\n", tag, output)
			return "", fmt.Errorf("Error creating tag %v: %v\n", tag, err)
		}
	}
	return tag, nil
}

func PackageTest(context *Context, pkgname string) error {
	//All this test does is make a temp dir, build the package into it, and then apply it.
	// It will barf if there are any errors in the templates.
	tempdir, err := os.MkdirTemp("", "coyote-test")
	if err != nil {
		return fmt.Errorf("Error creating temp dir: %v\n", err)
	}
	defer os.RemoveAll(tempdir)

	packagePath, err := context.PackageFiles.Build(pkgname, tempdir, "HEAD")
	if err != nil {
		return fmt.Errorf("Error building package: %v\n", err)
	}

	// Now we apply the package to the temp dir.
	// We need to change to the temp dir first, and then change back.
	cwd := os.Getenv("PWD")
	err = os.Chdir(tempdir)
	if err != nil {
		return fmt.Errorf("Error changing to temp dir: %v\n", err)
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
		return "", fmt.Errorf("Index source file not found: %v", indexSrcInput)
	}

	tag, err := tagForRelease(versionToReleaseAs, context, repoName)
	if err != nil {
		return "", err
	}

	indexFile, err := os.CreateTemp("", "coyote")
	if err != nil {
		return "", fmt.Errorf("Error creating temp file: %v", err)
	}
	defer os.Remove(indexFile.Name())

	err = BuildIndex(context, indexSrcInput, indexFile.Name())
	if err != nil {
		return "", fmt.Errorf("Error building index: %v", err)
	}

	return releaseFiles(context, repoName, tag, []string{indexFile.Name()})
}

func releaseFiles(context *Context, repoName string, tag string, filesToRelease []string) (string, error) {
	sourceControl := context.SourceControl
	indexOrg := context.Config.GetPackageOrg()

	assetURLs, err := sourceControl.CreateRelease(repoName, indexOrg, tag, filesToRelease)
	if err != nil {
		return "", fmt.Errorf("Error creating release: %v", err)
	}

	return assetURLs[0], nil
}
