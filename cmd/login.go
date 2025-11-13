package cmd

import (
	"fmt"

	"github.com/davidpimentel/calendar-sync/auth"
	"github.com/spf13/cobra"
)

var (
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
)

func init() {
	RootCmd.AddCommand(loginCmd)
	loginCmd.AddCommand(loginSourceCmd)
	loginCmd.AddCommand(loginDestinationCmd)
}
