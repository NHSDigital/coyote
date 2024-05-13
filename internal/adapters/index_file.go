/* Implementations of IndexFile for both local and http indexes */

package adapters

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	core "github.com/nhsdigital/coyote/internal/core"
)

type LocalIndexFile struct {
	filename string
	contents core.IndexData
}

func (indexFile LocalIndexFile) GetPackage(pkgName string) (core.PackageIndexEntry, error) {
	if pkg, ok := indexFile.contents.Packages[pkgName]; ok {
		return pkg, nil
	}
	return core.PackageIndexEntry{}, fmt.Errorf("package %s not found in index file %s", pkgName, indexFile.filename)
}

func (indexFile LocalIndexFile) Describe() string {
	return indexFile.filename
}

func openLocalIndexFile(filename string) (LocalIndexFile, error) {
	st, err := os.Stat(filename)
	if err != nil {
		return LocalIndexFile{}, fmt.Errorf("index file %s does not exist", filename)
	}
	if st.Mode().IsDir() {
		return LocalIndexFile{}, fmt.Errorf("index file %s is a directory, not a file", filename)
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

	var indexData core.IndexData
	err = json.Unmarshal(indexBytes, &indexData)
	if err != nil {
		return LocalIndexFile{}, fmt.Errorf("error parsing index file %s: %v", filename, err)
	}

	//Use the absolute path to the index file so we can use it if we change directories.
	absFilename, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	absFilename += "/" + filename

	return LocalIndexFile{filename: absFilename, contents: indexData}, nil
}

type GithubIndexFile struct {
	href           string
	localIndexFile LocalIndexFile
}

func openGithubIndexFile(context *core.Context, href string) (core.IndexFile, error) {
	localPath, err := context.SourceControl.DownloadReleaseFile(href)
	if err != nil {
		return nil, fmt.Errorf("error downloading index file: %v", err)
	} else if localPath == "" {
		return nil, fmt.Errorf("error downloading index file: no file returned")
	}
	defer os.Remove(localPath)

	localIndexFile, err := openLocalIndexFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("error opening index file: %v", err)
	}

	return GithubIndexFile{href: href, localIndexFile: localIndexFile}, nil
}

func (indexFile GithubIndexFile) GetPackage(pkgName string) (core.PackageIndexEntry, error) {
	return indexFile.localIndexFile.GetPackage(pkgName)
}

func (indexFile GithubIndexFile) Describe() string {
	return indexFile.href
}

func locationIsRemote(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

func OpenIndexFile(context *core.Context, filename string) (core.IndexFile, error) {
	if locationIsRemote(filename) {
		return openGithubIndexFile(context, filename)
	} else {
		return openLocalIndexFile(filename)
	}
}

type IndexFileProvider struct{}

func (p IndexFileProvider) OpenIndexFile(context *core.Context, filename string) (core.IndexFile, error) {
	return OpenIndexFile(context, filename)
}

func NewIndexFileProvider() IndexFileProvider {
	return IndexFileProvider{}
}
