package adapters

import (
	"fmt"
	"os"
	"strings"
)

// The InstalledFile struct represents the .coyote/installed file that records which packages are installed.
type InstalledFile struct {
	Filename string
}

func NewInstalledFile(filename string) InstalledFile {
	return InstalledFile{Filename: filename}
}

func writeLinesToFile(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, line := range lines {
		if line != "" {
			_, err = file.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func installedLines(filename string) ([]string, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		} else {
			return nil, err
		}
	}
	return strings.Split(string(contents), "\n"), nil
}

// Record marks a package as installed by adding its name to the .coyote/installed file.
func (f InstalledFile) Record(packageName string, version string) error {
	// packageName or version being empty is an error
	if packageName == "" {
		return fmt.Errorf("missing NAME")
	}
	if version == "" {
		return fmt.Errorf("missing VERSION")
	}

	// packageName or version having an = in them is an error
	if strings.Contains(packageName, "=") {
		return fmt.Errorf("NAME cannot contain '='")
	}
	if strings.Contains(version, "=") {
		return fmt.Errorf("VERSION cannot contain '='")
	}

	lines, err := installedLines(f.Filename)
	if err != nil {
		return err
	}

	// Check if the package is already recorded
	recorded := false
	for i, line := range lines {
		if strings.HasPrefix(line, packageName+"=") {
			if strings.HasSuffix(line, "="+version) {
			} else {
				lines[i] = packageName + "=" + version
			}
			recorded = true
		}
	}

	if !recorded {
		lines = append(lines, packageName+"="+version)
	}
	tempfile := f.Filename + ".tmp"
	err = writeLinesToFile(tempfile, lines)
	if err != nil {
		return err
	}
	err = os.Rename(tempfile, f.Filename)
	if err != nil {
		return err
	}
	return nil
}

// InstalledPackages returns a list of installed packages.
func (f InstalledFile) Read() ([][]string, error) {
	lines, err := installedLines(f.Filename)
	if err != nil {
		return nil, err
	}
	packages := make([][]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			packages = append(packages, strings.Split(line, "="))
		}
	}
	return packages, nil
}
