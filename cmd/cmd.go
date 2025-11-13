package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/davidpimentel/calendar-sync/auth"
	"github.com/davidpimentel/calendar-sync/sync"
	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "calendar-sync",
		Short: "Copy event blocks from one Google calendar to another",
		Long:  `A CLI tool to sync events from a source Google Calendar to a destination Google Calendar in order to accurately reflect your availability.`,
	}

	// loginCmd represents the login command
	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to Google Calendar",
		Long:  `Login to Google Calendar and save authentication token.`,
	}

	// loginSourceCmd represents the login source command
	loginSourceCmd = &cobra.Command{
		Use:   "source",
		Short: "Login to source Google Calendar",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authenticating source calendar account...")
			auth.GetSourceTokenFromWeb()
		},
	}

	// loginDestinationCmd represents the login destination command
	loginDestinationCmd = &cobra.Command{
		Use:   "destination",
		Short: "Login to destination Google Calendar",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authenticating destination calendar account...")
			auth.GetDestinationTokenFromWeb()
		},
	}

	// runCmd represents the run command
	runCmd = &cobra.Command{
		Use:   "run",
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

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(loginCmd)
	loginCmd.AddCommand(loginSourceCmd)
	loginCmd.AddCommand(loginDestinationCmd)
	RootCmd.AddCommand(runCmd)
}
