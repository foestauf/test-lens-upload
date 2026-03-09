package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "go coverprofile",
			content:  "mode: set\ngithub.com/foo/bar/main.go:10.33,12.2 1 1\n",
			expected: "go",
		},
		{
			name:     "go coverprofile atomic mode",
			content:  "mode: atomic\ngithub.com/foo/bar/main.go:10.33,12.2 1 1\n",
			expected: "go",
		},
		{
			name:     "lcov with TN prefix",
			content:  "TN:\nSF:src/index.ts\nDA:1,1\nend_of_record\n",
			expected: "lcov",
		},
		{
			name:     "lcov with SF prefix",
			content:  "SF:src/index.ts\nDA:1,1\nend_of_record\n",
			expected: "lcov",
		},
		{
			name:     "cobertura xml",
			content:  "<?xml version=\"1.0\" ?>\n<coverage version=\"5.5\">\n</coverage>\n",
			expected: "cobertura",
		},
		{
			name:     "jacoco xml",
			content:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE report>\n<report name=\"test\">\n</report>\n",
			expected: "jacoco",
		},
		{
			name:     "unknown format",
			content:  "some random content\nthat we don't recognise\n",
			expected: "",
		},
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "coverage")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("writing temp file: %v", err)
			}

			got := DetectFormat(path)
			if got != tt.expected {
				t.Errorf("DetectFormat() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectFormat_nonexistentFile(t *testing.T) {
	got := DetectFormat("/tmp/this-file-does-not-exist-at-all")
	if got != "" {
		t.Errorf("DetectFormat() = %q, want empty string for missing file", got)
	}
}
