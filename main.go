package main

import (
	"os"

	"github.com/foestauf/test-lens-upload/cmd"
)

// version is set by goreleaser at build time.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
