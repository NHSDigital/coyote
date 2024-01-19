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

type Context struct {
	Config Config
}
