package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func NewRootCommand(subcommands []*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeforkctl",
		Short: "CLI to create Virtual Cluster based on an identifier",
		Long: `kubeforkctl command offers an experience that you have your own cluster.
kubefork only copy target services, deployments, pods, and containers but the container image is what you specify.
When your in example-microservice, requests to example-microservie service with header below will be sent to copied resources.

  x-wantedly-propagated-context-routing-any-fork-identifier: <identifier>

This command also expands environmental variables in the containers whose images are switched.
`,
	}

	for _, c := range subcommands {
		cmd.AddCommand(c)
	}

	return cmd
}

func subcommands() []*cobra.Command {
	subCmd := []*cobra.Command{
		// add subcommands below
		NewManifestCmd(),
	}

	return subCmd
}

func Execute() {
	subCmd := subcommands()
	rootCmd := NewRootCommand(subCmd)
	err := rootCmd.Execute()
	if err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}
}
