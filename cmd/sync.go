package cmd

import (
	"context"
	"log"

	"github.com/davidpimentel/gcal-busy-blocker/auth"
	"github.com/davidpimentel/gcal-busy-blocker/sync"
	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	runCmd = &cobra.Command{
		Use:   "sync",
		Short: "Run the calendar sync",
		Run: func(cmd *cobra.Command, args []string) {
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
				DaysAhead:                  1,
				DryRun:                     false,
			}

			syncClient.RunSync()
		},
	}
)

func init() {
	RootCmd.AddCommand(runCmd)
}
