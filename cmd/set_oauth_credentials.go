package cmd

import (
	"log"

	"github.com/davidpimentel/gcal-busy-blocker/internal/auth"
	"github.com/spf13/cobra"
)

var setOauthCredentialsCmd = &cobra.Command{
	Use:   "set-oauth-credentials",
	Short: "Set the Oauth 2.0 Client ID json file generated from the GCP console",
	Run: func(cmd *cobra.Command, args []string) {
		credentialsPath, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatalf("Path must be provided")
		}

		err = auth.CopyCredentialsFile(credentialsPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	setOauthCredentialsCmd.Flags().StringP("path", "p", "", "Path to the credentials json file")
	RootCmd.AddCommand(setOauthCredentialsCmd)
}
