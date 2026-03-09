package cmd

import (
	"github.com/spf13/cobra"
)

var appVersion = "dev"

func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "test-lens",
	Short: "Upload coverage reports to test-lens",
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func Execute() error {
	rootCmd.Version = appVersion
	return rootCmd.Execute()
}
