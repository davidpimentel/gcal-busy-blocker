package cmd

import (
	"context"
	"log"

	"github.com/davidpimentel/gcal-busy-blocker/internal/auth"
	"github.com/davidpimentel/gcal-busy-blocker/internal/sync"
	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
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
			// Get source client
			sourceClient, err := auth.SourceClient()
			if err != nil {
				log.Fatalf("Unable to get source client: %v", err)
			}

			// Get destination client
			destClient, err := auth.DestinationClient()
			if err != nil {
				log.Fatalf("Unable to get destination client: %v", err)
			}

			// Create calendar service for source and destination
			sourceSrv, err := calendar.NewService(context.Background(), option.WithHTTPClient(sourceClient))
			if err != nil {
				log.Fatalf("Unable to retrieve source Calendar client: %v", err)
			}

			destSrv, err := calendar.NewService(context.Background(), option.WithHTTPClient(destClient))
			if err != nil {
				log.Fatalf("Unable to retrieve destination Calendar client: %v", err)
			}

			syncClient := &sync.SyncClient{
				SourceCalendarService:      sourceSrv,
				DestinationCalendarService: destSrv,
				DaysAhead:                  daysAhead,
				DryRun:                     dryRun,
			}

			syncClient.RunSync()
		},
	}
)

func init() {
	runCmd.Flags().Bool("dry-run", false, "Print out the created events instead of writing them to the destination calendar")
	runCmd.Flags().IntP("days-ahead", "d", 30, "Specify how many days into the future to sync")
	RootCmd.AddCommand(runCmd)
}
