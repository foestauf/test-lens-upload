package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/foestauf/test-lens-upload/internal/upload"
)

// newUploadServer returns a test server that accepts uploads and responds with
// a valid Result JSON.
func newUploadServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(upload.Result{UploadID: "test-id", Status: "accepted"})
	}))
}

func resetFlags() {
	flagFile = ""
	flagRepoURL = ""
	flagCommitSHA = ""
	flagBranch = ""
	flagAPIURL = ""
	flagNoOIDC = false
	flagPackage = ""
	flagConfig = ""
	flagFormat = ""
}

func TestRunUpload_singleFileExplicit(t *testing.T) {
	srv := newUploadServer(t)
	defer srv.Close()

	dir := t.TempDir()
	coveragePath := filepath.Join(dir, "coverage.out")
	os.WriteFile(coveragePath, []byte("mode: set\nmain.go:1.1,2.2 1 1\n"), 0644)

	resetFlags()
	flagFile = coveragePath
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = srv.URL
	flagNoOIDC = true
	flagFormat = "go"

	if err := runUpload(nil, nil); err != nil {
		t.Fatalf("runUpload() error: %v", err)
	}
}

func TestRunUpload_fileNotFound(t *testing.T) {
	resetFlags()
	flagFile = "/tmp/absolutely-does-not-exist.lcov"
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = "http://localhost"
	flagNoOIDC = true

	err := runUpload(nil, nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunUpload_apiURLFromEnv(t *testing.T) {
	srv := newUploadServer(t)
	defer srv.Close()

	dir := t.TempDir()
	coveragePath := filepath.Join(dir, "coverage.out")
	os.WriteFile(coveragePath, []byte("mode: set\nmain.go:1.1,2.2 1 1\n"), 0644)

	t.Setenv("TEST_LENS_API_URL", srv.URL)

	resetFlags()
	flagFile = coveragePath
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagNoOIDC = true

	if err := runUpload(nil, nil); err != nil {
		t.Fatalf("runUpload() error: %v", err)
	}
}

func TestRunUpload_autoDetectFormat(t *testing.T) {
	srv := newUploadServer(t)
	defer srv.Close()

	dir := t.TempDir()
	coveragePath := filepath.Join(dir, "coverage.out")
	os.WriteFile(coveragePath, []byte("mode: set\nmain.go:1.1,2.2 1 1\n"), 0644)

	resetFlags()
	flagFile = coveragePath
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = srv.URL
	flagNoOIDC = true
	// flagFormat intentionally left empty for auto-detection

	if err := runUpload(nil, nil); err != nil {
		t.Fatalf("runUpload() error: %v", err)
	}
}

func TestRunUpload_configWithPackageFlag(t *testing.T) {
	srv := newUploadServer(t)
	defer srv.Close()

	dir := t.TempDir()

	// Create config with relative paths
	configContent := `packages:
  - name: api
    path: apps/api
  - name: web
    path: apps/web
`
	configPath := filepath.Join(dir, ".testlens.yml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Create coverage file in package path
	apiDir := filepath.Join(dir, "apps", "api")
	os.MkdirAll(apiDir, 0755)
	os.WriteFile(filepath.Join(apiDir, "coverage.out"), []byte("mode: set\nmain.go:1.1,2.2 1 1\n"), 0644)

	// chdir so relative package path resolves correctly
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	resetFlags()
	flagConfig = configPath
	flagPackage = "api"
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = srv.URL
	flagNoOIDC = true

	if err := runUpload(nil, nil); err != nil {
		t.Fatalf("runUpload() error: %v", err)
	}
}

func TestRunUpload_configInvalidPackage(t *testing.T) {
	dir := t.TempDir()

	configContent := `packages:
  - name: api
    path: apps/api
`
	configPath := filepath.Join(dir, ".testlens.yml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	resetFlags()
	flagConfig = configPath
	flagPackage = "nonexistent"
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = "http://localhost"
	flagNoOIDC = true

	err := runUpload(nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid package name")
	}
}

func TestRunUpload_multiPackageMode(t *testing.T) {
	srv := newUploadServer(t)
	defer srv.Close()

	dir := t.TempDir()

	// Create config with two packages
	configContent := `packages:
  - name: api
    path: api
  - name: web
    path: web
`
	configPath := filepath.Join(dir, ".testlens.yml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Create coverage files for both packages
	for _, pkg := range []string{"api", "web"} {
		pkgDir := filepath.Join(dir, pkg)
		os.MkdirAll(pkgDir, 0755)
		os.WriteFile(filepath.Join(pkgDir, "coverage.out"), []byte("mode: set\nmain.go:1.1,2.2 1 1\n"), 0644)
	}

	// Need to chdir so config.Load(".", ...) and discover work from the right place
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	resetFlags()
	flagConfig = configPath
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = srv.URL
	flagNoOIDC = true
	// No --file and no --package triggers multi-package mode

	if err := runUpload(nil, nil); err != nil {
		t.Fatalf("runUpload() error: %v", err)
	}
}

func TestRunUpload_multiPackagePartialFailure(t *testing.T) {
	// Server that fails every other request
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(upload.Result{UploadID: "id", Status: "ok"})
	}))
	defer srv.Close()

	dir := t.TempDir()

	configContent := `packages:
  - name: api
    path: api
  - name: web
    path: web
`
	configPath := filepath.Join(dir, ".testlens.yml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	for _, pkg := range []string{"api", "web"} {
		pkgDir := filepath.Join(dir, pkg)
		os.MkdirAll(pkgDir, 0755)
		os.WriteFile(filepath.Join(pkgDir, "coverage.out"), []byte("mode: set\nmain.go:1.1,2.2 1 1\n"), 0644)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	resetFlags()
	flagConfig = configPath
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = srv.URL
	flagNoOIDC = true

	err := runUpload(nil, nil)
	if err == nil {
		t.Fatal("expected error when some packages fail")
	}
}

func TestRunUpload_multiPackageNoCoverage(t *testing.T) {
	srv := newUploadServer(t)
	defer srv.Close()

	dir := t.TempDir()

	configContent := `packages:
  - name: api
    path: api
`
	configPath := filepath.Join(dir, ".testlens.yml")
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Create pkg dir but no coverage file
	os.MkdirAll(filepath.Join(dir, "api"), 0755)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	resetFlags()
	flagConfig = configPath
	flagRepoURL = "https://github.com/test/repo"
	flagCommitSHA = "abcdef1234567890"
	flagBranch = "main"
	flagAPIURL = srv.URL
	flagNoOIDC = true

	// Should not error — just skips packages with no coverage
	if err := runUpload(nil, nil); err != nil {
		t.Fatalf("runUpload() error: %v", err)
	}
}
