package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
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
	versionHash   string
}

type GithubClient struct {
	token  string
	repo   Repository
	client *github.Client
}

func New(token string, repository string, releaseBranch string) (*GithubClient, error) {
	ctx := context.Background()

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token, TokenType: "token"})
	client := github.NewClient(oauth2.NewClient(ctx, tokenSource))

	parts := strings.Split(repository, "/")
	owner := parts[0]
	repoName := parts[1]

	version, commit, err := getLatestTag(client, owner, repoName)
	if err != nil {
		return nil, err
	}

	repo := Repository{
		repoName,
		owner,
		releaseBranch,
		version,
		commit,
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

	mergeCommit := pr.GetMergeCommitSHA()

	log.Printf("Event pull request:")
	log.Printf("  Action:          %s", action)
	log.Printf("  IsMerged:        %v", isMerged)
	log.Printf("  Base Ref:        %s", baseRef)
	log.Printf("  Merge Commit:    %s", mergeCommit)
	log.Printf("  Workflow Commit: %s", commitSha)

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

	if mergeCommit != commitSha {
		return fmt.Errorf("workflow run arguments and pull request data mismatch")
	}

	if mergeCommit == g.repo.versionHash {
		log.Printf("Detected this commit has already been tagged with the latest version. No new tag necessary.")

		return nil
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
	refValue := fmt.Sprintf("refs/tags/%s", version)
	ref := &github.Reference{
		Ref: github.String(refValue),
		Object: &github.GitObject{
			SHA: &commitSha,
		},
	}

	_, _, err := g.client.Git.CreateRef(ctx, g.repo.owner, g.repo.name, ref)
	if err != nil {
		return fmt.Errorf("failed to create new ref (%s): %v", refValue, err)
	}

	return nil
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

func getLatestTag(client *github.Client, owner string, repo string) (semver.SemVer, string, error) {
	res := semver.SemVer{}
	commit := ""
	ctx := context.Background()

	refs, response, err := client.Git.ListMatchingRefs(ctx, owner, repo, &github.ReferenceListOptions{
		Ref: "tags",
	})

	scopes := response.Header.Get("X-OAuth-Scopes")
	log.Printf("GitHub client authorized for scopes: %s", scopes)

	for k, v := range response.Header {
		log.Printf("Header: %-32s %v", k, v)
	}

	if err != nil {
		return res, commit, err
	}

	for _, ref := range refs {
		version, err := semver.New(strings.Replace(*ref.Ref, "refs/tags/", "", 1))
		if err != nil {
			log.Printf("Ignoring tag: %s", *ref.Ref)

			continue
		}

		if version.IsGreaterThan(res) {
			if ref.Object == nil || ref.Object.SHA == nil {
				return res, commit, fmt.Errorf("unable to extract hash from tag: %s", version)
			}

			res = version
			commit = *ref.Object.SHA
		}
	}

	log.Printf("Found previous version tag: %s (commit: %s)", res, commit)

	return res, commit, nil
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
