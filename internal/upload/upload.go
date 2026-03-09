package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type Options struct {
	APIURL    string
	FilePath  string
	RepoURL   string
	CommitSHA string
	Branch    string
	NoOIDC    bool
}

type Result struct {
	UploadID string `json:"uploadId"`
	Status   string `json:"status"`
}

func Upload(opts Options) (*Result, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add coverage file
	f, err := os.Open(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(opts.FilePath))
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copying file data: %w", err)
	}

	// Add metadata fields
	writer.WriteField("repoUrl", opts.RepoURL)
	writer.WriteField("commitSha", opts.CommitSHA)
	if opts.Branch != "" {
		writer.WriteField("branch", opts.Branch)
	}

	writer.Close()

	req, err := http.NewRequest("POST", opts.APIURL+"/uploads", body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// OIDC auth
	if !opts.NoOIDC {
		token, err := fetchOIDCToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "OIDC token request failed: %v, continuing without OIDC\n", err)
		} else if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			fmt.Fprintln(os.Stderr, "Using OIDC token for authentication")
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

func fetchOIDCToken() (string, error) {
	requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	requestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")

	if requestURL == "" || requestToken == "" {
		return "", nil
	}

	req, err := http.NewRequest("GET", requestURL+"&audience=test-lens", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC request returned %d", resp.StatusCode)
	}

	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	return body.Value, nil
}
