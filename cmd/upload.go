package cmd

import (
	"fmt"
	"os"

	"github.com/foestauf/test-lens-upload/internal/config"
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
	flagPackage   string
	flagConfig    string
	flagFormat    string
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

  Use --no-oidc to skip OIDC and upload without authentication.

Monorepo Support:
  Use --package <name> to tag the upload with a package name.
  Create a .testlens.yml in your repo root to define packages:

    packages:
      - name: api
        path: apps/api
      - name: web
        path: apps/web

  If a config exists and no --package is specified, all packages are
  uploaded in sequence (auto-discovering coverage files per package path).`,
	RunE: runUpload,
}

func init() {
	uploadCmd.Flags().StringVar(&flagFile, "file", "", "Path to coverage file (auto-detected if omitted)")
	uploadCmd.Flags().StringVar(&flagRepoURL, "repo-url", "", "Git repository URL (auto-detected from git remote)")
	uploadCmd.Flags().StringVar(&flagCommitSHA, "commit-sha", "", "Commit SHA (auto-detected from git)")
	uploadCmd.Flags().StringVar(&flagBranch, "branch", "", "Branch name (auto-detected from git)")
	uploadCmd.Flags().StringVar(&flagAPIURL, "api-url", "", "test-lens API base URL")
	uploadCmd.Flags().BoolVar(&flagNoOIDC, "no-oidc", false, "Skip OIDC token auto-detection")
	uploadCmd.Flags().StringVar(&flagPackage, "package", "", "Package name (for monorepo uploads)")
	uploadCmd.Flags().StringVar(&flagConfig, "config", "", "Path to .testlens.yml config file")
	uploadCmd.Flags().StringVar(&flagFormat, "format", "", "Coverage format: lcov, go, cobertura, jacoco (auto-detected if omitted)")
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

	// Load config if present
	cfg, err := config.Load(".", flagConfig)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// If --package is set, validate it against config (if config exists)
	if flagPackage != "" && cfg != nil {
		if _, err := cfg.FindPackage(flagPackage); err != nil {
			return err
		}
	}

	// Multi-package mode: config exists, no --file, no --package
	if cfg != nil && len(cfg.Packages) > 0 && flagFile == "" && flagPackage == "" {
		return uploadAllPackages(cfg, apiURL, repoURL, commitSHA, branch)
	}

	// Single upload mode
	filePath := flagFile
	if filePath == "" {
		searchDir := "."
		// If --package and config, scope discovery to the package path
		if flagPackage != "" && cfg != nil {
			pkg, _ := cfg.FindPackage(flagPackage)
			if pkg != nil {
				searchDir = pkg.Path
			}
		}
		found := discover.FindCoverageFile(searchDir)
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

	// Auto-detect format if not explicitly set
	format := flagFormat
	if format == "" {
		format = discover.DetectFormat(filePath)
		if format != "" {
			fmt.Fprintf(os.Stderr, "Auto-detected format: %s\n", format)
		}
	}

	fmt.Fprintf(os.Stderr, "Uploading %s for %s @ %s...\n", filePath, repoURL, commitSHA[:8])

	result, err := upload.Upload(upload.Options{
		APIURL:      apiURL,
		FilePath:    filePath,
		RepoURL:     repoURL,
		CommitSHA:   commitSHA,
		Branch:      branch,
		PackageName: flagPackage,
		Format:      format,
		NoOIDC:      flagNoOIDC,
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Upload accepted. ID: %s, status: %s\n", result.UploadID, result.Status)
	return nil
}

func uploadAllPackages(cfg *config.Config, apiURL, repoURL, commitSHA, branch string) error {
	var failed []string

	for _, pkg := range cfg.Packages {
		filePath := discover.FindCoverageFileInDir(pkg.Path)
		if filePath == "" {
			fmt.Fprintf(os.Stderr, "No coverage file found for package %s (path: %s), skipping\n", pkg.Name, pkg.Path)
			continue
		}

		// Auto-detect format per package if not explicitly set
		pkgFormat := flagFormat
		if pkgFormat == "" {
			pkgFormat = discover.DetectFormat(filePath)
			if pkgFormat != "" {
				fmt.Fprintf(os.Stderr, "Auto-detected format for %s: %s\n", pkg.Name, pkgFormat)
			}
		}

		fmt.Fprintf(os.Stderr, "Uploading %s for package %s @ %s...\n", filePath, pkg.Name, commitSHA[:8])

		result, err := upload.Upload(upload.Options{
			APIURL:      apiURL,
			FilePath:    filePath,
			RepoURL:     repoURL,
			CommitSHA:   commitSHA,
			Branch:      branch,
			PackageName: pkg.Name,
			Format:      pkgFormat,
			NoOIDC:      flagNoOIDC,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Upload failed for package %s: %v\n", pkg.Name, err)
			failed = append(failed, pkg.Name)
			continue
		}

		fmt.Fprintf(os.Stderr, "Package %s uploaded. ID: %s, status: %s\n", pkg.Name, result.UploadID, result.Status)
	}

	if len(failed) > 0 {
		return fmt.Errorf("upload failed for packages: %v", failed)
	}
	return nil
}
