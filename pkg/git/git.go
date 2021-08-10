package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v37/github"
	"github.com/infobloxopen/auto-semver-tag/pkg/semver"
	"golang.org/x/oauth2"
)

const (
	IncrementTypeMajorLabel = "major"
	IncrementTypeMinorLabel = "minor"
)

type GitRepository struct {
	name          string
	owner         string
	releaseBranch string
	version       semver.SemVer
}

type GitClient struct {
	token  string
	repo   GitRepository
	client *github.Client
}

func New(token string, repository string, releaseBranch string) (*GitClient, error) {
	ctx := context.Background()

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(ctx, tokenSource))

	parts := strings.Split(repository, "/")
	owner := parts[0]
	repoName := parts[1]

	version, err := getLatestTag(client, owner, repoName)
	if err != nil {
		return nil, err
	}

	repo := GitRepository{
		repoName,
		owner,
		releaseBranch,
		version,
	}

	return &GitClient{
		token,
		repo,
		client,
	}, nil
}

func (gitClient *GitClient) PerformAction(commitSha string, eventDataFilePath string) error {
	event, err := parseEventDataFile(eventDataFilePath)
	if err != nil {
		return err
	}

	if event.Action == nil || *event.Action != "closed" {
		return errors.New("pull request is not closed")
	}

	if event.PullRequest.Merged == nil || !*event.PullRequest.Merged {
		return errors.New("pull request is not merged")
	}

	if event.PullRequest.Base == nil || event.PullRequest.Base.Ref == nil {
		return errors.New("could not determine pull request base branch")
	}

	if *event.PullRequest.Base.Ref != gitClient.repo.releaseBranch {
		return errors.New("pull request is merged not into the release branch")
	}

	hasMajor, hasMinor := parsePullRequestLabels(event.PullRequest)

	var newVersion semver.SemVer
	if hasMajor {
		newVersion = gitClient.repo.version.IncrementVersion(semver.IncrementTypeMajor)
	} else if hasMinor {
		newVersion = gitClient.repo.version.IncrementVersion(semver.IncrementTypeMinor)
	} else {
		newVersion = gitClient.repo.version.IncrementVersion(semver.IncrementTypePatch)
	}

	if !newVersion.IsGreaterThan(semver.SemVer{}) {
		return errors.New("new version is 0.0.0")
	}

	err = gitClient.createTag(newVersion.String(), commitSha)
	if err != nil {
		return err
	}

	return nil
}

func (gitClient *GitClient) createTag(version string, commitSha string) error {
	ctx := context.Background()
	ref := &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/tags/%s", version)),
		Object: &github.GitObject{
			SHA: &commitSha,
		},
	}

	_, _, err := gitClient.client.Git.CreateRef(ctx, gitClient.repo.owner, gitClient.repo.name, ref)

	return err
}

func parsePullRequestLabels(pr *github.PullRequest) (hasMajor bool, hasMinor bool) {
	for _, label := range pr.Labels {
		if label.Name == nil {
			continue
		}

		if *label.Name == IncrementTypeMajorLabel {
			hasMajor = true
			continue
		}

		if *label.Name == IncrementTypeMinorLabel {
			hasMinor = true
			continue
		}

	}

	return
}

func parseEventDataFile(filePath string) (*github.PullRequestEvent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s. Filepath: %s", err, filePath)
	}
	defer file.Close()

	event, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%s. Filepath: %s", err, filePath)
	}

	eventData, err := github.ParseWebHook("pull_request", stripOrg(event))
	if err != nil {
		return nil, fmt.Errorf("%s. Filepath: %s", err, filePath)
	}

	res, ok := eventData.(*github.PullRequestEvent)
	if !ok {
		return nil, errors.New("could not parse GitHub event into a PullRequestEvent")
	}

	return res, nil
}

func getLatestTag(client *github.Client, owner string, repo string) (semver.SemVer, error) {
	res := semver.SemVer{}
	ctx := context.Background()

	refs, response, err := client.Git.ListMatchingRefs(ctx, owner, repo, &github.ReferenceListOptions{
		Ref: "tags",
	})
	if err != nil {
		return res, err
	}

	if response != nil && response.StatusCode == http.StatusNotFound {
		return res, nil
	}

	for _, ref := range refs {
		version, err := semver.New(strings.Replace(*ref.Ref, "refs/tags/", "", 1))
		if err != nil {
			continue
		}

		if version.IsGreaterThan(res) {
			res = version
		}
	}

	return res, nil
}

func stripOrg(byteString []byte) []byte {
	// workaround for https://github.com/google/go-github/issues/131
	var o map[string]interface{}
	_ = json.Unmarshal(byteString, &o)
	if o != nil {
		repo := o["repository"]
		if repo != nil {
			if repo, ok := repo.(map[string]interface{}); ok {
				delete(repo, "organization")
			}
		}
	}
	b, _ := json.MarshalIndent(o, "", "  ")
	return b
}
