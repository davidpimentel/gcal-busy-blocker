package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	appName                   = "calendar-sync"
	credentialsFile           = "credentials.json"
	sourceTokenFile           = "source_token.json"
	destTokenFile             = "destination_token.json"
	sourceEventIdPropertyKey  = "calendar-sync-source-event-id"
	sourceCalendarPropertyKey = "calendar-sync-source-calendar-id"
)

// Google Calendar permission scopes
var (
	sourceScope      = []string{calendar.CalendarReadonlyScope, calendar.CalendarEventsReadonlyScope}
	destinationScope = []string{calendar.CalendarScope}
)

// Command structure
var (
	rootCmd = &cobra.Command{
		Use:   "calendar-sync",
		Short: "Copy event blocks from one Google calendar to another",
		Long:  `A CLI tool to sync events from a source Google Calendar to a destination Google Calendar in order to accurately reflect your availability.`,
	}

	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to Google Calendar",
		Long:  `Login to Google Calendar and save authentication token.`,
	}

	loginSourceCmd = &cobra.Command{
		Use:   "source",
		Short: "Login to source Google Calendar",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authenticating source calendar account...")
			getTokenFromWeb(sourceTokenFile, sourceScope)
		},
	}

	loginDestinationCmd = &cobra.Command{
		Use:   "destination",
		Short: "Login to destination Google Calendar",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authenticating destination calendar account...")
			getTokenFromWeb(destTokenFile, destinationScope)
		},
	}

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the calendar sync",
		Run: func(cmd *cobra.Command, args []string) {
			runSync()
		},
	}
)

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.AddCommand(loginSourceCmd)
	loginCmd.AddCommand(loginDestinationCmd)
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getOauthConfig(scope []string) *oauth2.Config {
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf("Unable to read credentials.json: %v", err)
	}

	config, err := google.ConfigFromJSON(b, scope...)
	if err != nil {
		log.Fatalf("unable to parse client secret file to config: %v", err)
	}
	return config
}

func getClient(tokenFile string, scope []string) (*http.Client, error) {
	config := getOauthConfig(scope)

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("token not found, please run 'login' command first: %v", err)
	}

	return config.Client(context.Background(), tok), nil
}

func getTokenFromWeb(tokenFile string, scope []string) {
	config := getOauthConfig(scope)

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)
	fmt.Println("Enter the authorization code:")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	saveToken(tokenFile, tok)
	fmt.Printf("Authentication successful! Token saved to %s\n", tokenFile)
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func runSync() {
	fmt.Println("Starting calendar sync...")

	// Get source client
	sourceClient, err := getClient(sourceTokenFile, sourceScope)
	if err != nil {
		log.Fatalf("Unable to get source client: %v", err)
	}

	// Get destination client
	destClient, err := getClient(destTokenFile, destinationScope)
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

	// Get events from source calendar
	now := time.Now().Format(time.RFC3339)
	// oneMonthFromNow := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	oneDayFromNow := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)

	fmt.Println("Fetching events from source calendar...")
	fmt.Printf("Time range: %s to %s\n", now, oneDayFromNow)

	// Get primary calendar for source
	sourceCalendar, err := sourceSrv.Calendars.Get("primary").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve source calendar: %v", err)
	}

	// List events from source calendar
	events, err := sourceSrv.Events.List("primary").
		TimeMin(now).
		TimeMax(oneDayFromNow).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve events from source calendar: %v", err)
	}

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found in source calendar.")
		return
	}

	fmt.Printf("Found %d events in source calendar\n", len(events.Items))

	// Get any and all events from the destination calendar that we've created
	destinationEvents, err := destSrv.Events.List("primary").
		TimeMin(now).
		TimeMax(oneDayFromNow).
		PrivateExtendedProperty(fmt.Sprintf("%s=%s", appName, "true")).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		log.Fatalf("Unable to retrieve existing events from destination calendar: %v", err)
	}

	// Process each event
	for _, event := range events.Items {
		fmt.Printf("Event: %s (%s)\n", event.Summary, event.Id)

		if eventAlreadyExists(destinationEvents.Items, event.Id) {
			fmt.Printf("Event already synced: %s\n", event.Id)
		} else {
			// Create a new event in the destination calendar
			newEvent := &calendar.Event{
				ColorId:     "4",
				Summary:     "Busy",
				Description: "Created with calendar-sync",
				Start:       event.Start,
				End:         event.End,
				// Add extended properties to track the source event
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						appName:                   "true",
						sourceEventIdPropertyKey:  event.Id,
						sourceCalendarPropertyKey: sourceCalendar.Id,
					},
				},
			}

			// Insert the event
			fmt.Printf("Creating new event: %s\n", newEvent.Summary)
			b, err := json.MarshalIndent(newEvent, "", "  ")
			if err != nil {
				fmt.Println(err)
			}
			fmt.Print(string(b))

			_, err = destSrv.Events.Insert("primary", newEvent).Do()
			if err != nil {
				log.Printf("Error creating event: %v", err)
				continue
			}
		}
	}

	fmt.Println("Sync completed successfully")
}

func eventAlreadyExists(destinationEvents []*calendar.Event, sourceEventID string) bool {
	for _, event := range destinationEvents {
		if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
			if event.ExtendedProperties.Private[sourceEventIdPropertyKey] == sourceEventID {
				return true
			}
		}
	}
	return false
}
