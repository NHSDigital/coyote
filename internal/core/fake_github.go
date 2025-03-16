package core

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// We provide a null source control adapter for testing purposes.  It logs
// all the calls it receives to stdout, so the tests can see it, but does nothing.
// It keeps a track of the repos it's created, so the name availability check
// can return the correct result.

type NullSourceControl struct {
	//List of repo names that we've created
	createdRepos []string
}

// Return a pointer because we need to hold state client-side, which isn't
// the case with the real source control adapter
func NewNullSourceControl() *NullSourceControl {
	return &NullSourceControl{}
}

func (s *NullSourceControl) IsNameAvailable(repo string, org string) (bool, error) {
	fmt.Println("NullSourceControl called: NullSourceControl.IsNameAvailable(", repo, ",", org, ")")
	for _, createdRepo := range s.createdRepos {
		if createdRepo == repo {
			return false, nil
		}
	}
	return true, nil
}

func (s *NullSourceControl) CreateRepo(repo string, org string) error {
	fmt.Println("NullSourceControl called: NullSourceControl.CreateRepo(", repo, ",", org, ")")
	s.createdRepos = append(s.createdRepos, repo)
	return nil
}

func (s *NullSourceControl) DeleteRepo(repo string, org string) error {
	fmt.Println("NullSourceControl called: NullSourceControl.DeleteRepo(", repo, ",", org, ")")
	return nil
}

func (s *NullSourceControl) CreateRelease(repo string, org string, tag string, filenames []string) ([]string, error) {
	fmt.Println("NullSourceControl called: NullSourceControl.CreateRelease(",
		repo, ",", org, ",", tag, ",", filenames, ")")
	// We don't actually upload anything, but we do want to return something, so we just give the filename back
	return filenames, nil
}

func (s *NullSourceControl) DeleteRelease(repo string, org string, tag string) error {
	fmt.Println("NullSourceControl called: NullSourceControl.DeleteRelease(", repo, ",", org, ",", tag, ")")
	return nil
}

func (s *NullSourceControl) GetRateLimitDelayMilliseconds() int {
	return 0
}

func (s *NullSourceControl) DoesReleaseExist(repo string, org string, tag string) (bool, error) {
	fmt.Println("NullSourceControl called: NullSourceControl.DoesReleaseExist(", repo, ",", org, ",", tag, ")")
	return false, nil
}

func (s *NullSourceControl) DownloadReleaseFile(href string) (string, error) {
	/*  Note that this function only refuses to download files from github.com.
	 *  It will download files from localhost. */
	fmt.Println("NullSourceControl called: NullSourceControl.DownloadReleaseFile(", href, ")")
	hrefWithoutFragment := strings.Split(href, "#")[0]
	parsedUrl, err := url.Parse(hrefWithoutFragment)
	if err != nil {
		return "", err
	}

	if parsedUrl.Host == "github.com" || parsedUrl.Host == "api.github.com" {
		return "nosuchfile", nil
	}

	filename := parsedUrl.Path
	basename := strings.Split(filename, "/")[len(strings.Split(filename, "/"))-1]
	if basename == "" {
		return "", fmt.Errorf("error parsing filename from url %s", href)
	}

	targetFilename := "/tmp/" + basename

	cmd := exec.Command("wget", "-O", targetFilename, href)
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error downloading file from %s: %v", href, err)
	}
	return targetFilename, nil
}

func (s *NullSourceControl) GetRemoteURL(repo string, org string) string {
	fmt.Println("NullSourceControl called: NullSourceControl.GetRemoteURL(", repo, ",", org, ")")
	return "fake-remote-url"
}

func (s *NullSourceControl) Push(repo string, org string) error {
	fmt.Println("NullSourceControl called: NullSourceControl.Push(", repo, ",", org, ")")
	return nil
}
