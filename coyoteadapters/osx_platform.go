package coyoteadapters

import (
	"os/exec"

	core "nhs.uk/coyotecore"
)

type OSXPlatform struct{}

func (p OSXPlatform) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

func NewOSXPlatform() OSXPlatform {
	return OSXPlatform{}
}

func NewPlatform() core.Platform {
	// TODO: Detect the platform and return the appropriate implementation
	return NewOSXPlatform()
}
