package discover

import (
	"os"
	"path/filepath"
)

var candidatePaths = []string{
	"coverage/lcov.info",
	"coverage/coverage.lcov",
	"coverage/cobertura-coverage.xml",
	"coverage/clover.xml",
	"coverage/coverage-final.json",
	"build/reports/jacoco/test/jacocoTestReport.xml",
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
