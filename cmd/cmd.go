package cmd

import (
	"fmt"
	"os"

	"github.com/davidpimentel/calendar-sync/auth"
	"github.com/davidpimentel/calendar-sync/config"
	"github.com/davidpimentel/calendar-sync/sync"
	"github.com/spf13/cobra"
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
			auth.GetTokenFromWeb(config.SourceTokenFile, config.SourceScope)
		},
	}

	// loginDestinationCmd represents the login destination command
	loginDestinationCmd = &cobra.Command{
		Use:   "destination",
		Short: "Login to destination Google Calendar",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authenticating destination calendar account...")
			auth.GetTokenFromWeb(config.DestTokenFile, config.DestinationScope)
		},
	}

	// runCmd represents the run command
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the calendar sync",
		Run: func(cmd *cobra.Command, args []string) {
			sync.RunSync()
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
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
