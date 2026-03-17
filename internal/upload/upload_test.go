package upload

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeTempCoverage(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}

func TestUpload_success(t *testing.T) {
	coveragePath := writeTempCoverage(t, "mode: set\nmain.go:1.1,2.2 1 1\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/uploads" {
			t.Errorf("expected /uploads, got %s", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			t.Fatalf("parsing content-type: %v", err)
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		form, err := mr.ReadForm(10 << 20)
		if err != nil {
			t.Fatalf("reading multipart form: %v", err)
		}

		// Verify metadata fields
		if got := form.Value["repoUrl"][0]; got != "https://github.com/test/repo" {
			t.Errorf("repoUrl = %q, want %q", got, "https://github.com/test/repo")
		}
		if got := form.Value["commitSha"][0]; got != "abc12345" {
			t.Errorf("commitSha = %q, want %q", got, "abc12345")
		}
		if got := form.Value["branch"][0]; got != "main" {
			t.Errorf("branch = %q, want %q", got, "main")
		}
		if got := form.Value["packageName"][0]; got != "core" {
			t.Errorf("packageName = %q, want %q", got, "core")
		}
		if got := form.Value["format"][0]; got != "go" {
			t.Errorf("format = %q, want %q", got, "go")
		}

		// Verify file was included
		if len(form.File["file"]) != 1 {
			t.Fatalf("expected 1 file, got %d", len(form.File["file"]))
		}
		fh := form.File["file"][0]
		if fh.Filename != "coverage.out" {
			t.Errorf("filename = %q, want %q", fh.Filename, "coverage.out")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{UploadID: "upload-123", Status: "accepted"})
	}))
	defer srv.Close()

	result, err := Upload(Options{
		APIURL:      srv.URL,
		FilePath:    coveragePath,
		RepoURL:     "https://github.com/test/repo",
		CommitSHA:   "abc12345",
		Branch:      "main",
		PackageName: "core",
		Format:      "go",
		NoOIDC:      true,
	})
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if result.UploadID != "upload-123" {
		t.Errorf("UploadID = %q, want %q", result.UploadID, "upload-123")
	}
	if result.Status != "accepted" {
		t.Errorf("Status = %q, want %q", result.Status, "accepted")
	}
}

func TestUpload_omitsOptionalFieldsWhenEmpty(t *testing.T) {
	coveragePath := writeTempCoverage(t, "some coverage data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		_, params, _ := mime.ParseMediaType(ct)
		mr := multipart.NewReader(r.Body, params["boundary"])
		form, err := mr.ReadForm(10 << 20)
		if err != nil {
			t.Fatalf("reading form: %v", err)
		}

		if _, ok := form.Value["branch"]; ok {
			t.Error("branch field should be omitted when empty")
		}
		if _, ok := form.Value["packageName"]; ok {
			t.Error("packageName field should be omitted when empty")
		}
		if _, ok := form.Value["format"]; ok {
			t.Error("format field should be omitted when empty")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{UploadID: "id", Status: "ok"})
	}))
	defer srv.Close()

	_, err := Upload(Options{
		APIURL:    srv.URL,
		FilePath:  coveragePath,
		RepoURL:   "https://github.com/test/repo",
		CommitSHA: "abc12345",
		NoOIDC:    true,
	})
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
}

func TestUpload_fileNotFound(t *testing.T) {
	_, err := Upload(Options{
		APIURL:   "http://localhost",
		FilePath: "/tmp/does-not-exist-at-all.coverage",
		NoOIDC:   true,
	})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestUpload_serverError(t *testing.T) {
	coveragePath := writeTempCoverage(t, "data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "internal server error")
	}))
	defer srv.Close()

	_, err := Upload(Options{
		APIURL:    srv.URL,
		FilePath:  coveragePath,
		RepoURL:   "https://github.com/test/repo",
		CommitSHA: "abc",
		NoOIDC:    true,
	})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestUpload_badJSON(t *testing.T) {
	coveragePath := writeTempCoverage(t, "data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "not json at all")
	}))
	defer srv.Close()

	_, err := Upload(Options{
		APIURL:    srv.URL,
		FilePath:  coveragePath,
		RepoURL:   "https://github.com/test/repo",
		CommitSHA: "abc",
		NoOIDC:    true,
	})
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestUpload_withOIDCToken(t *testing.T) {
	coveragePath := writeTempCoverage(t, "data")

	// OIDC token server
	oidcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer request-token-123" {
			t.Errorf("OIDC request Authorization = %q, want %q", got, "Bearer request-token-123")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"value": "oidc-jwt-token"})
	}))
	defer oidcSrv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", oidcSrv.URL+"?dummy=1")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token-123")

	// Upload API server
	var gotAuth string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{UploadID: "id", Status: "ok"})
	}))
	defer apiSrv.Close()

	_, err := Upload(Options{
		APIURL:    apiSrv.URL,
		FilePath:  coveragePath,
		RepoURL:   "https://github.com/test/repo",
		CommitSHA: "abc",
		NoOIDC:    false,
	})
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if gotAuth != "Bearer oidc-jwt-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer oidc-jwt-token")
	}
}

func TestUpload_oidcEnvNotSet(t *testing.T) {
	coveragePath := writeTempCoverage(t, "data")

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{UploadID: "id", Status: "ok"})
	}))
	defer srv.Close()

	_, err := Upload(Options{
		APIURL:    srv.URL,
		FilePath:  coveragePath,
		RepoURL:   "https://github.com/test/repo",
		CommitSHA: "abc",
		NoOIDC:    false,
	})
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if gotAuth != "" {
		t.Errorf("expected no Authorization header when OIDC env not set, got %q", gotAuth)
	}
}

func TestUpload_oidcServerFailsContinuesWithout(t *testing.T) {
	coveragePath := writeTempCoverage(t, "data")

	oidcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer oidcSrv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", oidcSrv.URL+"?dummy=1")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "token")

	var gotAuth string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{UploadID: "id", Status: "ok"})
	}))
	defer apiSrv.Close()

	_, err := Upload(Options{
		APIURL:    apiSrv.URL,
		FilePath:  coveragePath,
		RepoURL:   "https://github.com/test/repo",
		CommitSHA: "abc",
		NoOIDC:    false,
	})
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if gotAuth != "" {
		t.Errorf("expected no Authorization after OIDC failure, got %q", gotAuth)
	}
}

func TestFetchOIDCToken_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("audience") != "test-lens" {
			t.Errorf("expected audience=test-lens query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"value": "my-token"})
	}))
	defer srv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL+"?dummy=1")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "req-token")

	token, err := fetchOIDCToken()
	if err != nil {
		t.Fatalf("fetchOIDCToken() error: %v", err)
	}
	if token != "my-token" {
		t.Errorf("token = %q, want %q", token, "my-token")
	}
}

func TestFetchOIDCToken_envNotSet(t *testing.T) {
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")

	token, err := fetchOIDCToken()
	if err != nil {
		t.Fatalf("fetchOIDCToken() error: %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestFetchOIDCToken_partialEnv(t *testing.T) {
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "http://something")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")

	token, err := fetchOIDCToken()
	if err != nil {
		t.Fatalf("fetchOIDCToken() error: %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token when only URL set, got %q", token)
	}
}
