package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v37/github"
	"github.com/infobloxopen/auto-semver-tag/pkg/semver"
	"golang.org/x/oauth2"
)

type Repository struct {
	name          string
	owner         string
	releaseBranch string
	version       semver.SemVer
}

type GithubClient struct {
	token  string
	repo   Repository
	client *github.Client
}

func New(token string, repository string, releaseBranch string) (*GithubClient, error) {
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

	repo := Repository{
		repoName,
		owner,
		releaseBranch,
		version,
	}

	return &GithubClient{
		token,
		repo,
		client,
	}, nil
}

func (g *GithubClient) PerformAction(commitSha string, eventDataFilePath string) error {
	log.Printf("Extracting event data")

	event, err := parseEventDataFile(eventDataFilePath)
	if err != nil {
		return err
	}

	pr := event.PullRequest
	if pr == nil {
		return fmt.Errorf("pull request not found in data file: %v", event)
	}

	action := ""
	if event.Action != nil {
		action = *event.Action
	}

	isMerged := false
	if pr.Merged != nil {
		isMerged = *pr.Merged
	}

	baseRef := ""
	if pr.Base != nil && pr.Base.Ref != nil {
		baseRef = *pr.Base.Ref
	}

	log.Printf("Event pull request:")
	log.Printf("  Action:   %s", action)
	log.Printf("  IsMerged: %v", isMerged)
	log.Printf("  Base Ref: %s", baseRef)

	if action != "closed" {
		return fmt.Errorf("pull request is not closed: %s", action)
	}

	if !isMerged {
		return fmt.Errorf("pull request is not merged")
	}

	if baseRef != g.repo.releaseBranch {
		return fmt.Errorf("pull request merged into a different branch (expected: %s, actual: %s)",
			g.repo.releaseBranch, baseRef)
	}

	log.Printf("Extracting SemVer labels from pull request...")

	incrementType := parsePullRequestLabels(pr)
	if incrementType == semver.IncrementTypeUnknown {
		log.Printf(`No SemVer labels found. Commit will still be using %s`, g.repo.version)
		return nil
	}

	log.Printf(`Found "%s" label.`, incrementType)

	newVersion := g.repo.version.IncrementVersion(incrementType)

	log.Printf("Incrementing to new version: %s", newVersion)

	err = g.createTag(newVersion.String(), commitSha)
	if err != nil {
		return err
	}

	return nil
}

func (g *GithubClient) createTag(version string, commitSha string) error {
	ctx := context.Background()
	ref := &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/tags/%s", version)),
		Object: &github.GitObject{
			SHA: &commitSha,
		},
	}

	_, _, err := g.client.Git.CreateRef(ctx, g.repo.owner, g.repo.name, ref)

	return err
}

func parsePullRequestLabels(pr *github.PullRequest) semver.IncrementType {
	incType := semver.IncrementTypeUnknown
	for _, label := range pr.Labels {
		if label.Name == nil {
			continue
		}

		t := semver.StringToIncrementType(*label.Name)

		if t < incType {
			incType = t
		}
	}

	return incType
}

func parseEventDataFile(filePath string) (*github.PullRequestEvent, error) {
	file, err := os.Open(filePath)
	defer func() { _ = file.Close() }()

	if err != nil {
		return nil, fmt.Errorf("%s. Filepath: %s", err, filePath)
	}

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

	if response != nil && response.StatusCode == http.StatusNotFound {
		// StatusNotFound would also cause `err != nil`, but it is not an error in this context.
		return res, nil
	}

	if err != nil {
		if response != nil {
			return res, fmt.Errorf("ListMatchingRefs failed with status: %d %s. %w",
				response.StatusCode, response.Status, err)
		}
		return res, err
	}

	for _, ref := range refs {
		version, err := semver.New(strings.Replace(*ref.Ref, "refs/tags/", "", 1))
		if err != nil {
			log.Printf("Ignoring tag: %s", *ref.Ref)
			continue
		}

		if version.IsGreaterThan(res) {
			res = version
		}
	}

	log.Printf("Found previous version tag: %s", res)

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
