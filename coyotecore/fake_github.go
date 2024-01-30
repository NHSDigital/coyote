package coyotecore

import "fmt"

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
	fmt.Println("NullSourceControl called: NullSourceControl.DownloadReleaseFile(", href, ")")
	return "", nil
}
