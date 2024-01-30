package coyotecore

import (
	"fmt"
	"os"
	"testing"
)

func TestPackageRelease(t *testing.T) {

	t.Run("Test not in a Coyote package", func(t *testing.T) {
		// Call the PackageRelease function without creating a .cypkg file
		_, err := PackageRelease(nil, "anything", "anything")
		expectedErr := fmt.Errorf("Not in a Coyote package.\n")
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedErr, err)
		}
	})

	t.Run("Test not in a git repo", func(t *testing.T) {
		// Create a temporary .cypkg file to simulate being in a Coyote package
		tmpDir, err := os.MkdirTemp("", "coyote")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %s", err)
		}
		defer os.RemoveAll(tmpDir)

		cypkgPath := tmpDir + "/.cypkg"
		_, err = os.Create(cypkgPath)
		if err != nil {
			t.Fatalf("Failed to create .cypkg file: %s", err)
		}

		cwd := os.Getenv("PWD")
		os.Chdir(tmpDir)
		defer os.Chdir(cwd)

		// Call the PackageRelease function without creating a .git file
		_, err = PackageRelease(nil, "anything", "anything")
		expectedErr := fmt.Errorf("Not in a git repo.\n")
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedErr, err)
		}
	})
}
