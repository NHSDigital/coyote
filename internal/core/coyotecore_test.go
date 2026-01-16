package core

import (
	"fmt"
	"os"
	"testing"
)

func TestCompareSemanticVersions(t *testing.T) {
	testCases := []struct {
		a, b     string
		expected int
	}{
		// Basic comparisons
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},

		// Minor version comparisons
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},

		// Patch version comparisons
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},

		// Semantic versioning: 1.0.11 > 1.0.2 (not asciibetic!)
		{"1.0.11", "1.0.2", 1},
		{"1.0.2", "1.0.11", -1},

		// With 'v' prefix
		{"v1.0.11", "v1.0.2", 1},
		{"v2.0.0", "v1.9.9", 1},

		// Mixed prefix
		{"v1.0.0", "1.0.0", 0},

		// Partial versions
		{"1.0", "1.0.0", 0},
		{"1", "1.0.0", 0},
		{"2", "1.9.9", 1},

		// With suffixes
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-alpha", 1},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_vs_%s", tc.a, tc.b), func(t *testing.T) {
			result := CompareSemanticVersions(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("CompareSemanticVersions(%q, %q) = %d, want %d", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

func TestPackageRelease(t *testing.T) {

	t.Run("Test not in a Coyote package", func(t *testing.T) {
		// Call the PackageRelease function without creating a .cypkg file
		_, err := PackageRelease(nil, "anything", "anything")
		expectedErr := fmt.Errorf("not in a Coyote package")
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
		expectedErr := fmt.Errorf("not in a git repository")
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedErr, err)
		}
	})
}
