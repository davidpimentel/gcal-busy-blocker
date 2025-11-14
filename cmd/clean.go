/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"

	"github.com/davidpimentel/gcal-busy-blocker/internal/sync"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all generated events from the destination calendar",
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Fatalf("Error parsing arg dry-run: %v", err)
		}
		syncClient := sync.NewSyncClient()
		syncClient.Clean(dryRun)
	},
}

func init() {
	cleanCmd.Flags().Bool("dry-run", false, "Print out the created events instead of writing them to the destination calendar")
	RootCmd.AddCommand(cleanCmd)
}
