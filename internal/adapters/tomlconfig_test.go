package adapters

import (
	"fmt"
	"os"
	"testing"
)

func CreateTomlFileWithContents(path string, contents string) {
	// Create a new toml file at the given path and write the index value
	// to the file
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.WriteString(contents)

	if err != nil {
		panic(err)
	}
}

func CreateTomlFile(path string, index string) {
	// Create a new toml file at the given path and write the index value
	// to the file
	CreateTomlFileWithContents(path, fmt.Sprintf("index = \"%s\"", index))
}

func TestConfigReader(t *testing.T) {
	t.Run("Test reading a toml file", func(t *testing.T) {
		// Create a new toml file in the temp directory
		path := os.TempDir() + "/test.toml"
		CreateTomlFile(path, "test")
		defer os.Remove(path)

		config, err := NewTomlConfig(path)
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}
		index := config.GetIndex()
		if index != "test" {
			t.Errorf("Expected index to be 'test', got '%s'", index)
		}
	})

	t.Run("Test reading a non-existent file", func(t *testing.T) {
		config, err := NewTomlConfig("non-existent-file.toml")
		if err == nil {
			t.Errorf("Expected error, got '%s'", err)
		}
		if config != nil {
			t.Errorf("Expected config to be nil, got '%s'", config)
		}
	})

	t.Run("Test reading a file with invalid toml", func(t *testing.T) {
		path := os.TempDir() + "/test.toml"
		CreateTomlFile(path, "test")
		defer os.Remove(path)

		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		_, err = file.WriteString("invalid toml")

		if err != nil {
			panic(err)
		}

		config, err := NewTomlConfig(path)
		if err == nil {
			t.Errorf("Expected error, got '%s'", err)
		}
		if config != nil {
			t.Errorf("Expected config to be nil, got '%s'", config)
		}
	})

	t.Run("Test reading a file with missing index", func(t *testing.T) {
		path := os.TempDir() + "/test.toml"
		CreateTomlFile(path, "")
		defer os.Remove(path)

		config, err := NewTomlConfig(path)
		if err == nil {
			t.Errorf("Expected error, got '%s'", err)
		}
		if config != nil {
			t.Errorf("Expected config to be nil, got '%s'", config)
		}
	})

	// TODO: test that the config struct stores its own path
	t.Run("Test that the config struct stores its own path", func(t *testing.T) {
		path := os.TempDir() + "/test.toml"
		CreateTomlFile(path, "test")
		defer os.Remove(path)

		config, err := NewTomlConfig(path)
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}
		returnedPath := config.GetPath()
		if returnedPath != path {
			t.Errorf("Expected config path to be '%s', got '%s'", path, returnedPath)
		}
	})

	t.Run("We can read the package org", func(t *testing.T) {
		path := os.TempDir() + "/test.toml"
		CreateTomlFileWithContents(path, "package_org = \"test-org\"\nindex = \"test\"")
		defer os.Remove(path)

		config, err := NewTomlConfig(path)
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}
		org := config.GetPackageOrg()
		if org != "test-org" {
			t.Errorf("Expected package org to be 'test-org', got '%s'", org)
		}
	})
}
