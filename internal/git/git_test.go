package git

import "testing"

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:user/repo.git", "https://github.com/user/repo"},
		{"https://github.com/user/repo.git", "https://github.com/user/repo"},
		{"https://github.com/user/repo", "https://github.com/user/repo"},
		{"https://github.com/user/repo/", "https://github.com/user/repo"},
		{"  https://github.com/user/repo.git  ", "https://github.com/user/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeRepoURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
