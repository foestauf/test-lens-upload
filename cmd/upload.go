package cmd

import (
	"fmt"
	"os"

	"github.com/foestauf/test-lens-upload/internal/discover"
	"github.com/foestauf/test-lens-upload/internal/git"
	"github.com/foestauf/test-lens-upload/internal/upload"
	"github.com/spf13/cobra"
)

var (
	flagFile      string
	flagRepoURL   string
	flagCommitSHA string
	flagBranch    string
	flagAPIURL    string
	flagNoOIDC    bool
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a coverage report",
	Long: `Upload a coverage report to test-lens.

Authentication:
  In GitHub Actions, the CLI automatically requests an OIDC token for
  authentication. Your workflow must include the following permission:

    permissions:
      id-token: write

  Use --no-oidc to skip OIDC and upload without authentication.`,
	RunE: runUpload,
}

func init() {
	uploadCmd.Flags().StringVar(&flagFile, "file", "", "Path to coverage file (auto-detected if omitted)")
	uploadCmd.Flags().StringVar(&flagRepoURL, "repo-url", "", "Git repository URL (auto-detected from git remote)")
	uploadCmd.Flags().StringVar(&flagCommitSHA, "commit-sha", "", "Commit SHA (auto-detected from git)")
	uploadCmd.Flags().StringVar(&flagBranch, "branch", "", "Branch name (auto-detected from git)")
	uploadCmd.Flags().StringVar(&flagAPIURL, "api-url", "", "test-lens API base URL")
	uploadCmd.Flags().BoolVar(&flagNoOIDC, "no-oidc", false, "Skip OIDC token auto-detection")
}

func runUpload(cmd *cobra.Command, args []string) error {
	// Resolve API URL
	apiURL := flagAPIURL
	if apiURL == "" {
		apiURL = os.Getenv("TEST_LENS_API_URL")
	}
	if apiURL == "" {
		apiURL = "https://test-lens-api.foestauf.dev"
	}

	// Resolve git metadata
	repoURL := flagRepoURL
	commitSHA := flagCommitSHA
	branch := flagBranch

	if repoURL == "" || commitSHA == "" || branch == "" {
		meta, err := git.DetectMeta()
		if err != nil {
			return fmt.Errorf("git detection failed: %w", err)
		}
		if repoURL == "" {
			repoURL = meta.RepoURL
		}
		if commitSHA == "" {
			commitSHA = meta.CommitSHA
		}
		if branch == "" {
			branch = meta.Branch
		}
	}

	// Resolve coverage file
	filePath := flagFile
	if filePath == "" {
		found := discover.FindCoverageFile(".")
		if found == "" {
			return fmt.Errorf("no coverage file found — pass --file explicitly")
		}
		fmt.Fprintf(os.Stderr, "Auto-detected coverage file: %s\n", found)
		filePath = found
	} else {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
	}

	fmt.Fprintf(os.Stderr, "Uploading %s for %s @ %s...\n", filePath, repoURL, commitSHA[:8])

	result, err := upload.Upload(upload.Options{
		APIURL:    apiURL,
		FilePath:  filePath,
		RepoURL:   repoURL,
		CommitSHA: commitSHA,
		Branch:    branch,
		NoOIDC:    flagNoOIDC,
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Upload accepted. ID: %s, status: %s\n", result.UploadID, result.Status)
	return nil
}
