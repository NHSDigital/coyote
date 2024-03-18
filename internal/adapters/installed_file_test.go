package adapters

import (
	"os"
	"testing"
)

// Test that we write the package name and version to the installed file
func TestInstalledFile_Record(t *testing.T) {
	// Create a new installed file in the temp directory
	path := os.TempDir() + "/test-installed"
	file, err := os.Create(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file.Close()
	defer os.Remove(path)

	installedFile := NewInstalledFile(path)
	installedFile.Record("test-package", "1.0.0")

	// Read the file and check that the package name and version are present
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	expected := "test-package=1.0.0\n"
	if string(contents) != expected {
		t.Errorf("Expected contents to be '%s', got '%s'", expected, string(contents))
	}
}

// Test that we append to the installed file, so that the install order is preserved
func TestInstalledFile_Record_Append(t *testing.T) {
	// Create a new installed file in the temp directory
	path := os.TempDir() + "/test-installed"
	file, err := os.Create(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file.WriteString("test-package=1.0.0\n")
	file.Close()
	defer os.Remove(path)

	installedFile := NewInstalledFile(path)
	installedFile.Record("test-package2", "1.0.0")

	// Read the file and check that the package name and version are present
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	expected := "test-package=1.0.0\ntest-package2=1.0.0\n"
	if string(contents) != expected {
		t.Errorf("Expected contents to be '%s', got '%s'", expected, string(contents))
	}
}

// Test that we create the installed file if it doesn't exist
func TestInstalledFile_Record_Create(t *testing.T) {
	// Create a new installed file in the temp directory
	path := os.TempDir() + "/test-installed"
	installedFile := NewInstalledFile(path)
	defer os.Remove(path)
	installedFile.Record("test-package", "1.0.0")

	// Read the file and check that the package name and version are present
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	expected := "test-package=1.0.0\n"
	if string(contents) != expected {
		t.Errorf("Expected contents to be '%s', got '%s'", expected, string(contents))
	}
}

// Test that we only write the package name and version the the installed file once
func TestInstalledFile_Record_Once(t *testing.T) {
	// Create a new installed file in the temp directory
	path := os.TempDir() + "/test-installed"

	installedFile := NewInstalledFile(path)
	defer os.Remove(path)

	installedFile.Record("test-package", "1.0.0")
	installedFile.Record("test-package", "1.0.0")

	// Read the file and check that the package name and version are present
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	expected := "test-package=1.0.0\n"
	if string(contents) != expected {
		t.Errorf("Expected contents to be '%s', got '%s'", expected, string(contents))
	}
}

// If the version is different, overwrite the version
func TestInstalledFile_Record_Overwrite(t *testing.T) {
	// Create a new installed file in the temp directory
	path := os.TempDir() + "/test-installed"

	installedFile := NewInstalledFile(path)
	defer os.Remove(path)

	installedFile.Record("test-package", "1.0.0")
	installedFile.Record("test-package", "1.0.1")

	// Read the file and check that the package name and version are present
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	expected := "test-package=1.0.1\n"
	if string(contents) != expected {
		t.Errorf("Expected contents to be '%s', got '%s'", expected, string(contents))
	}
}

// Test that we can read the installed file
func TestInstalledFile_Read(t *testing.T) {
	// Create a new installed file in the temp directory
	path := os.TempDir() + "/test-installed"
	file, err := os.Create(path)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file.WriteString("test-package=1.0.0\ntest-package2=1.0.0\n")
	file.Close()
	defer os.Remove(path)

	installedFile := NewInstalledFile(path)
	installed, err := installedFile.Read()
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}

	// Check that the installed packages are as expected
	expected := [][]string{
		{"test-package", "1.0.0"},
		{"test-package2", "1.0.0"},
	}
	if len(installed) != len(expected) {
		t.Errorf("Expected %d packages, got %d", len(expected), len(installed))
	}
	for i, pkg := range installed {
		if pkg[0] != expected[i][0] || pkg[1] != expected[i][1] {
			t.Errorf("Expected package %s version %s, got %s version %s", expected[i][0], expected[i][1], pkg[0], pkg[1])
		}
	}
}
