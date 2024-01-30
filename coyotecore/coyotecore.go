package coyotecore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
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

func projectReadProjectName() string {
	pkgname, err := os.ReadFile(".coyote/project-name")
	if err != nil {
		panic(err)
	}
	return string(pkgname)
}

func extractPackage(packageFiles IProvidePackageFiles, filename string) PackageFile {
	// This function extracts the package to the project. We record the installation
	// in .coyote/installed.

	var vars PackageTemplateVars
	vars.ProjectName = projectReadProjectName()
	pkg := packageFiles.Open(filename)

	pkg.Apply(vars)

	recordInstalledPackage(pkg.ReadMetadata("NAME"), pkg.ReadMetadata("VERSION"))

	return pkg
}

func recordInstalledPackage(packageName string, version string) {
	installedFilename := ".coyote/installed"

	file, err := os.OpenFile(installedFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write([]byte(packageName + "=" + version + "\n"))
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

func Apply(context *Context, filename string) {
	checkForCoyoteProject()
	pkg := context.PackageFiles.Open(filename)
	conflicts := conflictingInstalledPackages(pkg)
	if len(conflicts) == 0 {
		pkg := extractPackage(context.PackageFiles, filename)
		err := runOnInstall(pkg)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running on-install script: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Conflicts found: %s\n", strings.Join(conflicts, ", "))
		os.Exit(1)
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

func Init(context *Context, techStack string, projectName string, index string) error {
	// This function creates a new Coyote project, named projectName.
	// The project will be created in the current directory.
	// The name will be stored in .coyote/project-name.

	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("Project %s already exists.\n", projectName)
	}
	// Now we make the project directory and store the name in .coyote/project-name.
	os.MkdirAll(projectName+"/.coyote", 0777)
	os.WriteFile(projectName+"/.coyote/project-name", []byte(projectName), 0777)

	if techStack != "empty" {

		indexFile, err := openIndexFile(index)
		if err != nil {
			return fmt.Errorf("Error opening index file: %v\n", err)
		}
		//Now we extract the package to the project directory. After this, index is
		//no longer valid, use indexFile.filename instead.
		os.Chdir(projectName)
		defer os.Chdir("..")

		return installPackageTree(context.PackageFiles, techStack, indexFile, false)
	} else {
		return nil
	}
}

func installPackageTree(packageFiles IProvidePackageFiles, pkg string, indexFile IndexFile, reinstall bool) error {
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
			location, err = downloadFile(location)
			if err != nil {
				return fmt.Errorf("Error downloading package: %v\n", err)
			}
			defer os.Remove(location)
		}

		if _, err := os.Stat(location); err != nil {
			return fmt.Errorf("Package file missing: %v\n", err)
		}

		pkg := extractPackage(packageFiles, location)
		err = runOnInstall(pkg)

		if err != nil {
			return fmt.Errorf("Error running on-install script: %v\n", err)
		}
	}
	return nil
}

func Install(context *Context, pkgname string, index string, reinstall bool) error {
	localIndex := index
	if locationIsRemote(index) {
		myLocalIndex, err := downloadFile(index)
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

	return installPackageTree(context.PackageFiles, pkgname, indexFile, reinstall)
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

func downloadFile(location string) (string, error) {
	// This function downloads a file from a remote location, and returns the local filename.
	// It returns an error if the download fails.
	// The file is downloaded to /tmp, and the filename is returned.
	// Just use wget for now.
	// The local filename is the same as the remote filename, but because we might have query strings or a fragment suffix in the url
	// we need to strip them off.
	urlWithoutFragment := strings.Split(location, "#")[0]
	parsedUrl, err := url.Parse(urlWithoutFragment)
	if err != nil {
		return "", fmt.Errorf("Error parsing url %s: %v", location, err)
	}
	filename := parsedUrl.Path
	basename := strings.Split(filename, "/")[len(strings.Split(filename, "/"))-1]
	if basename == "" {
		return "", fmt.Errorf("Error parsing filename from url %s", location)
	}

	cmd := exec.Command("wget", "-O", "/tmp/"+basename, location)
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error downloading file from %s: %v", location, err)
	}
	return "/tmp/" + basename, nil
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
			localLocation, err = downloadFile(packageLocation)
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
		pkg := readIndexEntry(context.PackageFiles, localLocation)
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

func PackageInit(context *Context, pkgname string) {
	context.PackageFiles.Init(pkgname)
}

func PackageBuild(context *Context, pkgname string, outdir string) {
	context.PackageFiles.Build(pkgname, outdir)
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

	// We set the remote origin to the remote repo.
	remoteUrl := "https://github.com/" + packageOrg + "/" + actualName + ".git"
	err = exec.Command("git", "remote", "add", "origin", remoteUrl).Run()
	if err != nil {
		return fmt.Errorf("Error adding remote to git repo: %v", err)
	}
	// Now we can push the local repo to the remote.
	err = exec.Command("git", "push", "-u", "origin", "HEAD").Run()
	if err != nil {
		return fmt.Errorf("Error pushing to remote repo: %v", err)
	}
	return nil
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

func Open(context *Context) error {
	// If we're in a github repo, open the origin remote repo in the browser.
	remoteToOpen := "origin"
	remotes, err := exec.Command("git", "remote").Output()
	if err != nil {
		return fmt.Errorf("Error getting remote list: %v", err)
	}

	if !strings.Contains(string(remotes)+"\n", remoteToOpen) {
		return fmt.Errorf("No %s remote found.", remoteToOpen)
	}

	remote, err := exec.Command("git", "remote", "get-url", remoteToOpen).Output()
	if err != nil {
		return fmt.Errorf("Error getting remote url: %v", err)
	}

	platform := context.Platform
	return platform.OpenURL(strings.TrimSpace(string(remote)))
}
