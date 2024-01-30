package coyoteadapters

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type TomlConfig struct {
	Index      string `toml:"index"`
	PackageOrg string `toml:"package_org"`
	Path       string
}

func NewTomlConfig(path string) (*TomlConfig, error) {
	config := &TomlConfig{}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist", path)
	} else if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	err = toml.Unmarshal(raw, config)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// This is specific to the toml config.  It's not valid to set the index
	// to an empty string *here*, but the NullConfig returns it in the core module.
	if config.Index == "" {
		return nil, fmt.Errorf("Invalid config file: index must be set")
	}

	// Store the path to the config file so we can avoid having to repeatedly
	// reparse argv
	config.Path = path

	return config, nil
}

func (c *TomlConfig) GetIndex() string {
	return c.Index
}

func (c *TomlConfig) GetPackageOrg() string {
	return c.PackageOrg
}

func (c *TomlConfig) GetPath() string {
	return c.Path
}
