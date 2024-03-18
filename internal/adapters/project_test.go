package adapters

import (
	"os"
	"testing"

	core "github.com/nhsdigital/coyote/internal/core"
)

func newProject(path string) core.Project {
	return NewProjectProvider().NewProject(path, "test-project")
}

func maybeProject(path string) core.Project {
	return NewProjectProvider().MaybeProject(path)
}

// Test that we can read the project name from .coyote/project-name
func TestProject_GetName(t *testing.T) {
	path := "/tmp/test-project"
	err := os.MkdirAll(path+"/.coyote", 0755)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file, err := os.Create(path + "/.coyote/project-name")
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file.WriteString("test-project")
	file.Close()
	defer os.RemoveAll(path)

	project := newProject(path)
	name := project.GetName()
	if name != "test-project" {
		t.Errorf("Expected project name to be 'test-project', got '%s'", name)
	}
}

// Test that the project name is stripped of whitespace
func TestProject_GetName_Whitespace(t *testing.T) {
	path := "/tmp/test-project"
	err := os.MkdirAll(path+"/.coyote", 0755)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file, err := os.Create(path + "/.coyote/project-name")
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	file.WriteString("test-project\n")
	file.Close()
	defer os.RemoveAll(path)

	project := newProject(path)
	name := project.GetName()
	if name != "test-project" {
		t.Errorf("Expected project name to be 'test-project', got '%s'", name)
	}
}

// Test that MaybeProject returns a Project struct if the current working directory is a project
func TestMaybeProject_Yes(t *testing.T) {
	path := "/tmp/test-project"
	err := os.MkdirAll(path+"/.coyote", 0755)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	defer os.RemoveAll(path)

	os.Chdir(path)
	project := maybeProject(path)
	if project == nil {
		t.Errorf("Expected project to be non-nil")
	}
}

// Test that MaybeProject returns nil if the current working directory is not a project
func TestMaybeProject_No(t *testing.T) {
	path := "/tmp/test-project"
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}
	defer os.RemoveAll(path)

	os.Chdir(path)
	project := maybeProject(path)
	if project != nil {
		t.Errorf("Expected project to be nil")
	}
}

// Test that Init creates the .coyote directory and the project-name file
func TestProject_Init(t *testing.T) {
	path := "/tmp/test-project"
	defer os.RemoveAll(path)

	newProject(path)
	_, err := os.Stat(path + "/.coyote")
	if err != nil {
		t.Errorf("Expected .coyote directory to exist, got '%s'", err)
	}
	_, err = os.Stat(path + "/.coyote/project-name")
	if err != nil {
		t.Errorf("Expected project-name file to exist, got '%s'", err)
	}
}
