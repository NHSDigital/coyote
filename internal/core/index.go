package core

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Index struct {
	indexFile IndexFile
}

// PackageVersionEntry represents a specific version of a package
type PackageVersionEntry struct {
	Version      string   `json:"version"`
	Location     string   `json:"location"`
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
}

// PackageIndexEntry represents a package in the index, potentially with multiple versions
type PackageIndexEntry struct {
	Name     string                `json:"name"`
	Version  string                `json:"version"`  // Latest version for backwards compatibility
	Location string                `json:"location"` // Latest location for backwards compatibility
	Versions []PackageVersionEntry `json:"versions,omitempty"`
	// Keep these for backwards compatibility when there's only one version
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
}

type IndexData struct {
	Version  string                       `json:"version"`
	Packages map[string]PackageIndexEntry `json:"packages"`
}

// parseSemanticVersion parses a version string like "v1.2.3" or "1.2.3" into components
// Returns major, minor, patch as integers, and remainder string for any suffix
func parseSemanticVersion(version string) (major, minor, patch int, remainder string) {
	// Strip leading 'v' or 'V' if present
	v := strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")

	// Match semantic version pattern: major.minor.patch with optional remainder
	re := regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?(.*)$`)
	matches := re.FindStringSubmatch(v)

	if matches == nil {
		// Not a valid semver, return zeros and the original string
		return 0, 0, 0, version
	}

	major, _ = strconv.Atoi(matches[1])
	if matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	if matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}
	remainder = matches[4]

	return
}

// CompareSemanticVersions compares two version strings using semantic versioning rules.
// Returns:
//
//	1 if a > b
//	-1 if a < b
//	0 if a == b
func CompareSemanticVersions(a, b string) int {
	aMajor, aMinor, aPatch, aRemainder := parseSemanticVersion(a)
	bMajor, bMinor, bPatch, bRemainder := parseSemanticVersion(b)

	if aMajor != bMajor {
		if aMajor > bMajor {
			return 1
		}
		return -1
	}

	if aMinor != bMinor {
		if aMinor > bMinor {
			return 1
		}
		return -1
	}

	if aPatch != bPatch {
		if aPatch > bPatch {
			return 1
		}
		return -1
	}

	// If numeric parts are equal, compare remainders lexicographically
	if aRemainder != bRemainder {
		if aRemainder > bRemainder {
			return 1
		}
		return -1
	}

	return 0
}

// GetVersion returns a specific version of a package, or the latest if version is empty
func (entry PackageIndexEntry) GetVersion(version string) (PackageVersionEntry, error) {
	if len(entry.Versions) == 0 {
		// Backwards compatibility: single version stored in entry fields
		return PackageVersionEntry{
			Version:      entry.Version,
			Location:     entry.Location,
			Dependencies: entry.Dependencies,
			Conflicts:    entry.Conflicts,
		}, nil
	}

	if version == "" {
		// Return the latest version (highest by semantic versioning)
		return entry.GetLatestVersion()
	}

	for _, v := range entry.Versions {
		if v.Version == version {
			return v, nil
		}
	}
	return PackageVersionEntry{}, fmt.Errorf("version %s not found for package %s", version, entry.Name)
}

// GetLatestVersion returns the latest version of a package (highest by semantic versioning)
func (entry PackageIndexEntry) GetLatestVersion() (PackageVersionEntry, error) {
	if len(entry.Versions) == 0 {
		return PackageVersionEntry{
			Version:      entry.Version,
			Location:     entry.Location,
			Dependencies: entry.Dependencies,
			Conflicts:    entry.Conflicts,
		}, nil
	}

	versions := make([]PackageVersionEntry, len(entry.Versions))
	copy(versions, entry.Versions)
	sort.Slice(versions, func(i, j int) bool {
		return CompareSemanticVersions(versions[i].Version, versions[j].Version) > 0 // Descending order
	})
	return versions[0], nil
}

// ParsePackageSpec parses a package specification like "pkg" or "pkg@version"
// and returns the package name and version (empty string if no version specified)
func ParsePackageSpec(spec string) (name string, version string) {
	parts := strings.SplitN(spec, "@", 2)
	name = parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}
	return
}

func (index Index) ReadPackageLocation(pkgSpec string) (string, error) {
	name, version := ParsePackageSpec(pkgSpec)
	pkg, err := index.indexFile.GetPackage(name)
	if err != nil {
		return "", err
	}
	versionEntry, err := pkg.GetVersion(version)
	if err != nil {
		return "", err
	}
	return versionEntry.Location, nil
}

func (index Index) ReadPackageDependencies(pkgSpec string) ([]string, error) {
	name, version := ParsePackageSpec(pkgSpec)
	pkg, err := index.indexFile.GetPackage(name)
	if err != nil {
		return []string{}, err
	}
	versionEntry, err := pkg.GetVersion(version)
	if err != nil {
		return []string{}, err
	}
	return versionEntry.Dependencies, nil
}

func (index Index) ReadPackageConflicts(pkgSpec string) ([]string, error) {
	name, version := ParsePackageSpec(pkgSpec)
	pkg, err := index.indexFile.GetPackage(name)
	if err != nil {
		return []string{}, err
	}
	versionEntry, err := pkg.GetVersion(version)
	if err != nil {
		return []string{}, err
	}
	return versionEntry.Conflicts, nil
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
