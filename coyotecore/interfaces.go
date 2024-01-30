package coyotecore

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

type IProvidePackageFiles interface {
	Init(pkgname string)
	Build(pkgname string, outdir string, version string) (string, error)
	Open(location string) PackageFile
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
}

type Context struct {
	Config        Config
	PackageFiles  IProvidePackageFiles
	SourceControl IProvideSourceControl
	Platform      Platform
}

type PackageTemplateVars struct {
	ProjectName string
}

type Platform interface {
	OpenURL(url string) error
}
