package cmd

import (
	"log"

	"github.com/davidpimentel/gcal-busy-blocker/internal/sync"
	"github.com/spf13/cobra"
)

var (
	runCmd = &cobra.Command{
		Use:   "sync",
		Short: "Run the calendar sync",
		Run: func(cmd *cobra.Command, args []string) {
			daysAhead, err := cmd.Flags().GetInt("days-ahead")
			if err != nil {
				log.Fatalf("Error parsing arg days-ahead: %v", err)
			}
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				log.Fatalf("Error parsing arg dry-run: %v", err)
			}
			syncClient := sync.NewSyncClient()
			err = syncClient.RunSync(daysAhead, dryRun)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	runCmd.Flags().Bool("dry-run", false, "Print out the created events instead of writing them to the destination calendar")
	runCmd.Flags().IntP("days-ahead", "d", 30, "Specify how many days into the future to sync")
	RootCmd.AddCommand(runCmd)
}
