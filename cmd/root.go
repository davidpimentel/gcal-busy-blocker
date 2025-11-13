package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:   "calendar-sync",
		Short: "Copy event blocks from one Google calendar to another",
		Long:  `A CLI tool to sync events from a source Google Calendar to a destination Google Calendar in order to accurately reflect your availability.`,
	}
)

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
}
