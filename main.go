package main

import (
	"log"
	"os"

	"github.com/infobloxopen/auto-semver-tag/pkg/git"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "auto-semver-tag",
	}
	rootCmd.SetOut(os.Stdout)

	rootCmd.AddCommand(command())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func command() *cobra.Command {
	return &cobra.Command{
		Use:  "exec [REPOSITORY] [RELEASE_BRANCH] [COMMIT_SHA] [GH_EVENT_PATH]",
		Args: cobra.ExactArgs(4),
		Run:  executeCommand,
	}
}

func executeCommand(cmd *cobra.Command, args []string) {
	repository := args[0]
	releaseBranch := args[1]
	commitSha := args[2]
	githubEventFilePath := args[3]

	token, isExists := os.LookupEnv("GITHUB_TOKEN")
	if !isExists {
		log.Fatal("GITHUB_TOKEN env var does not exist")
	}

	client := git.New(token, repository, releaseBranch)
	client.PerformAction(commitSha, githubEventFilePath)
}
