package core

type Index struct {
	indexFile IndexFile
}

type PackageIndexEntry struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Location     string   `json:"location"`
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
}

type IndexData struct {
	Version  string                       `json:"version"`
	Packages map[string]PackageIndexEntry `json:"packages"`
}

func (index Index) ReadPackageLocation(pkgName string) (string, error) {
	pkg, err := index.indexFile.GetPackage(pkgName)
	if err != nil {
		return "", err
	}
	return pkg.Location, nil
}

func (index Index) ReadPackageDependencies(pkgName string) ([]string, error) {
	pkg, err := index.indexFile.GetPackage(pkgName)
	if err != nil {
		return []string{}, err
	}
	return pkg.Dependencies, nil
}

func (index Index) ReadPackageConflicts(pkgName string) ([]string, error) {
	pkg, err := index.indexFile.GetPackage(pkgName)
	if err != nil {
		return []string{}, err
	}
	return pkg.Conflicts, nil
}

func (index Index) Describe() string {
	return index.indexFile.Describe()
}

func openIndex(context *Context, filename string) (Index, error) {
	indexFile, err := context.IndexFiles.OpenIndexFile(context, filename)
	if err != nil {
		return Index{}, err
	}
	return Index{indexFile: indexFile}, nil
}
