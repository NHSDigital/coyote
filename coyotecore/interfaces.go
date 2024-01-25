package coyotecore

type Config interface {
	// The index key is the path or URL to the index file.
	GetIndex() string
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

type PackageFile interface {
	ReadMetadata(field string) string
	Apply(vars PackageTemplateVars)
}

type IProvidePackageFiles interface {
	Init(pkgname string)
	Build(pkgname string, outdir string)
	Open(location string) PackageFile
}

type Context struct {
	Config       Config
	PackageFiles IProvidePackageFiles
}

type PackageTemplateVars struct {
	ProjectName string
}

type IProvideSourceControl interface {
	IsNameAvailable(repo string, org string) (bool, error)
	CreateRepo(repo string, org string) error
	DeleteRepo(repo string, org string) error
	CreateRelease(repo string, org string, tag string, filenames []string) ([]string, error)
	DeleteRelease(repo string, org string, tag string) error
}
