package coyoteadapters

import (
	"context"
	"os"

	github "github.com/google/go-github/v58/github"
)

type GithubSourceControl struct {
	authToken string
	Client    *github.Client
	context   context.Context
}

func NewGithubSourceControl(authToken string) GithubSourceControl {
	return GithubSourceControl{
		authToken: authToken,
		Client:    github.NewClient(nil).WithAuthToken(authToken),
		context:   context.Background(),
	}
}

// we do this when we create a new package, project, or index
func (s GithubSourceControl) IsNameAvailable(repo string, org string) (bool, error) {
	_, response, err := s.Client.Repositories.Get(s.context, org, repo)

	if response.StatusCode == 404 {
		return true, nil
	} else if response.StatusCode == 200 {
		return false, nil
	} else {
		return false, err
	}
}

// we do this when we create a new package, project, or index
func (s GithubSourceControl) CreateRepo(repo string, org string) error {
	_, _, err := s.Client.Repositories.Create(s.context, org, &github.Repository{Name: &repo})
	return err
}

// we do this whenever we publish a package
// Returns a list of URLs to the files which can go straight in the index source
func (s GithubSourceControl) CreateRelease(repo string, org string, tag string, filenames []string) ([]string, error) {
	// Open all the files first, so that we can barf before we upload anything
	files := make([]*os.File, len(filenames))
	for i, filename := range filenames {
		file, err := os.Open(filename)
		defer file.Close()
		if err != nil {
			return nil, err
		}
		files[i] = file
	}

	// The github release model is that you create a release, then upload assets to it. So
	// we create the release first...
	release, _, err := s.Client.Repositories.CreateRelease(s.context, org, repo, &github.RepositoryRelease{
		TagName: &tag,
	})
	if err != nil {
		return nil, err
	}
	releaseId := release.GetID()
	result := make([]string, len(filenames))

	//...then upload the assets
	for _, file := range files {
		uploadResponse, _, err := s.Client.Repositories.UploadReleaseAsset(s.context, org, repo, releaseId, &github.UploadOptions{
			Name: file.Name(),
		}, file)
		if err != nil {
			return nil, err
		}
		result = append(result, uploadResponse.GetBrowserDownloadURL())
	}
	return result, nil
}

// This is just for cleaning up after testing, really, but we need it for convenience
func (s GithubSourceControl) DeleteRepo(repo string, org string) error {
	_, err := s.Client.Repositories.Delete(s.context, org, repo)
	return err
}

// In case we need to forget that a release happened
func (s GithubSourceControl) DeleteRelease(repo string, org string, tag string) error {
	release, _, err := s.Client.Repositories.GetReleaseByTag(s.context, org, repo, tag)
	if err != nil {
		return err
	}
	releaseId := release.GetID()
	_, err = s.Client.Repositories.DeleteRelease(s.context, org, repo, releaseId)
	return err
}

func (s GithubSourceControl) GetRateLimitDelayMilliseconds() int {
	return 500
}
