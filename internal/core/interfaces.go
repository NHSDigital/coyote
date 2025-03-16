package core

type Config interface {
	// The index key is the path or URL to the index file.
	GetIndex() string

	// The github org where we publish packages
	GetPackageOrg() string

	// The path key returns the original path to the config file.
	GetPath() string
}

type NullConfig struct{}

func (c *NullConfig) GetIndex() string {
	return ""
}

func (c *NullConfig) GetPath() string {
	return ""
}

func (c *NullConfig) GetPackageOrg() string {
	return ""
}

type PackageFile interface {
	ReadMetadata(field string) string
	Apply(vars PackageTemplateVars)
}

type Project interface {
	Init(name string) error
	GetPath() string
	GetName() string
	RecordInstalledPackage(pkg PackageFile) error
	ReadInstalledPackages() ([][]string, error)
}

type IProvideProjects interface {
	MaybeProject(path string) Project
	NewProject(path string, name string) Project
}

type IProvidePackageFiles interface {
	/*
		Set up the current directory as a package repository, with one package
		defined in the metadata.
	*/
	Init(pkgname string) error

	/*
		Build the named package in the current package directory, and write the
		result to the output directory.  The version can be "HEAD", "",
		or a specific version (without any `coyote-` prefix).
	*/
	Build(pkgname string, outdir string, version string) (string, error)

	/*
		Return a PackageFile, which can be used to read metadata or to apply it
		to a project.
	*/
	Open(location string) PackageFile

	/*
		Return the version string of the package that would be built according
		to the tags in the current package directory.
	*/
	Version() (string, error)
}

type IProvideSourceControl interface {
	IsNameAvailable(repo string, org string) (bool, error)
	CreateRepo(repo string, org string) error
	DeleteRepo(repo string, org string) error
	CreateRelease(repo string, org string, tag string, filenames []string) ([]string, error)
	DeleteRelease(repo string, org string, tag string) error
	GetRateLimitDelayMilliseconds() int
	DoesReleaseExist(repo string, org string, tag string) (bool, error)
	DownloadReleaseFile(href string) (string, error)
	GetRemoteURL(repo string, org string) string
	Push(repo string, org string) error
}

type Context struct {
	Config        Config
	PackageFiles  IProvidePackageFiles
	SourceControl IProvideSourceControl
	Platform      Platform
	Projects      IProvideProjects
	IndexFiles    IProvideIndexFiles
}

type PackageTemplateVars struct {
	ProjectName string
}

type Platform interface {
	OpenURL(url string) error
}

type IndexFile interface {
	GetPackage(pkgName string) (PackageIndexEntry, error)
	Describe() string
}

type IProvideIndexFiles interface {
	OpenIndexFile(context *Context, filename string) (IndexFile, error)
}
