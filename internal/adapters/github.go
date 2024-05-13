package adapters

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

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
		if err != nil {
			return nil, err
		}
		defer file.Close()

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
			Name: path.Base(file.Name()),
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

func (s GithubSourceControl) DoesReleaseExist(repo string, org string, tag string) (bool, error) {
	_, response, err := s.Client.Repositories.GetReleaseByTag(s.context, org, repo, tag)

	if response.StatusCode == 404 {
		return false, nil
	} else if response.StatusCode == 200 {
		return true, nil
	} else {
		return false, err
	}
}

func (s GithubSourceControl) DownloadReleaseFile(href string) (string, error) {
	// This function downloads a file from a remote location, and returns the local filename.
	// It returns an error if the download fails.
	// The file is downloaded to /tmp, and the filename is returned.
	// Just use wget for now.
	// The local filename is the same as the remote filename, but because we might have query strings or a fragment suffix in the url
	// we need to strip them off.
	// Note: the auth isn't covered by tests

	hrefWithoutFragment := strings.Split(href, "#")[0]
	parsedUrl, err := url.Parse(hrefWithoutFragment)
	if err != nil {
		return "", fmt.Errorf("error parsing url %s: %v", href, err)
	}
	filename := parsedUrl.Path
	basename := strings.Split(filename, "/")[len(strings.Split(filename, "/"))-1]
	if basename == "" {
		return "", fmt.Errorf("error parsing filename from url %s", href)
	}

	// Now because github's release downloads Just Don't Work with personal access tokens,
	// we have to use the API instead.  For that to work, we need the asset ID, which we
	// don't have at this point.

	if parsedUrl.Host == "github.com" {
		return downloadGithubReleaseUrl(parsedUrl, s, href, basename)
	} else {
		return downloadUrl(parsedUrl, s, href, basename)
	}

}

func downloadUrl(parsedUrl *url.URL, s GithubSourceControl, href string, basename string) (string, error) {
	targetFilename := "/tmp/" + basename

	cmd := exec.Command("wget", "-O", targetFilename, href)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error downloading file from %s: %v", href, err)
	}
	return targetFilename, nil
}

func downloadGithubReleaseUrl(parsedUrl *url.URL, s GithubSourceControl, href string, basename string) (string, error) {
	// The href we have is the browser download URL, which is not the same as the API download URL, and
	// also there is no direct way to get the asset ID from the browser download URL.  So we have to
	// use the API to list all the releases, find the one with a matching browser download URL, and then
	// get the asset ID from that.  Then we can use the API to download the file.

	// Because GitHub has apparently never heard of a redirect, or symlinks, we need to special-case
	// latest releases.  We can't just download the latest release, because the browser download URL in that
	// case doesn't appear in the list of releases.

	// What a mess.

	if parsedUrl.Host != "github.com" {
		return "", fmt.Errorf("error: download URL %s is not a github.com URL", href)
	}

	// First we need to know the org and repo.  I want to maintain the fiction that the
	// href is a URL we can just download from, so we have to parse it to get the org and repo.
	// We can assume the href is a github.com URL, because that's all we support, but we sanity check
	// anyway
	repo, org := urlToGithubRepo(parsedUrl)

	var release *github.RepositoryRelease
	var assetId int64 = -1
	var err error

	if strings.Contains(parsedUrl.Path, "releases/latest/download") {
		release, _, err = s.Client.Repositories.GetLatestRelease(s.context, org, repo)
		if err != nil {
			return "", fmt.Errorf("error getting latest release for %s/%s: %v", org, repo, err)
		}
		pathBasename := path.Base(parsedUrl.Path)
		for _, a := range release.Assets {
			assetBasename := path.Base(a.GetBrowserDownloadURL())
			// This is horribly fragile, but we don't have a better option
			if assetBasename == pathBasename {
				assetId = a.GetID()
				break
			}
		}
	} else {
		// first we list all the releases
		// Then we find the release with the matching browser download URL
		// Then we get the asset ID
		// Then finally we can download the file

		releases, _, err := s.Client.Repositories.ListReleases(s.context, org, repo, nil)
		if err != nil {
			return "", fmt.Errorf("error listing releases for %s/%s: %v", org, repo, err)
		}

		for _, r := range releases {
			for _, a := range r.Assets {
				if a.GetBrowserDownloadURL() == href {
					release = r
					break
				}
			}
		}
		if release == nil {
			return "", fmt.Errorf("error finding release with browser download URL %s", href)
		}

		for _, a := range release.Assets {
			if a.GetBrowserDownloadURL() == href {
				assetId = a.GetID()
				break
			}
		}
	}

	if assetId == -1 {
		return "", fmt.Errorf("error finding asset ID for %s", href)
	}

	assetUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/assets/%d", org, repo, assetId)

	targetFilename := "/tmp/" + basename

	// TODO: we don't need wget for this, we can just use the http client
	cmd := exec.Command("wget",
		"--header", "Authorization: Bearer "+s.authToken+"",
		"--header", "Accept: application/octet-stream",
		"--header", "X-GitHub-Api-Version: 2022-11-28",
		"-O", targetFilename,
		assetUrl)

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error downloading file from %s: %v", href, err)
	}
	return targetFilename, nil
}

func urlToGithubRepo(parsedUrl *url.URL) (string, string) {
	fragments := strings.Split(parsedUrl.Path, "/")[1:]
	org := ""
	repo := ""
	if len(fragments) > 0 {
		org = fragments[0]
	}
	if len(fragments) > 1 {
		repo = fragments[1]
	}
	return repo, org
}

func (s GithubSourceControl) Push(repo string, org string) error {
	// This function pushes the current directory to the remote repository.
	// It returns an error if the push fails.
	// Just use git for now.
	repoUrl := "https://github.com/" + org + "/" + repo + ".git"
	cmd := exec.Command("git", "push", repoUrl, "HEAD")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error pushing to remote repository: %v", err)
	}
	return nil
}
