package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Meta struct {
	RepoURL   string
	CommitSHA string
	Branch    string
}

func DetectMeta() (*Meta, error) {
	repoURL, err := detectRepoURL()
	if err != nil {
		return nil, fmt.Errorf("could not detect git remote URL — pass --repo-url explicitly: %w", err)
	}

	commitSHA, err := detectCommitSHA()
	if err != nil {
		return nil, fmt.Errorf("could not detect commit SHA — pass --commit-sha explicitly: %w", err)
	}

	branch := detectBranch()

	return &Meta{
		RepoURL:   NormalizeRepoURL(repoURL),
		CommitSHA: commitSHA,
		Branch:    branch,
	}, nil
}

func detectRepoURL() (string, error) {
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		return "https://github.com/" + repo, nil
	}
	return gitExec("remote", "get-url", "origin")
}

func detectCommitSHA() (string, error) {
	// In GitHub Actions PRs, GITHUB_SHA is the merge commit — read the actual
	// PR head SHA from the event payload so it matches the pull_request record.
	if os.Getenv("GITHUB_EVENT_NAME") == "pull_request" {
		if sha := readPRHeadSHA(); sha != "" {
			return sha, nil
		}
	}

	envVars := []string{"GITHUB_SHA", "CIRCLE_SHA1", "CI_COMMIT_SHA"}
	for _, key := range envVars {
		if v := os.Getenv(key); v != "" {
			return v, nil
		}
	}

	return gitExec("rev-parse", "HEAD")
}

func detectBranch() string {
	envVars := []string{"GITHUB_HEAD_REF", "GITHUB_REF_NAME", "CIRCLE_BRANCH", "CI_COMMIT_BRANCH"}
	for _, key := range envVars {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}

	branch, err := gitExec("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "unknown"
	}
	return branch
}

func readPRHeadSHA() string {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return ""
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return ""
	}

	var event struct {
		PullRequest struct {
			Head struct {
				SHA string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return ""
	}

	return event.PullRequest.Head.SHA
}

var sshPattern = regexp.MustCompile(`^git@([^:]+):(.+)$`)

// NormalizeRepoURL converts SSH and HTTPS git URLs to a canonical HTTPS form.
func NormalizeRepoURL(raw string) string {
	url := strings.TrimSpace(raw)

	if m := sshPattern.FindStringSubmatch(url); m != nil {
		url = "https://" + m[1] + "/" + m[2]
	}

	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimRight(url, "/")

	return url
}

func gitExec(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
