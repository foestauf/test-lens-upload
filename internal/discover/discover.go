package discover

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var candidatePaths = []string{
	"coverage/lcov.info",
	"coverage/coverage.lcov",
	"coverage/cobertura-coverage.xml",
	"coverage/clover.xml",
	"coverage/coverage-final.json",
	"build/reports/jacoco/test/jacocoTestReport.xml",
	"coverage.out",
}

// FindCoverageFile looks for common coverage file paths relative to dir.
// Returns the absolute path of the first match, or empty string if none found.
func FindCoverageFile(dir string) string {
	return findInDir(dir)
}

// FindCoverageFileInDir searches for coverage files within a specific package
// directory. Used for monorepo per-package discovery.
func FindCoverageFileInDir(packageDir string) string {
	return findInDir(packageDir)
}

func findInDir(dir string) string {
	for _, candidate := range candidatePaths {
		full := filepath.Join(dir, candidate)
		if _, err := os.Stat(full); err == nil {
			abs, err := filepath.Abs(full)
			if err != nil {
				return full
			}
			return abs
		}
	}
	return ""
}

// DetectFormat peeks at the first line of a coverage file to determine its format.
// Returns one of: "lcov", "go", "cobertura", "jacoco", or empty string if unknown.
func DetectFormat(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}
	firstLine := strings.TrimSpace(scanner.Text())

	switch {
	case strings.HasPrefix(firstLine, "mode:"):
		return "go"
	case strings.HasPrefix(firstLine, "TN:") || strings.HasPrefix(firstLine, "SF:"):
		return "lcov"
	case strings.HasPrefix(firstLine, "<?xml"):
		// Peek further to distinguish cobertura vs jacoco
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.Contains(line, "<coverage") {
				return "cobertura"
			}
			if strings.Contains(line, "<report") {
				return "jacoco"
			}
		}
		return "cobertura"
	}

	return ""
}
