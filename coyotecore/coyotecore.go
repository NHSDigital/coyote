package coyotecore

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"text/template"
)

func PackageInit(pkgname string) {
	os.Mkdir(".cypkg", 0777)
	os.Mkdir(".cypkg/"+pkgname, 0777)
	os.WriteFile(".cypkg/"+pkgname+"/DEPENDS",
		[]byte("# List package dependencies here, one per line."),
		0777)
	os.WriteFile(".cypkg/"+pkgname+"/CONFLICTS",
		[]byte("# List package conflicts here, one per line."),
		0777)
}

func CopyFile(src, dst string) {
	srcFile, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		panic(err)
	}
}

func CopyFileIfExist(src, dst string) {
	if _, err := os.Stat(src); err == nil {
		CopyFile(src, dst)
	}
}

func versionFromTags() string {
	cmd := exec.Command("git", "tag", "--list", "coyote-*")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	versions := strings.Split(string(output), "\n")
	sort.Strings(versions)

	version := versions[len(versions)-1]
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "coyote-")
	return version
}

func ReadMetadata(pkgFilename string, field string) string {
	fileCheck := exec.Command("tar", "-tf", pkgFilename, ".CYMETA/"+field)
	if err := fileCheck.Run(); err != nil {
		return ""
	} else {
		cmd := exec.Command("tar", "-xOf", pkgFilename, ".CYMETA/"+field)
		output, err := cmd.Output()
		if err != nil {
			panic(err)
		}
		return strings.TrimSpace(string(output))
	}
}

func PackageBuild(pkgname string) {
	version := versionFromTags()
	filename := pkgname + "-" + version + ".cypkg"

	tempDir, err := os.MkdirTemp("", "coyote")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	err = exec.Command("git", "clone", ".", tempDir).Run()
	if err != nil {
		panic(err)
	}
	os.RemoveAll(tempDir + "/.git")
	os.RemoveAll(tempDir + "/.cypkg")

	os.Mkdir(tempDir+"/.CYMETA", 0777)
	CopyFile(".cypkg/"+pkgname+"/DEPENDS", tempDir+"/.CYMETA/DEPENDS")
	CopyFile(".cypkg/"+pkgname+"/CONFLICTS", tempDir+"/.CYMETA/CONFLICTS")
	os.WriteFile(tempDir+"/.CYMETA/VERSION", []byte(version), 0777)
	os.WriteFile(tempDir+"/.CYMETA/NAME", []byte(pkgname), 0777)
	CopyFileIfExist(".cypkg/"+pkgname+"/on-install", tempDir+"/.CYMETA/on-install")

	os.Mkdir(".cypkg/tmp", 0777)
	exec.Command("tar", "-czf", ".cypkg/tmp/"+filename, "-C", tempDir, ".").Run()

	os.Rename(".cypkg/tmp/"+filename, filename)
	fmt.Println(filename)
}

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

func conflictsFound(filename string) []string {
	result := []string{}
	installeds, err := readInstalledPackages()
	if err != nil {
		panic(err)
	}

	if len(installeds) == 0 {
		return result
	}

	conflicts := ReadMetadata(filename, "CONFLICTS")
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

type ProjectTemplateVars struct {
	ProjectName string
}

func projectReadProjectName() string {
	pkgname, err := os.ReadFile(".coyote/project-name")
	if err != nil {
		panic(err)
	}
	return string(pkgname)
}

func extractPackage(filename string) {
	// This function extracts the package to the current directory, and templates each with text/template.
	// We don't extract files in .CYMETA, that's just for coyote to use, but we do record the installation
	// in .coyote/installed.

	// For each file in the package, we extract it as a string, and then template it.
	// We then write the templated string to the file.

	var vars ProjectTemplateVars
	vars.ProjectName = projectReadProjectName()

	tarFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer tarFile.Close()
	gzipReader, err := gzip.NewReader(tarFile)
	if err != nil {
		panic(err)
	}
	defer gzipReader.Close()

	files := tar.NewReader(gzipReader)
	for {
		header, err := files.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		if strings.HasPrefix(header.Name, "./.CYMETA/") {
			continue
		}

		extractFile(header, vars, files)
	}

	appendInstalledPackage(ReadMetadata(filename, "NAME"), ReadMetadata(filename, "VERSION"))
}

func extractFile(header *tar.Header, vars ProjectTemplateVars, file *tar.Reader) {
	templatedFilename := templateString(header.Name, vars)
	mode := header.FileInfo().Mode()

	if header.Typeflag == tar.TypeDir {
		os.MkdirAll(templatedFilename, mode)
	} else if header.Typeflag == tar.TypeSymlink {
		target := header.Linkname
		templatedTarget := templateString(target, vars)

		os.Symlink(templatedTarget, templatedFilename)
	} else {
		contents, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}
		templatedContents := templateString(string(contents), vars)

		os.WriteFile(templatedFilename, []byte(templatedContents), mode)
	}
}

func templateString(contents string, vars ProjectTemplateVars) string {
	tmpl := template.Must(template.New("file").Parse(contents))
	var templated bytes.Buffer
	err := tmpl.Execute(&templated, vars)
	if err != nil {
		panic(err)
	}
	return templated.String()
}

func appendInstalledPackage(packageName string, version string) {
	installedFilename := ".coyote/installed"

	file, err := os.OpenFile(installedFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write([]byte(packageName + "=" + version + "\n"))
}

func runOnInstall(filename string) error {
	onInstall := ReadMetadata(filename, "on-install")
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

func Apply(filename string) {
	checkForCoyoteProject()
	conflicts := conflictsFound(filename)
	if len(conflicts) == 0 {
		extractPackage(filename)
		err := runOnInstall(filename)

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

func OpenIndexFile(filename string) (IndexFile, error) {

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

func Init(techStack string, projectName string, index string) error {
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

		indexFile, err := OpenIndexFile(index)
		if err != nil {
			return fmt.Errorf("Error opening index file: %v\n", err)
		}
		//Now we extract the package to the project directory. After this, index is
		//no longer valid, use indexFile.filename instead.
		os.Chdir(projectName)
		defer os.Chdir("..")

		return installPackageTree(techStack, indexFile, false)
	} else {
		return nil
	}
}

func installPackageTree(pkg string, indexFile IndexFile, reinstall bool) error {
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
			location, err = DownloadFile(location)
			if err != nil {
				return fmt.Errorf("Error downloading package: %v\n", err)
			}
			defer os.Remove(location)
		}

		if _, err := os.Stat(location); err != nil {
			return fmt.Errorf("Package file missing: %v\n", err)
		}

		extractPackage(location)
		err = runOnInstall(location)

		if err != nil {
			return fmt.Errorf("Error running on-install script: %v\n", err)
		}
	}
	return nil
}

func Install(pkgname string, index string, reinstall bool) error {
	localIndex := index
	if locationIsRemote(index) {
		myLocalIndex, err := DownloadFile(index)
		localIndex = myLocalIndex
		if err != nil {
			return fmt.Errorf("Error downloading index file: %v\n", err)
		}
		defer os.Remove(localIndex)
	}

	indexFile, err := OpenIndexFile(localIndex)
	if err != nil {
		return fmt.Errorf("Error opening index file: %v\n", err)
	}

	return installPackageTree(pkgname, indexFile, reinstall)
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

func DownloadFile(location string) (string, error) {
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

func BuildIndex(indexSourceFilename string, indexFilename string) error {
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
			localLocation, err = DownloadFile(packageLocation)
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
		pkg := readIndexEntry(localLocation)
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

func readIndexEntry(packageLocation string) Package {
	packageFile, err := os.Open(packageLocation)
	if err != nil {
		panic(err)
	}
	defer packageFile.Close()

	pkg := Package{}

	pkg.Name = ReadMetadata(packageLocation, "NAME")
	pkg.Version = ReadMetadata(packageLocation, "VERSION")
	pkg.Conflicts = removeComments(ReadMetadata(packageLocation, "CONFLICTS"))
	pkg.Dependencies = removeComments(ReadMetadata(packageLocation, "DEPENDS"))
	return pkg
}
