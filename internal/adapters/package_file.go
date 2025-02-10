package adapters

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/template"

	core "github.com/nhsdigital/coyote/internal/core"
)

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

type PackageTarFile struct {
	Filename string
}

type PackageTarFileProvider struct{}

func NewPackageTarFileProvider() PackageTarFileProvider {
	return PackageTarFileProvider{}
}

func NewPackageFile(filename string) PackageTarFile {
	return PackageTarFile{Filename: filename}
}

func (p PackageTarFileProvider) Open(location string) core.PackageFile {
	return NewPackageFile(location)
}

func (p PackageTarFile) ReadMetadata(field string) string {
	fileCheck := exec.Command("tar", "-tf", p.Filename, ".CYMETA/"+field)
	if err := fileCheck.Run(); err != nil {
		return ""
	} else {
		cmd := exec.Command("tar", "-xOf", p.Filename, ".CYMETA/"+field)
		output, err := cmd.Output()
		if err != nil {
			panic(err)
		}
		return strings.TrimSpace(string(output))
	}
}

func templateString(contents string, vars core.PackageTemplateVars, label string) string {
	tmpl := template.Must(template.New(label).Parse(contents))
	var templated bytes.Buffer
	err := tmpl.Execute(&templated, vars)
	if err != nil {
		panic(err)
	}
	return templated.String()
}

func PackageInit(pkgname string) error {
	err := os.Mkdir(".cypkg", 0777)
	if err != nil {
		return fmt.Errorf("error creating .cypkg directory: %v", err)
	}

	err = os.Mkdir(".cypkg/"+pkgname, 0777)
	if err != nil {
		return fmt.Errorf("error creating .cypkg/%v directory: %v", pkgname, err)
	}

	err = os.WriteFile(".cypkg/"+pkgname+"/DEPENDS",
		[]byte("# List package dependencies here, one per line."),
		0777)
	if err != nil {
		return fmt.Errorf("error creating .cypkg/%v/DEPENDS: %v", pkgname, err)
	}

	err = os.WriteFile(".cypkg/"+pkgname+"/CONFLICTS",
		[]byte("# List package conflicts here, one per line."),
		0777)
	if err != nil {
		return fmt.Errorf("error creating .cypkg/%v/CONFLICTS: %v", pkgname, err)
	}
	return nil
}

func (p PackageTarFileProvider) Init(pkgname string) error {
	return PackageInit(pkgname)
}

// This function defines the versioning scheme for coyote packages.
// Any git tag that starts with "coyote-" is considered a coyote package version.
// Versions are sorted alphabetically. No other format is imposed.
// TODO: Is this actually a good idea? I find in trying out coyote that I want
// to specify any old git ref as a version, not just tags.
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

func tagFromVersion(version string) string {
	return "coyote-" + strings.TrimSpace(version)
}

func PackageBuild(pkgname string, outdir string, version string) (string, error) {
	tempDir, err := os.MkdirTemp("", "coyote")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var cmd *exec.Cmd
	// We support the magic version "HEAD" which means the latest commit, and is only useful for testing
	// This is the only way to build a package from a commit that isn't tagged
	if version == "HEAD" {
		cmd = exec.Command("git", "clone", ".", tempDir)
	} else {
		// If no version is specified, we build the latest version that's been tagged
		if version == "" {
			version = versionFromTags()
		}
		rev := tagFromVersion(version)
		cmd = exec.Command("git", "clone", "--branch", rev, ".", tempDir)
	}

	_, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running clone command `%v` %v: %v", cmd, version, err)
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

	filename := pkgname + "-" + version + ".cypkg"
	exec.Command("tar", "-czf", ".cypkg/tmp/"+filename, "-C", tempDir, ".").Run()

	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		os.MkdirAll(outdir, 0777)
	} else if err != nil {
		return "", fmt.Errorf("error making the output dir: %v", err)
	}

	outfile := outdir + "/" + filename

	err = os.Rename(".cypkg/tmp/"+filename, outfile)
	if err != nil {
		return "", fmt.Errorf("error moving the output file: %v", err)
	}

	return outfile, nil
}

func (p PackageTarFileProvider) Build(pkgname string, outdir string, version string) (string, error) {
	return PackageBuild(pkgname, outdir, version)
}

func extractFile(header *tar.Header, vars core.PackageTemplateVars, file *tar.Reader) {
	templatedFilename := templateString(header.Name, vars, "Filename "+header.Name)
	mode := header.FileInfo().Mode()

	if header.Typeflag == tar.TypeDir {
		os.MkdirAll(templatedFilename, mode)
	} else if header.Typeflag == tar.TypeSymlink {
		target := header.Linkname
		templatedTarget := templateString(target, vars, "Symlink "+header.Name+" -> "+target)

		os.Symlink(templatedTarget, templatedFilename)
	} else {
		contents, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}
		// TODO: we shouldn't template binary files, but I don't have a good answer for
		// how we should demarcate them yet
		templatedContents := templateString(string(contents), vars, "Contents "+header.Name)

		os.WriteFile(templatedFilename, []byte(templatedContents), mode)
	}
}

func (p PackageTarFile) Apply(vars core.PackageTemplateVars) {
	tarFile, err := os.Open(p.Filename)
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
}
